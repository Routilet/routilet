package main

import (
 "bytes"
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

const appCallProtocol="routilet-app-call/1"
const maxQueryOutput=1<<20

type appCall struct {
 id string
 args []string
 wait,query bool
}

// Existing --open-app ID remains valid. Only explicitly bounded,
// non-interactive query commands are accepted; no shell or arbitrary paths.
func parseAppCall(args []string)(appCall,error){
 var c appCall
 if len(args)<2{return c,errors.New("需要应用 ID")}
 c.id=args[1];c.query=args[0]=="--query-app"
 if c.id=="routilet"{return c,errors.New("不能通过应用调用启动管理器")}
 if _,e:=spec(c.id);e!=nil{return c,e}
 rest:=args[2:]
 if !c.query&&len(rest)>0&&rest[0]=="--wait"{c.wait=true;rest=rest[1:]}
 if len(rest)>0{
  if rest[0]!="--"||len(rest)==1{return c,errors.New("目标应用参数必须放在 -- 之后")}
  c.args=append([]string(nil),rest[1:]...)
 }
 if c.query&&(len(c.args)!=1||(c.args[0]!="--capabilities"&&c.args[0]!="--version")){
  return c,errors.New("查询仅支持 --capabilities 或 --version")
 }
 return c,nil
}

// The current manager is the protocol-capable entry, not necessarily the
// original downloaded bootstrap. Existing manager snapshots are retained.
func applicationEnv(manager,bootstrap string)[]string{
 env:=[]string{}
 for _,kv:=range os.Environ(){
  key,_,_:=strings.Cut(kv,"=")
  if strings.EqualFold(key,"ROUTILET_EXECUTABLE")||strings.EqualFold(key,"ROUTILET_BOOTSTRAP"){continue}
  env=append(env,kv)
 }
 return append(env,"ROUTILET_EXECUTABLE="+manager,"ROUTILET_BOOTSTRAP="+bootstrap)
}
func (m *Manager) appCommand(ctx context.Context,entry string,args []string)(*exec.Cmd,error){
 manager,e:=os.Executable();if e!=nil{return nil,e}
 cmd:=exec.CommandContext(ctx,entry,args...)
 cmd.Dir=filepath.Dir(entry)
 cmd.Env=applicationEnv(manager,m.bootstrap)
 return cmd,nil
}

// Caller holds the process-wide operation lock while selecting the active
// snapshot and placing a marker, then releases it before waiting for the app.
func (m *Manager) pinApp(c appCall)(appSpec,activation,string,string,error){
 var s appSpec;var a activation
 m.mu.Lock();defer m.mu.Unlock()
 unlock,e:=m.operationLock();if e!=nil{return s,a,"","",e};defer unlock()
 s,e=spec(c.id);if e!=nil{return s,a,"","",e}
 if s.ID=="routilet"||s.Package==""{return s,a,"","",errors.New("应用尚不可用")}
 if e=m.requireEnabled(s);e!=nil{return s,a,"","",e}
 a,e=m.active(s);if os.IsNotExist(e){return s,a,"","",errors.New("应用尚未安装: "+s.Name)};if e!=nil{return s,a,"","",e}
 snap:=filepath.Join(m.appDir(s),a.Snapshot)
 entry,e:=nativeEntry(snap,s,a.Version);if e!=nil{return s,a,"","",e}
 runs:=filepath.Join(snap,".runs");if e=os.MkdirAll(runs,0700);e!=nil{return s,a,"","",e}
 marker,e:=os.MkdirTemp(runs,"run-")
 return s,a,entry,marker,e
}

type queryBuffer struct{b bytes.Buffer;overflow bool}
func (b *queryBuffer) Write(p []byte)(int,error){
 n:=len(p);left:=maxQueryOutput-b.b.Len()
 if n>left{b.overflow=true;p=p[:left]}
 _,_=b.b.Write(p);return n,nil
}

// Waiting calls run directly under this supervisor process, so exit status
// is the actual child status on both platforms (not macOS open's status).
// An unexpected supervisor death leaves the marker conservatively in place.
func(m *Manager) callApp(c appCall,stdout,stderr io.Writer)error{
 if !c.wait&&!c.query&&len(c.args)==0{return m.launch(c.id)}
 s,a,entry,marker,e:=m.pinApp(c);if e!=nil{return e}
 if !c.wait&&!c.query{
  return m.spawnSupervisor(s,a,marker,c.args)
 }
 ctx:=context.Background()
 if c.query{var cancel context.CancelFunc;ctx,cancel=context.WithTimeout(ctx,10*time.Second);defer cancel()}
 cmd,e:=m.appCommand(ctx,entry,c.args);if e!=nil{_ = os.RemoveAll(marker);return e}
 var output,diagnostic queryBuffer
 if c.query{
  hideChild(cmd);cmd.Stdout=&output;cmd.Stderr=&diagnostic;cmd.WaitDelay=time.Second
 }else{
  cmd.Stdout=stdout;cmd.Stderr=stderr
 }
 if e=cmd.Start();e!=nil{_ = os.RemoveAll(marker);return e}
 e=cmd.Wait()
 // Wait has reaped the direct child. Do not retain a marker merely because
 // the app returned a nonzero status.
 cleanup:=os.RemoveAll(marker)
 if c.query{
  if diagnostic.b.Len()>0{_,_=stderr.Write(diagnostic.b.Bytes())}
  if ctx.Err()!=nil{return errors.New("应用能力查询超时")}
  if output.overflow||diagnostic.overflow{return errors.New("应用能力查询输出过大")}
  if e==nil{if _,err:=stdout.Write(output.b.Bytes());err!=nil{return err}}
 }
 if e!=nil{return e}
 return cleanup
}
func appExitCode(err error)int{
 var exit *exec.ExitError
 if errors.As(err,&exit)&&exit.ExitCode()>0{return exit.ExitCode()}
 return 1
}
func (m *Manager) spawnSupervisor(s appSpec,a activation,marker string,args []string)error{
 exe,e:=os.Executable();if e!=nil{_ = os.RemoveAll(marker);return e}
 argv:=[]string{"--supervise",s.ID,a.Snapshot,a.Version,filepath.Base(marker)}
 if len(args)>0{argv=append(append(argv,"--"),args...)}
 cmd:=exec.Command(exe,argv...)
 cmd.Env=applicationEnv(exe,m.bootstrap)
 hideChild(cmd)
 if e=cmd.Start();e!=nil{_ = os.RemoveAll(marker);return e}
 _=cmd.Process.Release()
 until:=time.Now().Add(10*time.Second)
 for time.Now().Before(until){
  if b,e:=os.ReadFile(filepath.Join(marker,"started"));e==nil&&string(b)=="ok"{return nil}
  if b,e:=os.ReadFile(filepath.Join(marker,"error"));e==nil{return errors.New(string(b))}
  time.Sleep(50*time.Millisecond)
 }
 return fmt.Errorf("启动未确认；为安全起见保留运行标记")
}
