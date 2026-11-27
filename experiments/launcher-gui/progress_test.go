package main

import (
 "bytes"
 "context"
 "errors"
 "testing"
)

func TestProgressAndCancellation(t *testing.T){
 m:=&Manager{}
 j,err:=m.beginJob("observer");if err!=nil{t.Fatal(err)}
 if _,err=m.beginJob("observer");err==nil{t.Fatal("accepted concurrent job")}
 var b bytes.Buffer
 w:=&progressWriter{output:&b,job:j}
 for _,s:=range []string{"Progress: resolved 2, reused 0, down","loaded 1, added 0\r\nPackages: +2\n","Progress: resolved 2, reused 0, downloaded 2, added 1\n"}{_,_=w.Write([]byte(s))}
 p:=m.progress()
 if p.Total!=2||p.Added!=1||p.Downloaded!=2||p.Resolved!=2{t.Fatalf("bad progress %+v",p)}
 if !m.cancelJob()||!errors.Is(j.ctx.Err(),context.Canceled){t.Fatal("cancel failed")}
 if !errors.Is(j.commit(),context.Canceled){t.Fatal("cancelled task activated")}
 m.endJob(j)
 if m.progress().Active{t.Fatal("job left active")}
 j,err=m.beginJob("observer");if err!=nil{t.Fatal(err)}
 if err=j.commit();err!=nil{t.Fatal(err)}
 if m.cancelJob(){t.Fatal("accepted cancel during activation")}
 m.endJob(j)
}
