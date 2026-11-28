package main

import(
 "context"
 "errors"
 "fmt"
 "io"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "time"
)

var buildVersion="0.1.7"
// Set only for the never-published CI fixture. Production builds leave empty.
var fixtureMode=""
var managerSpec=appSpec{"routilet","Routilet","应用启动台","@routilet/routilet","latest"}
type SelfInfo struct{
 Version string `json:"version"`
 Available string `json:"available"`
 Update bool `json:"update"`
 Pending bool `json:"pending"`
 PendingVersion string `json:"pendingVersion"`
 ReleaseStatus string `json:"releaseStatus"`
 Note string `json:"note"`
}
func(m *Manager)selfLocal()SelfInfo{
 a:=SelfInfo{Version:buildVersion,ReleaseStatus:"unknown"}
 if v,e:=m.active(managerSpec);e==nil&&newer(v.Version,buildVersion){a.Pending=true;a.PendingVersion=v.Version}
 return a
}
func(m *Manager)selfInfo()SelfInfo{
 a:=m.selfLocal()
 v,e:=m.remote(managerSpec);if e!=nil{a.Note=e.Error();if errors.Is(e,errNoRelease){a.ReleaseStatus="unavailable"}else{a.ReleaseStatus="error"};return a}
 a.ReleaseStatus="available";a.Available=v;a.Update=newer(v,buildVersion)&&(!a.Pending||newer(v,a.PendingVersion))
 return a
}
func(m *Manager)setBootstrap(path string)error{
 if !filepath.IsAbs(path){return errors.New("无效启动入口")}
 st,e:=os.Stat(path);if e!=nil{return e};if !st.Mode().IsRegular(){return errors.New("启动入口不是文件")}
 m.bootstrap=path
 m.pnpm=bundledPNPM(path)
 return nil
}
func probeManager(ctx context.Context,entry,version string)error{
 ctx,cancel:=context.WithTimeout(ctx,15*time.Second);defer cancel()
 cmd:=exec.CommandContext(ctx,entry,"--manager-probe");hideChild(cmd)
 out,e:=cmd.CombinedOutput()
 if e!=nil{return fmt.Errorf("新版自检失败: %w %s",e,out)}
 if strings.TrimSpace(string(out))!="routilet-manager/1 "+version{return errors.New("新版管理器协议或版本不匹配")}
 return nil
}
// The bundled executable is both stable bootstrap and fallback GUI. It
// never overwrites itself or pnpm. Updated manager packages omit pnpm.
func(m *Manager)dispatch()bool{
 a,e:=m.active(managerSpec);if e!=nil{return false}
 done,e:=m.operationLock();if e!=nil{return false};defer done()
 // Re-read under the process-wide operation lock.
 a,e=m.active(managerSpec);if e!=nil{return false}
 if newer(a.Version,buildVersion)&&m.startManager(a.Snapshot,a.Version)==nil{return true}
 if a.Previous!=""&&newer(a.PreviousVersion,buildVersion)&&m.startManager(a.Previous,a.PreviousVersion)==nil{
  _=atomicJSON(filepath.Join(m.appDir(managerSpec),"activation.json"),activation{Snapshot:a.Previous,Version:a.PreviousVersion})
  return true
 }
 // Preserve failed snapshots for diagnosis; bundled GUI is always usable.
 _=os.Remove(filepath.Join(m.appDir(managerSpec),"activation.json"))
 return false
}
func(m *Manager)startManager(snapshot,version string)error{
 entry,e:=nativeEntry(filepath.Join(m.appDir(managerSpec),snapshot),managerSpec,version);if e!=nil{return e}
 if e=probeManager(context.Background(),entry,version);e!=nil{return e}
 dir,e:=os.MkdirTemp(m.root,"manager-ready-");if e!=nil{return e};defer os.RemoveAll(dir)
 ready:=filepath.Join(dir,"ready")
 cmd:=exec.Command(entry,"--managed",m.bootstrap,ready);cmd.Dir=filepath.Dir(entry)
 if e=cmd.Start();e!=nil{return e}
 // Do not attach pipes to the child: it must outlive bootstrap.
 // Poll process completion and explicit frontend readiness.
 finished:=make(chan error,1);go func(){finished<-cmd.Wait()}()
 deadline:=time.NewTimer(25*time.Second);defer deadline.Stop()
 tick:=time.NewTicker(100*time.Millisecond);defer tick.Stop()
 for{
  select{
  case e:=<-finished:return fmt.Errorf("新版提前退出: %v",e)
  case <-deadline.C:_=cmd.Process.Kill();<-finished;return errors.New("新版未完成启动")
  case <-tick.C:if b,e:=os.ReadFile(ready);e==nil&&string(b)=="ready"{return nil}
  }
 }
}
func selfUpdateTest(tarball,version string)error{
 m,e:=newManager();if e!=nil{return e}
 root,e:=os.MkdirTemp("","routilet self update 测试 ");if e!=nil{return e};defer os.RemoveAll(root);m.root=root
 if e=m.setup();e!=nil{return e}
 dir:=m.appDir(managerSpec);if e=os.MkdirAll(dir,0700);e!=nil{return e}
 snap,e:=os.MkdirTemp(dir,"release-");if e!=nil{return e}
 abs,e:=filepath.Abs(tarball);if e!=nil{return e}
 // Exercise the same relative local archive dependency used by the verified
 // mirror path, with the real bundled pnpm on each native CI runner.
 archive:=filepath.Join(snap,mirrorArchive)
 src,e:=os.Open(abs);if e!=nil{return e}
 dst,e:=os.OpenFile(archive,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600)
 if e!=nil{src.Close();return e}
 _,copyErr:=io.Copy(dst,src)
 closeErr:=dst.Close();src.Close()
 if copyErr!=nil{return copyErr};if closeErr!=nil{return closeErr}
 if e=m.installPackage(snap,managerSpec.Package,"file:./"+mirrorArchive);e!=nil{return e}
 if e=os.Remove(archive);e!=nil{return e}
 entry,e:=nativeEntry(snap,managerSpec,version);if e!=nil{return e}
 if e=probeManager(context.Background(),entry,version);e!=nil{return e}
 a:=activation{Snapshot:filepath.Base(snap),Version:version}
 if e=atomicJSON(filepath.Join(dir,"activation.json"),a);e!=nil{return e}
 if got,e:=m.active(managerSpec);e!=nil||got!=a{return fmt.Errorf("activation %v %v",got,e)}
 // Execute the installed native fixture through the real bootstrap dispatch
 // and readiness handshake. This tests the protocol, not WebView rendering.
 if !m.dispatch(){return errors.New("healthy manager did not start")}
 time.Sleep(1200*time.Millisecond)
 // Invalid candidate never prevents bundled GUI fallback. Remove only the
 // candidate entry, rather than writing through a possibly shared hardlink.
 if e=os.Remove(entry);e!=nil{return e}
 if m.dispatch(){return errors.New("damaged manager launched")}
 if _,e=m.active(managerSpec);!os.IsNotExist(e){return errors.New("failed activation not cleared")}
 fmt.Println("PASS: local npm manager install, manifest, protocol, activation, ready handshake, corrupt candidate fallback")
 return nil
}
