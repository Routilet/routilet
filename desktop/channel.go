package main

import (
 "context"
 "encoding/json"
 "errors"
 "os"
 "path/filepath"
 "regexp"
 "strings"
)

const stableChannel="stable"
const testChannel="test"
type channelSettings struct{ Channel string `json:"channel"` }
func(m *Manager)channel()(string,error){
 b,err:=os.ReadFile(filepath.Join(m.root,"release-channel.json"))
 if os.IsNotExist(err){return stableChannel,nil};if err!=nil{return "",err}
 var s channelSettings
 if err=json.Unmarshal(b,&s);err!=nil{return "",err}
 if s.Channel!=stableChannel&&s.Channel!=testChannel{return "",errors.New("发行通道设置无效")}
 return s.Channel,nil
}
func(m *Manager)setChannel(channel string)error{
 if channel!=stableChannel&&channel!=testChannel{return errors.New("未知发行通道")}
 if m.progress().Active{return errors.New("安装任务进行中")}
 m.mu.Lock();defer m.mu.Unlock()
 done,err:=m.operationLock();if err!=nil{return err};defer done()
 return atomicJSON(filepath.Join(m.root,"release-channel.json"),channelSettings{Channel:channel})
}
var releasePattern=regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)
var digitsPattern=regexp.MustCompile(`^[0-9]+$`)
func validRelease(v string)bool{
 if !releasePattern.MatchString(v){return false}
 v,_,_=strings.Cut(v,"+");_,pre,ok:=strings.Cut(v,"-")
 if ok{for _,x:=range strings.Split(pre,"."){if len(x)>1&&x[0]=='0'&&digitsPattern.MatchString(x){return false}}}
 return true
}
func compatibleRelease(s appSpec,v string)bool{
 if v==""||!validRelease(v){return false}
 if s.ID=="observer"{core,_,_:=strings.Cut(v,"+");return observerVersion.MatchString(core)}
 return true
}
func chooseRelease(s appSpec,channel,stable,next string)string{
 base,_,_:=strings.Cut(stable,"+")
 if !compatibleRelease(s,stable)||strings.Contains(base,"-"){stable=""}
 if channel!=testChannel||!compatibleRelease(s,next){return stable}
 if stable==""||newer(next,stable){return next}
 return stable
}
func(m *Manager)resolveRelease(ctx context.Context,s appSpec,channel string)(string,error){
 stable,err:=m.fetchTag(ctx,s,"latest");if err!=nil{return "",err}
 next:=""
 if channel==testChannel{
  next,err=m.fetchTag(ctx,s,"next");if err!=nil{return "",err}
 }
 v:=chooseRelease(s,channel,stable,next)
 if v==""{return "",errors.New("当前通道暂无兼容的原生版本")}
 return v,nil
}
