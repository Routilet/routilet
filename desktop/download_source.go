package main

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "net/http"
 "net/url"
 "os"
 "path/filepath"
 "strings"
 "time"
)

const mirrorRegistry = "https://registry.npmmirror.com/"

type downloadSettings struct { Source string `json:"source"` }
func validDownloadSource(s string) bool { return s=="auto" || s=="official" }
func(m *Manager) downloadSource()(string,error){
 b,e:=os.ReadFile(filepath.Join(m.root,"download-source.json"))
 if os.IsNotExist(e){return "auto",nil};if e!=nil{return "",e}
 var cfg downloadSettings
 if e=json.Unmarshal(b,&cfg);e!=nil{return "",e}
 if !validDownloadSource(cfg.Source){return "",errors.New("下载源设置无效")}
 return cfg.Source,nil
}
func(m *Manager) setDownloadSource(source string)error{
 if !validDownloadSource(source){return errors.New("未知下载源")}
 if m.progress().Active{return errors.New("安装任务进行中")}
 m.mu.Lock();defer m.mu.Unlock()
 done,e:=m.operationLock();if e!=nil{return e};defer done()
 if e=os.MkdirAll(m.root,0700);e!=nil{return e}
 return atomicJSON(filepath.Join(m.root,"download-source.json"),downloadSettings{Source:source})
}

// pnpm owns resolution, caching and integrity checks. No extra official
// metadata request is needed when the selected mirror works.
func installFromSources(ctx context.Context,job *installJob,snap string,install func(string)error)error{
 if e:=ctx.Err();e!=nil{return e}
 if job!=nil{job.stage("使用镜像源下载与安装")}
 mirrorErr:=install(mirrorRegistry)
 if mirrorErr==nil{return nil}
 if e:=ctx.Err();e!=nil{return e}
 // Only the newly created candidate snapshot is touched. A failed attempt
 // must not leave a registry-pinned lockfile or partially linked modules.
 for _,name:=range []string{"node_modules","pnpm-lock.yaml"}{
  if e:=os.RemoveAll(filepath.Join(snap,name));e!=nil{return errors.Join(mirrorErr,e)}
 }
 if job!=nil{job.stage("镜像不可用，切换官方源")}
 if e:=install(registry);e!=nil{return errors.Join(fmt.Errorf("镜像源：%w",mirrorErr),fmt.Errorf("官方源：%w",e))}
 return nil
}
func allowedDownloadURL(u *url.URL,mirror bool)bool{
 if u==nil||u.Scheme!="https"||u.User!=nil||u.Fragment!=""||u.Port()!=""{return false}
 host:=strings.ToLower(u.Hostname())
 if mirror{return host=="registry.npmmirror.com"||host=="cdn.npmmirror.com"}
 return host=="registry.npmjs.org"
}
func(m *Manager) downloadClient(timeout time.Duration,mirror bool)*http.Client{
 c:=http.Client{Timeout:timeout}
 if m.client!=nil{c.Transport=m.client.Transport}
 c.CheckRedirect=func(req *http.Request,via []*http.Request)error{
  if len(via)>=5||!allowedDownloadURL(req.URL,mirror){return errors.New("软件源重定向无效")}
  return nil
 }
 return &c
}
