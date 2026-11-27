package main

import (
 "os"
 "path/filepath"
 "testing"
)

func TestActivation(t *testing.T) {
 p:=&probe{data:t.TempDir()}
 if a,err:=p.state();err!=nil||a.Current!=""{t.Fatalf("initial: %+v %v",a,err)}
 for _,a:=range []activation{{Current:"release-a"},{Current:"release-b",Previous:"release-a"}} {
  if err:=p.activate(a);err!=nil{t.Fatal(err)}
  got,err:=p.state();if err!=nil||got!=a{t.Fatalf("got %+v %v",got,err)}
 }
 if err:=os.WriteFile(filepath.Join(p.data,"activation.json"),[]byte(`{"current":"../outside"}`),0600);err!=nil{t.Fatal(err)}
 if _,err:=p.state();err==nil{t.Fatal("unsafe path accepted")}
}
