package main

import (
 "context"
 "crypto/sha512"
 "crypto/subtle"
 "encoding/base64"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "os"
 "path/filepath"
 "strings"
 "time"
)

const mirrorRegistry = "https://registry.npmmirror.com/"
const maxMirrorBytes int64 = 512 << 20
const mirrorArchive = "verified-download.tgz"

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

type tarballMetadata struct {
 Name string `json:"name"`
 Version string `json:"version"`
 Dist struct{
  Integrity string `json:"integrity"`
  Tarball string `json:"tarball"`
 } `json:"dist"`
 // Keep the current native, self-contained package contract. Future
 // dependency-bearing distributions use the normal official pnpm path.
 Dependencies map[string]string `json:"dependencies"`
 OptionalDependencies map[string]string `json:"optionalDependencies"`
 PeerDependencies map[string]string `json:"peerDependencies"`
}
func sha512Integrity(s string)([]byte,error){
 for _,part:=range strings.Fields(s){
  if !strings.HasPrefix(part,"sha512-"){continue}
  raw,e:=base64.StdEncoding.DecodeString(strings.TrimPrefix(part,"sha512-"))
  if e==nil&&len(raw)==sha512.Size{return raw,nil}
 }
 return nil,errors.New("缺少有效 SHA-512 完整性信息")
}
func allowedDownloadURL(u *url.URL,mirror bool)bool{
 if u==nil||u.Scheme!="https"||u.User!=nil||u.Fragment!=""||u.Port()!=""{return false}
 host:=strings.ToLower(u.Hostname())
 if mirror{return host=="registry.npmmirror.com"||host=="cdn.npmmirror.com"}
 return host=="registry.npmjs.org"
}
func(m *Manager) downloadClient(timeout time.Duration,mirror bool)*http.Client{
 c:=http.Client{}
 if m.client!=nil{c.Transport=m.client.Transport}
 c.Timeout=timeout
 c.CheckRedirect=func(req *http.Request,via []*http.Request)error{
  if len(via)>=5||!allowedDownloadURL(req.URL,mirror){return errors.New("下载重定向不受信任")}
  return nil
 }
 return &c
}
func(m *Manager) tarballInfo(ctx context.Context,pkg,version string,mirror bool)(tarballMetadata,error){
 var v tarballMetadata
 base,timeout:=registry,8*time.Second
 if mirror{base,timeout=mirrorRegistry,3*time.Second}
 req,e:=http.NewRequestWithContext(ctx,http.MethodGet,base+url.PathEscape(pkg)+"/"+url.PathEscape(version),nil);if e!=nil{return v,e}
 req.Header.Set("Accept","application/json")
 resp,e:=m.downloadClient(timeout,mirror).Do(req);if e!=nil{return v,e};defer resp.Body.Close()
 if resp.StatusCode!=http.StatusOK{return v,fmt.Errorf("软件源 HTTP %d",resp.StatusCode)}
 if e=json.NewDecoder(io.LimitReader(resp.Body,4<<20)).Decode(&v);e!=nil{return v,e}
 if v.Name!=pkg||v.Version!=version{return v,errors.New("软件包身份或版本不一致")}
 return v,nil
}
func(j *installJob) mirrorProgress(n,total int64){
 if j==nil{return}
 j.mu.Lock();defer j.mu.Unlock()
 if !j.p.Cancelled&&total>0&&n>=0&&n<=total{j.p.DownloadBytes=n;j.p.DownloadTotal=total}
}
type mirrorCounter struct { job *installJob; n,total int64 }
func(w *mirrorCounter) Write(p []byte)(int,error){
 w.n+=int64(len(p));w.job.mirrorProgress(w.n,w.total);return len(p),nil
}

// Download only a tarball whose hash is pinned by the official registry.
// Never give pnpm an unverified mirror archive or a mirror registry.
func(m *Manager) verifiedMirrorArchive(ctx context.Context,job *installJob,snap,pkg,version string)(string,error){
 official,e:=m.tarballInfo(ctx,pkg,version,false);if e!=nil{return "",e}
 if len(official.Dependencies)+len(official.OptionalDependencies)+len(official.PeerDependencies)!=0{return "",errors.New("非独立软件包，使用官方源")}
 want,e:=sha512Integrity(official.Dist.Integrity);if e!=nil{return "",e}
 mirror,e:=m.tarballInfo(ctx,pkg,version,true);if e!=nil{return "",e}
 got,e:=sha512Integrity(mirror.Dist.Integrity)
 if e!=nil||subtle.ConstantTimeCompare(want,got)!=1{return "",errors.New("镜像尚未同步或完整性信息不一致")}
 u,e:=url.Parse(mirror.Dist.Tarball);if e!=nil||!allowedDownloadURL(u,true){return "",errors.New("镜像下载地址不受信任")}
 req,e:=http.NewRequestWithContext(ctx,http.MethodGet,u.String(),nil);if e!=nil{return "",e}
 req.Header.Set("Accept-Encoding","identity")
 if job!=nil{job.stage("正在使用加速源下载…")}
 resp,e:=m.downloadClient(2*time.Minute,true).Do(req);if e!=nil{return "",e};defer resp.Body.Close()
 if resp.StatusCode!=http.StatusOK{return "",fmt.Errorf("加速源 HTTP %d",resp.StatusCode)}
 if resp.ContentLength>maxMirrorBytes{return "",errors.New("加速下载文件过大")}
 file:=filepath.Join(snap,mirrorArchive)
 f,e:=os.OpenFile(file,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if e!=nil{return "",e}
 keep:=false;defer func(){f.Close();if !keep{os.Remove(file)}}()
 h:=sha512.New()
 counter:=&mirrorCounter{job:job,total:resp.ContentLength}
 job.mirrorProgress(0,resp.ContentLength)
 n,e:=io.Copy(io.MultiWriter(f,h,counter),io.LimitReader(resp.Body,maxMirrorBytes+1))
 if e!=nil{return "",e}
 if n>maxMirrorBytes||resp.ContentLength>=0&&n!=resp.ContentLength{return "",errors.New("加速下载大小不符")}
 if subtle.ConstantTimeCompare(want,h.Sum(nil))!=1{return "",errors.New("加速下载完整性校验失败")}
 if e=f.Close();e!=nil{return "",e}
 if e=ctx.Err();e!=nil{return "",e}
 keep=true
 return file,nil
}
