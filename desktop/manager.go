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
 {"companion","Companion","快速测试代码，定位输出差异","@routilet/companion","latest"},
 {"observer","Observer","看清程序运行中的数据变化","@routilet/c-observer","latest"},
 {"c-exam-kit","C Exam Kit","常用函数随查随用，无需重复编写","@routilet/c-exam-kit","latest"},
 {"accoding-adapter","Accoding Adapter","在本地完成 Accoding 导题与提交","@routilet/accoding-adapter","latest"},
}
func(s appSpec) resource()bool{return s.ID=="c-exam-kit"}
func(s appSpec) adapter()bool{return s.ID=="accoding-adapter"}
func(s appSpec) component()bool{return s.resource()||s.adapter()}
type AppState struct {
 Component bool `json:"component"`
 ID string `json:"id"`
 Name string `json:"name"`
 Description string `json:"description"`
 Installed string `json:"installed"`
 Available string `json:"available"`
 Update bool `json:"update"`
 Rollback bool `json:"rollback"`
 Previous string `json:"previous"`
 ReleaseStatus string `json:"releaseStatus"`
 LocalError string `json:"localError"`
 Disabled bool `json:"disabled"`
 PreferenceError string `json:"preferenceError"`
 Note string `json:"note"`
}
type Manager struct {
 root,pnpm,bootstrap,ready string
 client *http.Client
 mu sync.Mutex
 jobMu sync.Mutex
 job *installJob
}
type activation struct { Snapshot, Version string; Previous, PreviousVersion string }
type nativeManifest struct {
 SchemaVersion int `json:"schemaVersion"`
 Version string `json:"version"`
 Adapter string `json:"adapter"`
 Protocol string `json:"protocol"`
 Platforms map[string]struct{Executable,Path,SHA256 string} `json:"platforms"`
}

func newManager() (*Manager,error) {
 exe,err:=os.Executable();if err!=nil{return nil,err}
 config,err:=os.UserConfigDir();if err!=nil{return nil,err}
 return &Manager{root:filepath.Join(config,"Routilet"),bootstrap:exe,pnpm:bundledPNPM(exe),client:&http.Client{Timeout:8*time.Second}},nil
}
func spec(id string)(appSpec,error){if id=="routilet"{return managerSpec,nil};for _,s:=range catalog{if s.ID==id{return s,nil}};return appSpec{},errors.New("未知应用")}
func (m *Manager) appDir(s appSpec)string{if s.ID=="routilet"{return filepath.Join(m.root,"manager")};return filepath.Join(m.root,"apps",s.ID)}
func validSnapshot(s string)bool{return strings.HasPrefix(s,"release-")&&filepath.Base(s)==s&&!strings.ContainsAny(s,"/\\")}
func (m *Manager) active(s appSpec)(activation,error){
 var a activation
 b,err:=os.ReadFile(filepath.Join(m.appDir(s),"activation.json"));if err!=nil{return a,err}
 if err=json.Unmarshal(b,&a);err!=nil{return a,err}
 if !validSnapshot(a.Snapshot)||a.Version==""||(a.Previous!=""&&(!validSnapshot(a.Previous)||a.PreviousVersion=="")){return a,errors.New("安装记录无效")}
 return a,nil
}
func atomicJSON(path string,a any)error{
 b,err:=json.Marshal(a);if err!=nil{return err}
 f,err:=os.CreateTemp(filepath.Dir(path),"activation-*.tmp");if err!=nil{return err}
 defer os.Remove(f.Name())
 if _,err=f.Write(b);err!=nil{f.Close();return err};if err=f.Sync();err!=nil{f.Close();return err}
 if err=f.Close();err!=nil{return err};return replaceFile(f.Name(),path)
}
var observerVersion=regexp.MustCompile(`^0\.(?:[2-9]|[1-9][0-9]+)\.[0-9]+(?:-[0-9A-Za-z.-]+)?$|^[1-9][0-9]*\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)

func (m *Manager) remote(s appSpec)(string,error){return m.remoteContext(context.Background(),s)}
func (m *Manager) remoteContext(ctx context.Context,s appSpec)(string,error){
 if s.Package==""{return "",errors.New("新版开发中")}
 channel,err:=m.channel();if err!=nil{return "",err}
 return m.resolveRelease(ctx,s,channel)
}
func (m *Manager) fetchTag(ctx context.Context,s appSpec,tag string)(string,error){
 addr:=registry+url.PathEscape(s.Package)+"/"+url.PathEscape(tag)
 req,err:=http.NewRequestWithContext(ctx,http.MethodGet,addr,nil);if err!=nil{return "",err}
 req.Header.Set("Accept","application/json")
 r,err:=m.client.Do(req);if err!=nil{return "",err};defer r.Body.Close()
 if r.StatusCode==http.StatusNotFound{return "",nil}
 if r.StatusCode!=200{return "",fmt.Errorf("软件源 HTTP %d",r.StatusCode)}
 var v struct{Version string}
 if err=json.NewDecoder(io.LimitReader(r.Body,4<<20)).Decode(&v);err!=nil{return "",err}
 if !validRelease(v.Version){return "",errors.New("软件源版本号无效")}
 return v.Version,nil
}
func (m *Manager) list()[]AppState{
 out:=make([]AppState,0,len(catalog))
 for _,s:=range catalog{
  a:=AppState{ID:s.ID,Name:s.Name,Description:s.Description,ReleaseStatus:"unknown",Component:s.component()}
  if s.Package==""{a.ReleaseStatus="development";a.Note="新版开发中"}else{
   current,err:=m.active(s)
   if err==nil{a.Installed=current.Version;a.Rollback=current.Previous!="";a.Previous=current.PreviousVersion}else if !os.IsNotExist(err){a.LocalError="安装记录异常"}
  }
  disabled,e:=m.disabled(s);a.Disabled=disabled
  if e!=nil{a.PreferenceError=e.Error()}
  if a.Disabled||a.PreferenceError!=""{a.ReleaseStatus="disabled"}
  out=append(out,a)
 }
 return out
}
func (m *Manager) check()[]AppState{
 out:=m.list()
 for i,s:=range catalog{
  if s.Package==""||out[i].Disabled||out[i].PreferenceError!=""{continue}
  v,err:=m.remote(s)
  if err!=nil{
   out[i].Note=err.Error()
   if errors.Is(err,errNoRelease){out[i].ReleaseStatus="unavailable"}else{out[i].ReleaseStatus="error"}
   continue
  }
  out[i].ReleaseStatus="available";out[i].Available=v
  out[i].Update=out[i].Installed!=""&&newer(v,out[i].Installed)
 }
 return out
}
// SemVer precedence: compare numeric strings without machine-integer overflow;
// build metadata does not affect precedence.
func newer(a,b string)bool{
 if !validRelease(a)||!validRelease(b){return false}
 parse:=func(v string)([]string,string){v,_,_=strings.Cut(v,"+");core,pre,_:=strings.Cut(v,"-");return strings.Split(core,"."),pre}
 numCompare:=func(x,y string)int{if len(x)<len(y){return -1};if len(x)>len(y){return 1};return strings.Compare(x,y)}
 x,p:=parse(a);y,q:=parse(b)
 for i:=0;i<3;i++{if c:=numCompare(x[i],y[i]);c!=0{return c>0}}
 if p==q{return false};if p==""{return true};if q==""{return false}
 ps,qs:=strings.Split(p,"."),strings.Split(q,".")
 for i:=0;i<len(ps)&&i<len(qs);i++{
  if ps[i]==qs[i]{continue}
  pn,qn:=digitsPattern.MatchString(ps[i]),digitsPattern.MatchString(qs[i])
  if pn&&qn{return numCompare(ps[i],qs[i])>0};if pn!=qn{return !pn}
  return ps[i]>qs[i]
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
 root,err:=filepath.EvalSymlinks(snapshot);if err!=nil{return "",fmt.Errorf("解析安装目录 %s: %w",snapshot,err)}
 pkgPath:=filepath.Join(root,"node_modules",filepath.FromSlash(s.Package))
 pkg,err:=filepath.EvalSymlinks(pkgPath);if err!=nil{return "",fmt.Errorf("解析软件包目录 %s: %w",pkgPath,err)}
 if !within(root,pkg){return "",errors.New("软件包路径超出安装目录")}
 manifestPath:=filepath.Join(pkg,"manifest.json")
 b,err:=os.ReadFile(manifestPath);if err!=nil{return "",fmt.Errorf("读取应用清单 %s: %w",manifestPath,err)}
 var manifest nativeManifest
 if err=json.Unmarshal(b,&manifest);err!=nil{return "",err}
 if manifest.SchemaVersion!=1||manifest.Version!=version{return "",errors.New("应用清单不兼容")}
 platform,ok:=manifest.Platforms[target()]
 if s.adapter(){if manifest.Adapter!="accoding"||manifest.Protocol!="routilet-oj-adapter/1"||platform.Executable!=""{return "",errors.New("适配器身份或协议不兼容")};platform.Executable=platform.Path}
 if !ok||platform.Executable==""||filepath.IsAbs(platform.Executable)||!filepath.IsLocal(filepath.FromSlash(platform.Executable)){return "",errors.New("当前平台没有有效入口")}
 entryPath:=filepath.Join(pkg,filepath.FromSlash(platform.Executable))
 entry,err:=filepath.EvalSymlinks(entryPath);if err!=nil{return "",fmt.Errorf("解析原生入口 %s: %w；%s",entryPath,err,describeEntryPath(entryPath))}
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
 if err:=checkPNPM(m.pnpm);err!=nil{return "",err}
 if st,err:=os.Stat(dir);err!=nil{return "",fmt.Errorf("pnpm 工作目录 %s: %w",dir,err)}else if !st.IsDir(){return "",fmt.Errorf("pnpm 工作路径不是目录: %s",dir)}
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
 if err!=nil{return output.String(),fmt.Errorf("pnpm %s 失败（程序: %s；工作目录: %s）: %w\n%s",args[0],m.pnpm,dir,err,output.String())}
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
func (m *Manager) install(id string)error{return m.installMode(id,false)}
func (m *Manager) installMode(id string,repair bool)error{return m.installExpected(id,repair,"")}
func (m *Manager) installExpected(id string,repair bool,expected string)error{
 s,err:=spec(id);if err!=nil{return err}
 if s.Package==""{return errors.New("新版尚未发布")}
 job,err:=m.beginJob(id);if err!=nil{return err}
 defer m.endJob(job)
 m.mu.Lock();defer m.mu.Unlock()
 ctx:=job.ctx
 release,err:=m.operationLock();if err!=nil{return err};defer release()
 if err=m.requireEnabled(s);err!=nil{return err}
 old,oldErr:=m.active(s)
 if oldErr!=nil{old=activation{}}
 var version string
 if repair {if oldErr!=nil{return oldErr};version=old.Version}else{version,err=m.remoteContext(ctx,s);if err!=nil{return err}}
 if expected!=""&&version!=expected{return errors.New("可用版本已变化，请重新检查更新")}
 if s.ID=="routilet"&&!repair&&(!newer(version,buildVersion)||(oldErr==nil&&!newer(version,old.Version))){return errors.New("Routilet 已是当前版本或新版已准备就绪")}
 if oldErr==nil&&old.Version!=version&&!newer(version,old.Version){return errors.New("当前版本比发行渠道更新，不自动降级")}
 job.stage("准备安装")
 if err=m.setupContext(ctx);err!=nil{return fmt.Errorf("准备安装 %s: %w",s.Name,err)}
 dir:=m.appDir(s);if err=os.MkdirAll(dir,0700);err!=nil{return err}
 snap,err:=os.MkdirTemp(dir,"release-");if err!=nil{return err}
 keep:=false;defer func(){if !keep{_ = os.RemoveAll(snap)}}()
 job.stage("解析与下载")
 if err=m.installPackageContext(ctx,job,snap,s.Package,version);err!=nil{return fmt.Errorf("下载与安装 %s %s: %w",s.Package,version,err)}
 job.stage("校验应用")
 if s.resource(){if err=sealKit(snap,s,version);err!=nil{return fmt.Errorf("校验资源 %s %s: %w",s.Name,version,err)}}else{
  entry,e:=nativeEntry(snap,s,version);if e!=nil{return fmt.Errorf("校验应用 %s %s: %w",s.Name,version,e)}
  if s.ID=="routilet"{if e=probeManager(ctx,entry,version);e!=nil{return e}}
  if s.adapter(){if e=probeAdapter(ctx,entry,version);e!=nil{return e}}
 }
 if err=job.commit();err!=nil{return err}
 if err=atomicJSON(filepath.Join(dir,"activation.json"),activation{Snapshot:filepath.Base(snap),Version:version,Previous:old.Snapshot,PreviousVersion:old.Version});err!=nil{return fmt.Errorf("激活版本 %s: %w",s.Name,err)}
 keep=true
 if s.ID!="routilet"{_ = m.cleanup(s)}
 return nil
}
func (m *Manager) installPackage(snap,pkg,version string)error{return m.installPackageContext(context.Background(),nil,snap,pkg,version)}
func (m *Manager) installPackageContext(ctx context.Context,job *installJob,snap,pkg,version string)error{
 b,err:=json.Marshal(map[string]any{"name":"routilet-managed-install","private":true,"dependencies":map[string]string{pkg:version}})
 if err!=nil{return err}
 if err=os.WriteFile(filepath.Join(snap,"package.json"),b,0600);err!=nil{return err}
 store,err:=prepareNativeInstall(snap,m.root);if err!=nil{return err}
 _,err=m.pnpmContext(ctx,job,snap,"install","--prod","--ignore-scripts","--no-frozen-lockfile","--registry="+registry,"--store-dir="+store,"--reporter=append-only")
 return err
}
func (m *Manager) launch(id string)error{
 return m.launchWith(id,m.spawnSupervisor)
}
// Pin the verified snapshot under the operation lock, then release it before
// starting/waiting for the supervisor. A child may read managed resources at startup.
func (m *Manager) launchWith(id string,spawn func(appSpec,activation,string,[]string)error)error{
 if id=="routilet"{return errors.New("使用原始入口打开 Routilet")}
 s,err:=spec(id);if err!=nil{return err}
 if s.component(){return errors.New("组件由 Companion 调用，不能直接打开")}
 s,a,_,marker,err:=m.pinApp(appCall{id:id});if err!=nil{return err}
 return spawn(s,a,marker,nil)
}
