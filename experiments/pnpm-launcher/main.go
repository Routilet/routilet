package main

import (
 "bufio"
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "time"
)

const (
 pnpmVersion = "12.22.0"
 registry = "https://registry.npmjs.org/"
 oldVersion = "0.25.11"
 newVersion = "0.25.12"
)

type activation struct {
 Current string `json:"current"`
 Previous string `json:"previous,omitempty"`
}

type probe struct { base, pnpm, data string }

func newProbe(base string) (*probe, error) {
 exe := "pnpm"
 if runtime.GOOS == "windows" { exe += ".exe" }
 p := &probe{base:base, pnpm:filepath.Join(base, exe), data:filepath.Join(base,"probe-data")}
 if err := os.MkdirAll(p.data,0700); err != nil { return nil,err }
 for _, name := range []string{"user.npmrc","global.npmrc"} {
  if err := os.WriteFile(filepath.Join(p.data,name),nil,0600); err != nil { return nil,err }
 }
 // Change only this process and children, never the system/user PATH.
 path := "/usr/bin:/bin:/usr/sbin:/sbin"
 if runtime.GOOS == "windows" {
  root := os.Getenv("SystemRoot")
  path = filepath.Join(root,"System32")+";"+root
 }
 if err := os.Setenv("PATH",path); err != nil { return nil,err }
 if _,err := exec.LookPath("node"); err == nil { return nil,errors.New("Node is still on the experiment PATH") }
 return p,nil
}

func (p *probe) command(dir, name string, args ...string) ([]byte,error) {
 ctx,cancel := context.WithTimeout(context.Background(),5*time.Minute)
 defer cancel()
 cmd := exec.CommandContext(ctx,name,args...)
 cmd.Dir = dir
 // Isolate package-manager configuration/cache; keep its default linker/import method.
 for _, kv := range os.Environ() {
  key := strings.ToLower(strings.SplitN(kv,"=",2)[0])
  if strings.HasPrefix(key,"npm_config_") || strings.HasPrefix(key,"pnpm_") || key=="node_options" || key=="node_path" { continue }
  cmd.Env = append(cmd.Env,kv)
 }
 cmd.Env = append(cmd.Env,"NPM_CONFIG_USERCONFIG="+filepath.Join(p.data,"user.npmrc"),
  "NPM_CONFIG_GLOBALCONFIG="+filepath.Join(p.data,"global.npmrc"),
  "PNPM_HOME="+filepath.Join(p.data,"pnpm-home"))
 var output bytes.Buffer
 cmd.Stdout = &output
 cmd.Stderr = &output
 err := cmd.Run()
 if err != nil { return output.Bytes(),fmt.Errorf("%s %v: %w\n%s",filepath.Base(name),args,err,output.String()) }
 return output.Bytes(),nil
}

func (p *probe) verifyRuntime() error {
 out,err := p.command(p.base,p.pnpm,"--version")
 if err != nil { return err }
 if strings.TrimSpace(string(out)) != pnpmVersion { return fmt.Errorf("unexpected pnpm version: %s",out) }
 fmt.Println("pnpm",strings.TrimSpace(string(out)),"(Node not on PATH)")
 return nil
}

func (p *probe) state() (activation,error) {
 var a activation
 b,err := os.ReadFile(filepath.Join(p.data,"activation.json"))
 if os.IsNotExist(err) { return a,nil }
 if err != nil { return a,err }
 err=json.Unmarshal(b,&a)
 for _,s:=range []string{a.Current,a.Previous} {
  if s!="" && (!strings.HasPrefix(s,"release-") || filepath.Base(s)!=s) { return a,errors.New("invalid activation path") }
 }
 return a,err
}

func (p *probe) activate(a activation) error {
 b,err:=json.MarshalIndent(a,"","  ")
 if err!=nil{return err}
 f,err:=os.CreateTemp(p.data,"activation-*.tmp")
 if err!=nil{return err}
 defer os.Remove(f.Name())
 if _,err=f.Write(b);err!=nil{f.Close();return err}
 if err=f.Sync();err!=nil{f.Close();return err}
 if err=f.Close();err!=nil{return err}
 return os.Rename(f.Name(),filepath.Join(p.data,"activation.json"))
}

func (p *probe) runVersion(id string) (string,error) {
 if !strings.HasPrefix(id,"release-") || filepath.Base(id)!=id { return "",errors.New("no valid installed snapshot") }
 dir:=filepath.Join(p.data,id)
 pkg:="darwin-arm64"
 entry:=[]string{"bin","esbuild"}
 if runtime.GOOS=="windows" { pkg="win32-x64";entry=[]string{"esbuild.exe"} }
 if !(runtime.GOOS=="windows" && runtime.GOARCH=="amd64" || runtime.GOOS=="darwin" && runtime.GOARCH=="arm64") { return "",errors.New("unsupported experiment platform") }
 // esbuild declares platform binaries as optional dependencies. Resolve the
 // platform package from its own dependency context, not from .bin shims.
 mainDir,err:=filepath.EvalSymlinks(filepath.Join(dir,"node_modules","esbuild"))
 if err!=nil{return "",err}
 target:=filepath.Join(append([]string{filepath.Dir(mainDir),"@esbuild",pkg},entry...)...)
 native,err:=filepath.EvalSymlinks(target)
 if err!=nil{return "",err}
 rel,err:=filepath.Rel(dir,native)
 if err!=nil || rel==".." || strings.HasPrefix(rel,".."+string(os.PathSeparator)) {return "",errors.New("native path escaped snapshot")}
 out,err:=p.command(dir,native,"--version")
 return strings.TrimSpace(string(out)),err
}

func (p *probe) prepare(version, restoreFrom string, offline bool) (id string, err error) {
 dir,err:=os.MkdirTemp(p.data,"release-")
 if err!=nil{return "",err}
 defer func(){if err!=nil{_ = os.RemoveAll(dir)}}()
 id=filepath.Base(dir)
 manifest:=[]byte(fmt.Sprintf("{\"name\":\"routilet-pnpm-probe\",\"private\":true,\"dependencies\":{\"esbuild\":%q}}\n",version))
 if restoreFrom!="" {
  for _,name:=range []string{"package.json","pnpm-lock.yaml"} {
   b,e:=os.ReadFile(filepath.Join(p.data,restoreFrom,name));if e!=nil{return "",e}
   if e=os.WriteFile(filepath.Join(dir,name),b,0600);e!=nil{return "",e}
  }
 }else if err=os.WriteFile(filepath.Join(dir,"package.json"),manifest,0600);err!=nil{return "",err}
 args:=[]string{"install","--prod","--ignore-scripts","--registry="+registry,"--store-dir="+filepath.Join(p.data,"store"),"--reporter=append-only"}
 if restoreFrom!="" {args=append(args,"--frozen-lockfile")}else{args=append(args,"--no-frozen-lockfile")}
 if offline {args=append(args,"--offline")}
 start:=time.Now()
 out,err:=p.command(dir,p.pnpm,args...)
 if err!=nil{return "",err}
 fmt.Printf("%s\nInstall elapsed: %s\n",out,time.Since(start).Round(time.Millisecond))
 if _,err=os.Stat(filepath.Join(dir,"pnpm-lock.yaml"));err!=nil{return "",err}
 actual,err:=p.runVersion(id)
 if err!=nil{return "",err}
 if actual!=version{return "",fmt.Errorf("expected %s, got %s",version,actual)}
 fmt.Println("Verified native binary",actual,"in",id)
 return id,nil
}

func (p *probe) install(version string) error {
 a,err:=p.state();if err!=nil{return err}
 id,err:=p.prepare(version,"",false);if err!=nil{return err}
 return p.activate(activation{Current:id,Previous:a.Current})
}

func (p *probe) rollback() error {
 a,err:=p.state();if err!=nil{return err}
 if a.Previous=="" {return errors.New("no previous snapshot")}
 if _,err=p.runVersion(a.Previous);err!=nil{return err}
 return p.activate(activation{Current:a.Previous,Previous:a.Current})
}

func (p *probe) current() (string,error) {
 a,err:=p.state();if err!=nil{return "",err}
 return p.runVersion(a.Current)
}

func (p *probe) selfTest() error {
 // Only the packaged artifact's private directory is used. Do not overwrite
 // any existing activation: CI starts from a freshly extracted artifact.
 a,err:=p.state();if err!=nil{return err}
 if a.Current!="" {return errors.New("self-test requires fresh probe-data")}
 if err=p.install(oldVersion);err!=nil{return err}
 a,err=p.state();if err!=nil{return err}
 if err=p.install(newVersion);err!=nil{return err}
 v,err:=p.runVersion(a.Current);if err!=nil||v!=oldVersion{return fmt.Errorf("old snapshot: %s %v",v,err)}
 v,err=p.current();if err!=nil||v!=newVersion{return fmt.Errorf("new snapshot: %s %v",v,err)}
 before,err:=p.state();if err!=nil{return err}
 if _,err=p.prepare("999.999.999","",true);err==nil{return errors.New("impossible offline install succeeded")}
 after,err:=p.state();if err!=nil||after!=before{return errors.New("failed candidate changed activation")}
 if err=p.rollback();err!=nil{return err}
 v,err=p.current();if err!=nil||v!=oldVersion{return fmt.Errorf("rollback: %s %v",v,err)}
 if _,err=p.prepare(oldVersion,a.Current,true);err!=nil{return fmt.Errorf("offline frozen restore: %w",err)}
 fmt.Println("PASS: native pnpm, official registry, platform package, default store, update, old snapshot, failed candidate, rollback, offline frozen restore")
 return nil
}

func run() error {
 exe,err:=os.Executable();if err!=nil{return err}
 p,err:=newProbe(filepath.Dir(exe));if err!=nil{return err}
 if err=p.verifyRuntime();err!=nil{return err}
 if len(os.Args)==2 && os.Args[1]=="--self-test" {return p.selfTest()}
 if len(os.Args)>1 {return errors.New("unknown arguments")}
 scanner:=bufio.NewScanner(os.Stdin)
 for {
  a,err:=p.state();if err!=nil{return err}
  fmt.Printf("\nRoutilet pnpm probe — isolated experiment\nCurrent: %s\n1 Install %s\n2 Update %s\n3 Launch native program (--version)\n4 Roll back\n0 Exit\n> ",a.Current,oldVersion,newVersion)
  if !scanner.Scan(){return scanner.Err()}
  switch strings.TrimSpace(scanner.Text()) {
  case "1":err=p.install(oldVersion)
  case "2":err=p.install(newVersion)
  case "3":var v string;v,err=p.current();if err==nil{fmt.Println("Native program version:",v)}
  case "4":err=p.rollback()
  case "0":return nil
  default:fmt.Println("Unknown selection")
  }
  if err!=nil{fmt.Println("Error:",err)}
 }
}

func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}
