package main
import ("errors";"io";"os";"path/filepath";"testing";"runtime";"strings")
func TestDisablePersistenceAndProtection(t *testing.T){
 m:=&Manager{root:t.TempDir()};s,_:=spec("observer")
 if e:=m.setDisabled(s.ID,true);e!=nil{t.Fatal(e)}
 if e:=m.setDisabled("companion",true);e!=nil{t.Fatal(e)}
 if e:=m.setDisabled("routilet",true);e==nil{t.Fatal("disabled manager")}
 for _,a:=range m.check(){if !a.Disabled||a.ReleaseStatus!="disabled"||a.Update{t.Fatal(a)}}
 if e:=m.install(s.ID);!errors.Is(e,errAppDisabled){t.Fatal(e)}
 if e:=m.launch(s.ID);!errors.Is(e,errAppDisabled){t.Fatal(e)}
 if e:=m.callApp(appCall{id:s.ID,query:true,args:[]string{"--version"}},io.Discard,io.Discard);!errors.Is(e,errAppDisabled){t.Fatal(e)}
 if e:=m.callApp(appCall{id:s.ID,wait:true,args:[]string{"--session","test"}},io.Discard,io.Discard);!errors.Is(e,errAppDisabled){t.Fatal(e)}
 dir:=m.appDir(s);snap:=filepath.Join(dir,"release-a")
 if e:=os.MkdirAll(snap,0700);e!=nil{t.Fatal(e)}
 if e:=atomicJSON(filepath.Join(dir,"activation.json"),activation{Snapshot:"release-a",Version:"0.2.0"});e!=nil{t.Fatal(e)}
 m2:=&Manager{root:m.root}
 if a:=m2.list()[1];!a.Disabled||a.Installed!="0.2.0"{t.Fatal(a)}
 if r,e:=m2.clean();e!=nil||r.Protected!=1{t.Fatal(r,e)}
 if e:=m2.setDisabled(s.ID,false);e!=nil{t.Fatal(e)}
 if a:=m2.list()[1];a.Disabled||a.Installed!="0.2.0"{t.Fatal(a)}
 if e:=m2.setDisabled(s.ID,true);e!=nil{t.Fatal(e)}
 if e:=m2.uninstall(s.ID);e!=nil{t.Fatal(e)}
 if a:=m2.list()[1];!a.Disabled||a.Installed!=""{t.Fatal(a)}
}
func TestBundledPNPMPaths(t *testing.T){
 dir:=t.TempDir();boot:=filepath.Join(dir,"Routilet.exe")
 if runtime.GOOS!="windows"{if bundledPNPM(boot)!=filepath.Join(dir,"pnpm"){t.Fatal(bundledPNPM(boot))};return}
 legacy:=filepath.Join(dir,"pnpm.exe");preferred:=filepath.Join(dir,"runtime","pnpm.exe")
 if bundledPNPM(boot)!=preferred{t.Fatal(bundledPNPM(boot))}
 if e:=checkPNPM(preferred);e==nil||!strings.Contains(e.Error(),preferred){t.Fatal(e)}
 if e:=os.WriteFile(legacy,nil,0600);e!=nil{t.Fatal(e)}
 if bundledPNPM(boot)!=legacy{t.Fatal(bundledPNPM(boot))}
 if e:=os.MkdirAll(filepath.Dir(preferred),0700);e!=nil{t.Fatal(e)}
 if e:=os.WriteFile(preferred,nil,0600);e!=nil{t.Fatal(e)}
 if bundledPNPM(boot)!=preferred{t.Fatal(bundledPNPM(boot))}
}
