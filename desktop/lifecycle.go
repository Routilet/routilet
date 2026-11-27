package main

import (
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "time"
)

func (m *Manager) operationLock()(func(),error){
 if err:=os.MkdirAll(m.root,0700);err!=nil{return nil,err}
 f,err:=os.OpenFile(filepath.Join(m.root,"operations.lock"),os.O_CREATE|os.O_RDWR,0600);if err!=nil{return nil,err}
 if err=lockFile(f);err!=nil{f.Close();return nil,errors.New("另一 Routilet 实例正在操作，请稍后重试")}
 return func(){unlockFile(f);f.Close()},nil
}
type CleanupReport struct {
 Removed int `json:"removed"`
 Protected int `json:"protected"`
 Failed []string `json:"failed"`
}
func (m *Manager) cleanup(s appSpec)CleanupReport{
 result:=CleanupReport{}
 a,err:=m.active(s)
 if err!=nil&&!os.IsNotExist(err){result.Failed=append(result.Failed,s.Name+": "+err.Error());return result}
 dir:=m.appDir(s)
 entries,err:=os.ReadDir(dir);if os.IsNotExist(err){return result};if err!=nil{result.Failed=append(result.Failed,s.Name+": "+err.Error());return result}
 for _,e:=range entries{
  if !e.IsDir()||!validSnapshot(e.Name()){continue}
  if e.Name()==a.Snapshot||e.Name()==a.Previous{result.Protected++;continue}
  path:=filepath.Join(dir,e.Name())
  // Retain snapshots with running markers, even when stale or unreadable.
  runs,err:=os.ReadDir(filepath.Join(path,".runs"))
  if err!=nil&&!os.IsNotExist(err){result.Protected++;result.Failed=append(result.Failed,s.Name+": "+err.Error());continue}
  if len(runs)>0{result.Protected++;continue}
  if err=os.RemoveAll(path);err!=nil{result.Failed=append(result.Failed,s.Name+": "+err.Error())}else{result.Removed++}
 }
 return result
}
func (m *Manager) rollback(id string)error{return m.rollbackExpected(id,"")}
func (m *Manager) rollbackExpected(id,expected string)error{
 if id=="routilet"{return errors.New("不能通过应用管理修改 Routilet 自身")}
 m.mu.Lock();defer m.mu.Unlock()
 done,err:=m.operationLock();if err!=nil{return err};defer done()
 s,err:=spec(id);if err!=nil{return err}
 a,err:=m.active(s);if err!=nil{return err}
 if a.Previous==""{return errors.New("没有可回退版本")}
 if expected!=""&&expected!=a.PreviousVersion{return errors.New("回退目标已变化，请重新检查")}
 if _,err=nativeEntry(filepath.Join(m.appDir(s),a.Previous),s,a.PreviousVersion);err!=nil{return err}
 return atomicJSON(filepath.Join(m.appDir(s),"activation.json"),activation{Snapshot:a.Previous,Version:a.PreviousVersion,Previous:a.Snapshot,PreviousVersion:a.Version})
}
func (m *Manager) uninstall(id string)error{
 if id=="routilet"{return errors.New("不能通过应用管理修改 Routilet 自身")}
 m.mu.Lock();defer m.mu.Unlock()
 done,err:=m.operationLock();if err!=nil{return err};defer done()
 s,err:=spec(id);if err!=nil{return err}
 if err=os.Remove(filepath.Join(m.appDir(s),"activation.json"));err!=nil&&!os.IsNotExist(err){return err}
 result:=m.cleanup(s)
 if len(result.Failed)>0{return fmt.Errorf("应用已卸载，但部分文件未清理：%s",strings.Join(result.Failed,"; "))}
 return nil
}
func (m *Manager) clean()(CleanupReport,error){
 m.mu.Lock();defer m.mu.Unlock()
 done,err:=m.operationLock();if err!=nil{return CleanupReport{},err};defer done()
 var report CleanupReport
 for _,s:=range catalog{r:=m.cleanup(s);report.Removed+=r.Removed;report.Protected+=r.Protected;report.Failed=append(report.Failed,r.Failed...)}
 return report,nil
}
func (m *Manager) startProtected(s appSpec,a activation)error{
 snap:=filepath.Join(m.appDir(s),a.Snapshot)
 runs:=filepath.Join(snap,".runs");if err:=os.MkdirAll(runs,0700);err!=nil{return err}
 marker,err:=os.MkdirTemp(runs,"run-");if err!=nil{return err}
 exe,err:=os.Executable();if err!=nil{_ = os.RemoveAll(marker);return err}
 cmd:=exec.Command(exe,"--supervise",s.ID,a.Snapshot,a.Version,filepath.Base(marker))
 hideChild(cmd)
 if err=cmd.Start();err!=nil{_ = os.RemoveAll(marker);return err}
 _=cmd.Process.Release()
 // The helper acknowledges that the OS accepted the start. Do not remove
 // a marker on timeout: it could already protect a live process.
 until:=time.Now().Add(10*time.Second)
 for time.Now().Before(until){
  b,e:=os.ReadFile(filepath.Join(marker,"started"))
  if e==nil&&string(b)=="ok"{return nil}
  if b,e=os.ReadFile(filepath.Join(marker,"error"));e==nil{return errors.New(string(b))}
  time.Sleep(50*time.Millisecond)
 }
 return errors.New("启动未确认；为安全起见保留运行标记")
}
func (m *Manager) supervise(id,snapshot,version,run string)error{
 if !validSnapshot(snapshot)||!strings.HasPrefix(run,"run-")||filepath.Base(run)!=run{return errors.New("无效运行标记")}
 s,err:=spec(id);if err!=nil{return err}
 snap:=filepath.Join(m.appDir(s),snapshot);marker:=filepath.Join(snap,".runs",run)
 if st,e:=os.Stat(marker);e!=nil||!st.IsDir(){return errors.New("缺少运行标记")}
 fail:=func(e error)error{_ = os.WriteFile(filepath.Join(marker,"error"),[]byte(e.Error()),0600);return e}
 entry,err:=nativeEntry(snap,s,version);if err!=nil{return fail(err)}
 var cmd *exec.Cmd
 if runtime.GOOS=="darwin"{
  app:=entry
  for filepath.Ext(app)!=".app"{p:=filepath.Dir(app);if p==app{return fail(errors.New("macOS 应用包不存在"))};app=p}
  cmd=exec.Command("/usr/bin/open","-n","-W",app)
 }else{cmd=exec.Command(entry);cmd.Dir=filepath.Dir(entry)}
 if err=cmd.Start();err!=nil{return fail(err)}
 if err=os.WriteFile(filepath.Join(marker,"started"),[]byte("ok"),0600);err!=nil{_ = cmd.Wait();return err}
 err=cmd.Wait()
 // open -W failure may be ambiguous; retain marker. Direct child on
 // Windows has certainly exited, even if it returned a nonzero status.
 if err!=nil&&runtime.GOOS=="darwin"{return fail(err)}
 // Delay removal briefly so the launching GUI can observe a fast exit.
 time.Sleep(time.Second)
 return os.RemoveAll(marker)
}
func quoteShell(s string)string{return "'"+strings.ReplaceAll(s,"'","'\\''")+"'"}
func (m *Manager) shortcut(id string)error{
 if id=="routilet"{return errors.New("不能通过应用管理修改 Routilet 自身")}
 m.mu.Lock();defer m.mu.Unlock()
 done,err:=m.operationLock();if err!=nil{return err};defer done()
 s,err:=spec(id);if err!=nil{return err}
 if _,err=m.active(s);err!=nil{return err}
 // Shortcuts point to the stable original entry, never a versioned app.
 if runtime.GOOS=="windows"{
  cmd:=exec.Command("powershell.exe","-NoProfile","-NonInteractive","-Command",`$w=New-Object -ComObject WScript.Shell; $p=Join-Path ($w.SpecialFolders.Item('Desktop')) ($env:ROUTILET_SHORTCUT_NAME+'.lnk'); if(Test-Path -LiteralPath $p){throw 'Shortcut already exists'}; $s=$w.CreateShortcut($p); $s.TargetPath=$env:ROUTILET_BOOTSTRAP; $s.Arguments='--open-app '+$env:ROUTILET_APP_ID; $s.Save()`)
  hideChild(cmd)
  cmd.Env=append(os.Environ(),"ROUTILET_SHORTCUT_NAME="+s.Name,"ROUTILET_BOOTSTRAP="+m.bootstrap,"ROUTILET_APP_ID="+id)
  out,e:=cmd.CombinedOutput();if e!=nil{return fmt.Errorf("快捷方式: %w %s",e,out)};return nil
 }
 home,err:=os.UserHomeDir();if err!=nil{return err}
 // Do not overwrite any preexisting user-owned shortcut.
 path:=filepath.Join(home,"Desktop","Routilet - "+s.Name+".command")
 f,err:=os.OpenFile(path,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0700);if err!=nil{return err}
 _,err=f.WriteString("#!/bin/sh\nexec "+quoteShell(m.bootstrap)+" --open-app "+quoteShell(id)+"\n")
 if e:=f.Close();err==nil{err=e}
 if err!=nil{_ = os.Remove(path)}
 return err
}
