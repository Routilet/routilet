package main

import (
 "fmt"
 "errors"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "runtime"
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
 // Companion uses the new package, never legacy Accoding. Validate both
 // release channels, skipping only confirmed absent/incompatible releases.
 companion,_:=spec("companion")
 var lastCompanion string
 for _,channel:=range []string{stableChannel,testChannel}{
  if err=m.setChannel(channel);err!=nil{return err}
  cv,e:=m.remote(companion)
  if errors.Is(e,errNoRelease){fmt.Println("Companion "+channel+" availability:",e);continue}
  if e!=nil{return fmt.Errorf("Companion %s registry: %w",channel,e)}
  if cv!=lastCompanion{
   if e=m.installExpected(companion.ID,false,cv);e!=nil{return fmt.Errorf("Companion %s: %w",cv,e)}
   lastCompanion=cv
  }
  fmt.Println("PASS: Companion "+channel+" native manifest and hash",cv)
 }
 // Actual published beta, installed using the same manager and channel.
 adapter,_:=spec("accoding-adapter")
 if err=m.setChannel(testChannel);err!=nil{return err}
 av,err:=m.remote(adapter);if err!=nil{return fmt.Errorf("Adapter registry: %w",err)}
 if err=m.installExpected(adapter.ID,false,av);err!=nil{return fmt.Errorf("Adapter install: %w",err)}
 if err=m.adapterAvailable(adapter.ID);err!=nil{return err}
 var adapterOut strings.Builder
 if err=m.callAdapter(adapter.ID,strings.NewReader("{\"id\":1,\"method\":\"capabilities\",\"params\":{\"protocol\":\"routilet-oj-adapter/1\"}}\n"),&adapterOut);err!=nil{return err}
 if !strings.Contains(adapterOut.String(),"routilet-oj-adapter/1"){return errors.New("Adapter capability response invalid")}
 fmt.Println("PASS: actual Adapter install, verified manifest and protected stdio",av)
 if err=m.setChannel(stableChannel);err!=nil{return err}
 kit,_:=spec("c-exam-kit");kv,e:=m.remote(kit);if e!=nil{return e}
 if e=m.installExpected(kit.ID,false,kv);e!=nil{return e}
 k,e:=m.resourceData(kit.ID);if e!=nil||k.Version!=kv{return fmt.Errorf("Exam Kit export: %v",e)}
 fmt.Println("PASS: actual Exam Kit install and verified resource export",kv)
 if e=m.setDisabled(kit.ID,true);e!=nil{return e}
 if _,e=m.resourceData(kit.ID);!errors.Is(e,errAppDisabled){return errors.New("disabled resource exported")}
 if e=m.setDisabled(kit.ID,false);e!=nil{return e}
 // Observer may not be published yet. Only report the actual result.
 observer,_:=spec("observer")
 v,err:=m.remote(observer)
 if err!=nil{fmt.Println("Observer availability:",err);return nil}
 if err=m.install("observer");err!=nil{return fmt.Errorf("Observer %s: %w",v,err)}
 if runtime.GOOS=="windows"{
  active,e:=m.active(observer);if e!=nil{return e}
  snap:=filepath.Join(m.appDir(observer),active.Snapshot)
  // Windows may expand short names (RUNNER~1) or resolve a junction in the
  // temporary root. Compare paths relative to the same canonical snapshot.
  canonicalSnap,e:=filepath.EvalSymlinks(snap);if e!=nil{return e}
  pkg:=filepath.Join(canonicalSnap,"node_modules",filepath.FromSlash(observer.Package))
  resolved,e:=filepath.EvalSymlinks(pkg);if e!=nil{return e}
  if !strings.EqualFold(filepath.Clean(resolved),filepath.Clean(pkg)){return fmt.Errorf("Windows native package layout: expected %s; resolved %s",pkg,resolved)}
  if _,e=os.Stat(filepath.Join(m.root,"store-native"));e!=nil{return e}
  fmt.Println("PASS: Windows native package physical layout and isolated store")
  // Exercise the warm store and repair path, keeping the previous snapshot.
  if e=m.installMode(observer.ID,true);e!=nil{return fmt.Errorf("Windows warm-store repair: %w",e)}
  fmt.Println("PASS: Windows warm-store repair")
 }
 fmt.Println("PASS: Observer native manifest and hash",v)
 var caps strings.Builder
 if err=m.callApp(appCall{id:"observer",query:true,args:[]string{"--capabilities"}},&caps,os.Stderr);err!=nil{return err}
 if !strings.Contains(caps.String(),"sessionSchemaVersions"){return fmt.Errorf("Observer capability response invalid: %s",caps.String())}
 fmt.Println("PASS: actual Observer protected capability query",caps.String())
 return nil
}
