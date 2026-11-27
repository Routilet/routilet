package main

import (
 "context"
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
func (m *Manager) cleanup(s appSpec){
 a,err:=m.active(s)
 if err!=nil&&!os.IsNotExist(err){return}
 dir:=m.appDir(s)
 entries,err:=os.ReadDir(dir);if err!=nil{return}
 for _,e:=range entries{
  if !e.IsDir()||!validSnapshot(e.Name())||e.Name()==a.Snapshot||e.Name()==a.Previous{continue}
  path:=filepath.Join(dir,e.Name())
  // A marker is created BEFORE spawning the supervisor, and removed only
  // after a known process exit. Crashes leave markers and leak space rather
  // than risk deletion of a live application. No timestamp-based expiry.
  runs,err:=os.ReadDir(filepath.Join(path,".runs"))
  if err!=nil&&!os.IsNotExist(err){continue}
  if len(runs)>0{continue}
  _=os.RemoveAll(path)
 }
}
func (m *Manager) rollback(id string)error{
 if id=="routilet"{return errors.New("不能通过应用管理修改 Routilet 自身")}
 m.mu.Lock();defer m.mu.Unlock()
 done,err:=m.operationLock();if err!=nil{return err};defer done()
 s,err:=spec(id);if err!=nil{return err}
 a,err:=m.active(s);if err!=nil{return err}
 if a.Previous==""{return errors.New("没有可回退版本")}
 if _,err=nativeEntry(filepath.Join(m.appDir(s),a.Previous),s,a.PreviousVersion);err!=nil{return err}
 return atomicJSON(filepath.Join(m.appDir(s),"activation.json"),activation{Snapshot:a.Previous,Version:a.PreviousVersion,Previous:a.Snapshot,PreviousVersion:a.Version})
}
func (m *Manager) uninstall(id string)error{
 if id=="routilet"{return errors.New("不能通过应用管理修改 Routilet 自身")}
 m.mu.Lock();defer m.mu.Unlock()
 done,err:=m.operationLock();if err!=nil{return err};defer done()
 s,err:=spec(id);if err!=nil{return err}
 if err=os.Remove(filepath.Join(m.appDir(s),"activation.json"));err!=nil&&!os.IsNotExist(err){return err}
 m.cleanup(s)
 return nil
}
func (m *Manager) clean()error{
 m.mu.Lock();defer m.mu.Unlock()
 done,err:=m.operationLock();if err!=nil{return err};defer done()
 for _,s:=range catalog{m.cleanup(s)}
 return nil
}
func (m *Manager) startProtected(s appSpec,a activation)error{
 snap:=filepath.Join(m.appDir(s),a.Snapshot)
 runs:=filepath.Join(snap,".runs");if err:=os.MkdirAll(runs,0700);err!=nil{return err}
 marker,err:=os.MkdirTemp(runs,"run-");if err!=nil{return err}
 return m.spawnSupervisor(s,a,marker,nil)
}
func (m *Manager) supervise(id,snapshot,version,run string,args ...string)error{
 if !validSnapshot(snapshot)||!strings.HasPrefix(run,"run-")||filepath.Base(run)!=run{return errors.New("无效运行标记")}
 s,err:=spec(id);if err!=nil{return err}
 snap:=filepath.Join(m.appDir(s),snapshot);marker:=filepath.Join(snap,".runs",run)
 if st,e:=os.Stat(marker);e!=nil||!st.IsDir(){return errors.New("缺少运行标记")}
 fail:=func(e error)error{_ = os.WriteFile(filepath.Join(marker,"error"),[]byte(e.Error()),0600);return e}
 entry,err:=nativeEntry(snap,s,version);if err!=nil{return fail(err)}
 // Direct native launch preserves environment and exact argv on both
 // platforms, and gives a reliable process lifetime instead of open -W.
 cmd,err:=m.appCommand(context.Background(),entry,args);if err!=nil{return fail(err)}
 if err=cmd.Start();err!=nil{return fail(err)}
 if err=os.WriteFile(filepath.Join(marker,"started"),[]byte("ok"),0600);err!=nil{_ = cmd.Wait();return err}
 err=cmd.Wait()
 // Direct child has exited, even if nonzero. Keep acknowledgement briefly.
 time.Sleep(time.Second)
 cleanup:=os.RemoveAll(marker)
 if err!=nil{return err};return cleanup
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
