package main

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "os/exec"
 "path/filepath"
 "time"
)

const adapterCallProtocol="routilet-adapter-call/1"
func adapterSpec(id string)(appSpec,error){
 s,e:=spec(id);if e!=nil||!s.component(){return appSpec{},errors.New("未知适配器组件")};return s,nil
}
// Availability is offline and read-only. Launch rechecks active version under
// the operation lock, so this result is not a promise about a later launch.
func(m *Manager) adapterAvailable(id string)error{
 s,e:=adapterSpec(id);if e!=nil{return e}
 m.mu.Lock();defer m.mu.Unlock()
 done,e:=m.operationLock();if e!=nil{return e};defer done()
 if e=m.requireEnabled(s);e!=nil{return e}
 a,e:=m.active(s);if e!=nil{return e}
 _,e=nativeEntry(filepath.Join(m.appDir(s),a.Snapshot),s,a.Version);return e
}
// Forward only stdio, never shell arguments. A marker covers the whole child
// lifetime. On abrupt supervisor termination the marker remains conservative.
func(m *Manager) callAdapter(id string,stdin io.Reader,stdout io.Writer)error{
 if _,e:=adapterSpec(id);e!=nil{return e}
 _,_,entry,marker,e:=m.pinApp(appCall{id:id});if e!=nil{return e}
 cmd:=exec.Command(entry,"--stdio");cmd.Dir=filepath.Dir(entry);hideChild(cmd)
 cmd.Stdin=stdin;cmd.Stdout=stdout;cmd.Stderr=nil
 if e=cmd.Start();e!=nil{_ = os.RemoveAll(marker);return fmt.Errorf("适配器进程启动失败: %w",e)}
 e=cmd.Wait();cleanup:=os.RemoveAll(marker);if e!=nil{return fmt.Errorf("适配器进程异常退出: %w",e)};return cleanup
}
// Check the candidate without credentials, network, or writable operations.
func probeAdapter(parent context.Context,entry,version string)error{
 ctx,cancel:=context.WithTimeout(parent,10*time.Second);defer cancel()
 cmd:=exec.CommandContext(ctx,entry,"--stdio");cmd.Dir=filepath.Dir(entry);hideChild(cmd)
 cmd.Stdin=bytes.NewBufferString("{\"id\":1,\"method\":\"capabilities\",\"params\":{\"protocol\":\"routilet-oj-adapter/1\"}}\n")
 var out queryBuffer;cmd.Stdout=&out;cmd.Stderr=nil;cmd.WaitDelay=time.Second
 if e:=cmd.Run();e!=nil{return fmt.Errorf("适配器离线握手失败: %w",e)}
 if out.overflow{return errors.New("适配器握手输出过大")}
 var r struct{ID int;Result struct{Adapter,Version,Protocol string;Methods []string};Error json.RawMessage}
 if json.Unmarshal(out.b.Bytes(),&r)!=nil||r.ID!=1||len(r.Error)!=0||r.Result.Adapter!="accoding"||r.Result.Version!=version||r.Result.Protocol!="routilet-oj-adapter/1"{return errors.New("适配器握手身份或版本不兼容")}
 for _,method:=range r.Result.Methods{if method=="capabilities"{return nil}}
 return errors.New("适配器缺少能力声明")
}
