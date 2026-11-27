package main

import (
 "context"
 "bytes"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "runtime"
 "strings"
 "sync"
 "time"
)

const registry = "https://registry.npmjs.org/"
const pnpmVersion = "12.22.0"

type appSpec struct { ID, Name, Description, Package, Channel string }
var catalog=[]appSpec{
 {"companion","Companion","代码管理与本地评测","",""},
 {"observer","Observer","程序运行观察","@routilet/c-observer","next"},
}
type AppState struct {
 ID string `json:"id"`
 Name string `json:"name"`
 Description string `json:"description"`
 Installed string `json:"installed"`
 Available string `json:"available"`
 Update bool `json:"update"`
 Note string `json:"note"`
}
type Manager struct {
 root,pnpm string
 client *http.Client
 mu sync.Mutex
 jobMu sync.Mutex
 job *installJob
}
type activation struct { Snapshot, Version string }
type nativeManifest struct {
 SchemaVersion int `json:"schemaVersion"`
 Version string `json:"version"`
 Platforms map[string]struct{Executable,SHA256 string} `json:"platforms"`
}

func newManager() (*Manager,error) {
 exe,err:=os.Executable();if err!=nil{return nil,err}
 config,err:=os.UserConfigDir();if err!=nil{return nil,err}
 name:="pnpm";if runtime.GOOS=="windows"{name+=".exe"}
 return &Manager{root:filepath.Join(config,"Routilet"),pnpm:filepath.Join(filepath.Dir(exe),name),client:&http.Client{Timeout:8*time.Second}},nil
}
func spec(id string)(appSpec,error){for _,s:=range catalog{if s.ID==id{return s,nil}};return appSpec{},errors.New("未知应用")}
func (m *Manager) appDir(s appSpec)string{return filepath.Join(m.root,"apps",s.ID)}
func validSnapshot(s string)bool{return strings.HasPrefix(s,"release-")&&filepath.Base(s)==s&&!strings.ContainsAny(s,"/\\")}
func (m *Manager) active(s appSpec)(activation,error){
 var a activation
 b,err:=os.ReadFile(filepath.Join(m.appDir(s),"activation.json"));if err!=nil{return a,err}
 if err=json.Unmarshal(b,&a);err!=nil{return a,err}
 if !validSnapshot(a.Snapshot)||a.Version==""{return a,errors.New("安装记录无效")}
 return a,nil
}
func atomicJSON(path string,a activation)error{
 b,err:=json.Marshal(a);if err!=nil{return err}
 f,err:=os.CreateTemp(filepath.Dir(path),"activation-*.tmp");if err!=nil{return err}
 defer os.Remove(f.Name())
 if _,err=f.Write(b);err!=nil{f.Close();return err};if err=f.Sync();err!=nil{f.Close();return err}
 if err=f.Close();err!=nil{return err};return os.Rename(f.Name(),path)
}
var observerVersion=regexp.MustCompile(`^0\.(?:[2-9]|[1-9][0-9]+)\.[0-9]+(?:-[0-9A-Za-z.-]+)?$|^[1-9][0-9]*\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)

func (m *Manager) remote(s appSpec)(string,error){return m.remoteContext(context.Background(),s)}
func (m *Manager) remoteContext(ctx context.Context,s appSpec)(string,error){
 if s.Package==""{return "",errors.New("新版开发中")}
 // Request only the explicit channel; do not silently fall back to legacy TUI.
 addr:=registry+url.PathEscape(s.Package)+"/"+url.PathEscape(s.Channel)
 req,err:=http.NewRequestWithContext(ctx,http.MethodGet,addr,nil);if err!=nil{return "",err}
 req.Header.Set("Accept","application/json")
 r,err:=m.client.Do(req);if err!=nil{return "",err};defer r.Body.Close()
 if r.StatusCode==http.StatusNotFound{return "",errors.New("原生预览版尚未发布")}
 if r.StatusCode!=200{return "",fmt.Errorf("软件源 HTTP %d",r.StatusCode)}
 var v struct{Version string}
 if err=json.NewDecoder(io.LimitReader(r.Body,4<<20)).Decode(&v);err!=nil{return "",err}
 if s.ID=="observer"&&!observerVersion.MatchString(v.Version){return "",errors.New("原生预览版尚未发布")}
 return v.Version,nil
}
func (m *Manager) list()[]AppState{
 out:=make([]AppState,0,len(catalog))
 for _,s:=range catalog{
  a:=AppState{ID:s.ID,Name:s.Name,Description:s.Description}
  if s.Package==""{a.Note="新版开发中"}else{
   current,err:=m.active(s)
   if err==nil{a.Installed=current.Version}else if !os.IsNotExist(err){a.Note="安装记录异常"}
  }
  out=append(out,a)
 }
 return out
}
func (m *Manager) check()[]AppState{
 out:=m.list()
 for i,s:=range catalog{
  if s.Package==""{continue}
  v,err:=m.remote(s)
  if err!=nil{
   if out[i].Installed==""{out[i].Note=err.Error()}
   continue
  }
  out[i].Available=v
  // Channel changes can include rollback; do not advertise a downgrade.
  out[i].Update=out[i].Installed!=""&&newer(v,out[i].Installed)
 }
 return out
}
// Small comparison for the published numeric core and beta.N/rc.N channels.
// Stable outranks prerelease of the same core. Other suffixes compare by
// dot-separated numeric identifiers, as in SemVer.
func newer(a,b string)bool{
 split:=func(v string)([3]int,string){
  core,pre,_:=strings.Cut(v,"-");var n [3]int
  fmt.Sscanf(core,"%d.%d.%d",&n[0],&n[1],&n[2]);return n,pre
 }
 x,p:=split(a);y,q:=split(b)
 for i:=0;i<3;i++{if x[i]!=y[i]{return x[i]>y[i]}}
 if p==q{return false};if p==""{return true};if q==""{return false}
 ps,qs:=strings.Split(p,"."),strings.Split(q,".")
 for i:=0;i<len(ps)&&i<len(qs);i++{
  if ps[i]==qs[i]{continue}
  var n,m int
  _,en:=fmt.Sscanf(ps[i],"%d",&n);_,em:=fmt.Sscanf(qs[i],"%d",&m)
  numeric:=regexp.MustCompile(`^[0-9]+$`)
  pn,qn:=en==nil&&numeric.MatchString(ps[i]),em==nil&&numeric.MatchString(qs[i])
  if pn&&qn{return n>m};if pn!=qn{return !pn};return ps[i]>qs[i]
 }
 return len(ps)>len(qs)
}
func target() string {
 if runtime.GOOS=="windows"&&runtime.GOARCH=="amd64"{return "win32-x64"}
 if runtime.GOOS=="darwin"&&runtime.GOARCH=="arm64"{return "darwin-arm64"}
 return ""
}
func within(root,p string)bool{
 r,err:=filepath.Rel(root,p)
 return err==nil&&r!=".."&&!strings.HasPrefix(r,".."+string(os.PathSeparator))
}
func nativeEntry(snapshot string,s appSpec,version string)(string,error){
 root,err:=filepath.EvalSymlinks(snapshot);if err!=nil{return "",err}
 pkg,err:=filepath.EvalSymlinks(filepath.Join(root,"node_modules",filepath.FromSlash(s.Package)));if err!=nil{return "",err}
 if !within(root,pkg){return "",errors.New("软件包路径超出安装目录")}
 b,err:=os.ReadFile(filepath.Join(pkg,"manifest.json"));if err!=nil{return "",err}
 var manifest nativeManifest
 if err=json.Unmarshal(b,&manifest);err!=nil{return "",err}
 if manifest.SchemaVersion!=1||manifest.Version!=version{return "",errors.New("应用清单不兼容")}
 platform,ok:=manifest.Platforms[target()]
 if !ok||platform.Executable==""||filepath.IsAbs(platform.Executable)||!filepath.IsLocal(filepath.FromSlash(platform.Executable)){return "",errors.New("当前平台没有有效入口")}
 entry,err:=filepath.EvalSymlinks(filepath.Join(pkg,filepath.FromSlash(platform.Executable)));if err!=nil{return "",err}
 if !within(pkg,entry){return "",errors.New("可执行文件路径超出软件包")}
 f,err:=os.Open(entry);if err!=nil{return "",err};defer f.Close()
 st,err:=f.Stat();if err!=nil{return "",err};if !st.Mode().IsRegular(){return "",errors.New("入口不是普通文件")}
 h:=sha256.New();if _,err=io.Copy(h,f);err!=nil{return "",err}
 if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)),platform.SHA256){return "",errors.New("原生文件校验失败")}
 return entry,nil
}
func (m *Manager) pnpmCommand(dir string,args ...string)(string,error){
 return m.pnpmContext(context.Background(),nil,dir,args...)
}
func (m *Manager) pnpmContext(parent context.Context,job *installJob,dir string,args ...string)(string,error){
 ctx,cancel:=context.WithTimeout(parent,5*time.Minute);defer cancel()
 cmd:=exec.CommandContext(ctx,m.pnpm,args...);cmd.Dir=dir;hideChild(cmd)
 for _,kv:=range os.Environ(){
  k:=strings.ToLower(strings.SplitN(kv,"=",2)[0])
  if strings.HasPrefix(k,"npm_config_")||strings.HasPrefix(k,"pnpm_")||k=="node_options"||k=="node_path"{continue}
  cmd.Env=append(cmd.Env,kv)
 }
 cmd.Env=append(cmd.Env,"PNPM_HOME="+filepath.Join(m.root,"pnpm-home"),"NPM_CONFIG_USERCONFIG="+filepath.Join(m.root,"user.npmrc"),"NPM_CONFIG_GLOBALCONFIG="+filepath.Join(m.root,"global.npmrc"),"NPM_CONFIG_UPDATE_NOTIFIER=false")
 var output bytes.Buffer
 w:=&progressWriter{output:&output,job:job}
 cmd.Stdout=w;cmd.Stderr=w
 err:=cmd.Run();w.finish()
 if parent.Err()!=nil{return "",parent.Err()}
 if err!=nil{return output.String(),fmt.Errorf("pnpm: %w\n%s",err,output.String())}
 return strings.TrimSpace(output.String()),nil
}
func (m *Manager) setup()error{return m.setupContext(context.Background())}
func (m *Manager) setupContext(ctx context.Context)error{
 if err:=os.MkdirAll(m.root,0700);err!=nil{return err}
 for _,n:=range []string{"user.npmrc","global.npmrc"}{if err:=os.WriteFile(filepath.Join(m.root,n),nil,0600);err!=nil{return err}}
 v,err:=m.pnpmContext(ctx,nil,m.root,"--version");if err!=nil{return err}
 if v!=pnpmVersion{return fmt.Errorf("需要随附的 pnpm %s，实际 %s",pnpmVersion,v)}
 return nil
}
func (m *Manager) install(id string)error{
 s,err:=spec(id);if err!=nil{return err}
 if s.Package==""{return errors.New("新版尚未发布")}
 job,err:=m.beginJob(id);if err!=nil{return err}
 defer m.endJob(job)
 m.mu.Lock();defer m.mu.Unlock()
 ctx:=job.ctx
 version,err:=m.remoteContext(ctx,s);if err!=nil{return err}
 if old,e:=m.active(s);e==nil&&old.Version!=version&&!newer(version,old.Version){return errors.New("当前版本比发行渠道更新，不自动降级")}
 job.stage("准备安装")
 if err=m.setupContext(ctx);err!=nil{return err}
 dir:=m.appDir(s);if err=os.MkdirAll(dir,0700);err!=nil{return err}
 snap,err:=os.MkdirTemp(dir,"release-");if err!=nil{return err}
 keep:=false;defer func(){if !keep{_ = os.RemoveAll(snap)}}()
 job.stage("解析与下载")
 if err=m.installPackageContext(ctx,job,snap,s.Package,version);err!=nil{return err}
 job.stage("校验应用")
 if _,err=nativeEntry(snap,s,version);err!=nil{return err}
 if err=job.commit();err!=nil{return err}
 if err=atomicJSON(filepath.Join(dir,"activation.json"),activation{filepath.Base(snap),version});err!=nil{return err}
 keep=true
 return nil
}
func (m *Manager) installPackage(snap,pkg,version string)error{return m.installPackageContext(context.Background(),nil,snap,pkg,version)}
func (m *Manager) installPackageContext(ctx context.Context,job *installJob,snap,pkg,version string)error{
 b,err:=json.Marshal(map[string]any{"name":"routilet-preview-install","private":true,"dependencies":map[string]string{pkg:version}})
 if err!=nil{return err}
 if err=os.WriteFile(filepath.Join(snap,"package.json"),b,0600);err!=nil{return err}
 _,err=m.pnpmContext(ctx,job,snap,"install","--prod","--ignore-scripts","--no-frozen-lockfile","--registry="+registry,"--store-dir="+filepath.Join(m.root,"store"),"--reporter=append-only")
 return err
}
func (m *Manager) launch(id string)error{
 m.mu.Lock();defer m.mu.Unlock()
 s,err:=spec(id);if err!=nil{return err}
 a,err:=m.active(s);if err!=nil{return err}
 entry,err:=nativeEntry(filepath.Join(m.appDir(s),a.Snapshot),s,a.Version);if err!=nil{return err}
 // macOS launch the .app via LaunchServices; on Windows start the GUI
 // executable directly. No parent-owned pipes or cancellation.
 if runtime.GOOS=="darwin"{
  app:=entry
  for filepath.Ext(app)!=".app"{
   next:=filepath.Dir(app);if next==app{return errors.New("macOS 应用包不存在")};app=next
  }
  return exec.Command("/usr/bin/open","-n",app).Run()
 }
 cmd:=exec.Command(entry);cmd.Dir=filepath.Dir(entry)
 if err=cmd.Start();err!=nil{return err}
 return cmd.Process.Release()
}
