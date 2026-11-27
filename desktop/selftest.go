package main

import (
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
)

func selfTest()error{
 m,err:=newManager();if err!=nil{return err}
 dir,err:=os.MkdirTemp("","routilet 测试 ");if err!=nil{return err};defer os.RemoveAll(dir)
 m.root=dir
 if err=m.setup();err!=nil{return err}
 snap,err:=os.MkdirTemp(dir,"release-");if err!=nil{return err}
 if err=m.installPackage(snap,"esbuild","0.25.12");err!=nil{return err}
 pkg,err:=filepath.EvalSymlinks(filepath.Join(snap,"node_modules","esbuild"));if err!=nil{return err}
 path:=filepath.Join(filepath.Dir(pkg),"@esbuild",target(),"bin","esbuild")
 if target()=="win32-x64"{path=filepath.Join(filepath.Dir(pkg),"@esbuild",target(),"esbuild.exe")}
 out,err:=exec.Command(path,"--version").CombinedOutput();if err!=nil{return fmt.Errorf("native smoke: %w %s",err,out)}
 if strings.TrimSpace(string(out))!="0.25.12"{return fmt.Errorf("wrong native version: %s",out)}
 fmt.Println("PASS: bundled pnpm and native npm installation")
 // Observer may not be published yet. Only report the actual result.
 observer,_:=spec("observer")
 v,err:=m.remote(observer)
 if err!=nil{fmt.Println("Observer availability:",err);return nil}
 if err=m.install("observer");err!=nil{return fmt.Errorf("Observer %s: %w",v,err)}
 fmt.Println("PASS: Observer native manifest and hash",v)
 var caps strings.Builder
 if err=m.callApp(appCall{id:"observer",query:true,args:[]string{"--capabilities"}},&caps,os.Stderr);err!=nil{return err}
 if !strings.Contains(caps.String(),"sessionSchemaVersions"){return fmt.Errorf("Observer capability response invalid: %s",caps.String())}
 fmt.Println("PASS: actual Observer protected capability query",caps.String())
 return nil
}
