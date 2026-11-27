package main
import("os";"path/filepath";"testing")
func TestCleanupAndLock(t *testing.T){
 m:=&Manager{root:t.TempDir()};s,_:=spec("observer");dir:=m.appDir(s)
 for _,n:=range []string{"release-a","release-b","release-c","release-d"}{if e:=os.MkdirAll(filepath.Join(dir,n),0700);e!=nil{t.Fatal(e)}}
 if e:=os.MkdirAll(filepath.Join(dir,"release-c",".runs","run-live"),0700);e!=nil{t.Fatal(e)}
 if e:=atomicJSON(filepath.Join(dir,"activation.json"),activation{Snapshot:"release-a",Version:"1",Previous:"release-b",PreviousVersion:"0"});e!=nil{t.Fatal(e)}
 done,e:=m.operationLock();if e!=nil{t.Fatal(e)}
 if other,e:=m.operationLock();e==nil{other();t.Fatal("concurrent lock accepted")}
 m.cleanup(s);done()
 for _,n:=range []string{"release-a","release-b","release-c"}{if _,e=os.Stat(filepath.Join(dir,n));e!=nil{t.Fatal(n,e)}}
 if _,e=os.Stat(filepath.Join(dir,"release-d"));!os.IsNotExist(e){t.Fatal("unused snapshot survived",e)}
 if e=m.uninstall("observer");e!=nil{t.Fatal(e)}
 if _,e=os.Stat(filepath.Join(dir,"release-c"));e!=nil{t.Fatal("live version removed",e)}
 if _,e=os.Stat(filepath.Join(dir,"release-a"));!os.IsNotExist(e){t.Fatal("uninstalled snapshot survived",e)}
}
