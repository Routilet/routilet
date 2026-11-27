package main

import(
 "testing"
 "net/http"
 "strings"
 "io"
 "os"
 "path/filepath"
)
func TestUIReleaseState(t *testing.T){
 m:=&Manager{root:t.TempDir()}
 s,_:=spec("observer");dir:=m.appDir(s)
 if err:=os.MkdirAll(dir,0700);err!=nil{t.Fatal(err)}
 if err:=atomicJSON(filepath.Join(dir,"activation.json"),activation{Snapshot:"release-a",Version:"0.2.0",Previous:"release-b",PreviousVersion:"0.1.9"});err!=nil{t.Fatal(err)}
 status,body:=503,""
 m.client=&http.Client{Transport:tagTransport(func(r *http.Request)(*http.Response,error){return &http.Response{StatusCode:status,Body:io.NopCloser(strings.NewReader(body)),Header:make(http.Header)},nil})}
 local:=m.list();if local[0].ReleaseStatus!="unknown"||local[1].ReleaseStatus!="unknown"||local[1].Previous!="0.1.9"{t.Fatal(local)}
 a:=m.check()[1];if a.ReleaseStatus!="error"||a.Note==""||a.Installed!="0.2.0"||a.Update{t.Fatal(a)}
 status,body=200,`{"version":"0.1.2"}`
 a=m.check()[1];if a.ReleaseStatus!="unavailable"||a.Update{t.Fatal(a)}
 status,body=200,`{"version":"0.2.1"}`
 a=m.check()[1];if a.ReleaseStatus!="available"||!a.Update||a.Available!="0.2.1"{t.Fatal(a)}
 if e:=m.installExpected("observer",false,"0.2.2");e==nil||!strings.Contains(e.Error(),"可用版本已变化"){t.Fatal(e)}
 if e:=m.rollbackExpected("observer","0.2.3");e==nil||!strings.Contains(e.Error(),"回退目标已变化"){t.Fatal(e)}
}
func TestUILocalSelfAndCleanup(t *testing.T){
 m:=&Manager{root:t.TempDir()}
 dir:=m.appDir(managerSpec);if e:=os.MkdirAll(dir,0700);e!=nil{t.Fatal(e)}
 if e:=atomicJSON(filepath.Join(dir,"activation.json"),activation{Snapshot:"release-a",Version:"9999.0.0"});e!=nil{t.Fatal(e)}
 info:=m.selfLocal();if !info.Pending||info.PendingVersion!="9999.0.0"||info.ReleaseStatus!="unknown"{t.Fatal(info)}
 s,_:=spec("observer");d:=m.appDir(s);if e:=os.MkdirAll(filepath.Join(d,"release-a"),0700);e!=nil{t.Fatal(e)}
 if e:=os.MkdirAll(filepath.Join(d,"release-b",".runs","run-live"),0700);e!=nil{t.Fatal(e)}
 report,e:=m.clean();if e!=nil||report.Removed!=1||report.Protected!=1||len(report.Failed)!=0{t.Fatal(report,e)}
 if e:=os.WriteFile(filepath.Join(d,"activation.json"),[]byte("invalid"),0600);e!=nil{t.Fatal(e)}
 report,e=m.clean();if e!=nil||len(report.Failed)!=1{t.Fatal(report,e)}
}
