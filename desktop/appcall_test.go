package main

import (
 "bytes"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "os"
 "path/filepath"
 "reflect"
 "strings"
 "testing"
 "time"
)

// The copied test executable is a native fixture, not a shell script.
// TestMain handles only explicit fixture invocations.
func TestMain(m *testing.M){
 if os.Getenv("ROUTILET_APP_CALL_TEST")=="1"&&len(os.Args)>1{
  switch os.Args[1]{
  case "--capabilities":
   os.Stdout.WriteString("{\"sessionSchemaVersions\":[1]}\n");os.Exit(0)
  case "--version":
   os.Stderr.WriteString("fixture failure\n");os.Exit(7)
  case "--session":
   if len(os.Args)!=3||os.Getenv("ROUTILET_EXECUTABLE")==""||os.Getenv("ROUTILET_BOOTSTRAP")==""{os.Exit(9)}
   if e:=os.WriteFile(os.Args[2],[]byte("started"),0600);e!=nil{os.Exit(10)}
   for i:=0;i<100;i++{if _,e:=os.Stat(os.Args[2]+".finish");e==nil{os.Exit(0)};time.Sleep(50*time.Millisecond)}
   os.Exit(11)
  }
 }
 os.Exit(m.Run())
}
func TestParseAppCall(t *testing.T){
 c,e:=parseAppCall([]string{"--open-app","observer","--wait","--","--session","中文 path.json"})
 if e!=nil||!c.wait||c.query||!reflect.DeepEqual(c.args,[]string{"--session","中文 path.json"}){t.Fatal(c,e)}
 for _,args:=range [][]string{
  {"--query-app","observer"},{"--query-app","observer","--","--session","x"},
  {"--open-app","observer","--session","x"},{"--open-app","observer","--"},
  {"--open-app","routilet"},{"--open-app","unknown"},
 }{if _,e:=parseAppCall(args);e==nil{t.Fatal("accepted",args)}}
 if c,e=parseAppCall([]string{"--open-app","observer"});e!=nil||c.wait||len(c.args)!=0{t.Fatal(c,e)}
}
func appCallFixture(t *testing.T)(*Manager,appSpec,string){
 t.Helper();if target()==""{t.Skip("target platform only")}
 t.Setenv("ROUTILET_APP_CALL_TEST","1")
 exe,e:=os.Executable();if e!=nil{t.Fatal(e)}
 raw,e:=os.ReadFile(exe);if e!=nil{t.Fatal(e)}
 m:=&Manager{root:t.TempDir(),bootstrap:exe}
 s,_:=spec("observer");snap:=filepath.Join(m.appDir(s),"release-fixture")
 pkg:=filepath.Join(snap,"node_modules",filepath.FromSlash(s.Package))
 if e=os.MkdirAll(pkg,0700);e!=nil{t.Fatal(e)}
 name:="fixture.exe"
 if e=os.WriteFile(filepath.Join(pkg,name),raw,0700);e!=nil{t.Fatal(e)}
 sum:=sha256.Sum256(raw)
 manifest:=map[string]any{"schemaVersion":1,"version":"0.2.0","platforms":map[string]any{target():map[string]string{"executable":name,"sha256":hex.EncodeToString(sum[:])}}}
 b,_:=json.Marshal(manifest);if e=os.WriteFile(filepath.Join(pkg,"manifest.json"),b,0600);e!=nil{t.Fatal(e)}
 if e=atomicJSON(filepath.Join(m.appDir(s),"activation.json"),activation{Snapshot:"release-fixture",Version:"0.2.0"});e!=nil{t.Fatal(e)}
 return m,s,snap
}
func TestProtectedQuery(t *testing.T){
 m,_,snap:=appCallFixture(t)
 var out,errOut bytes.Buffer
 e:=m.callApp(appCall{id:"observer",query:true,args:[]string{"--capabilities"}},&out,&errOut)
 if e!=nil||!strings.Contains(out.String(),"sessionSchemaVersions")||errOut.Len()!=0{t.Fatal(e,out.String(),errOut.String())}
 out.Reset();errOut.Reset()
 e=m.callApp(appCall{id:"observer",query:true,args:[]string{"--version"}},&out,&errOut)
 if e==nil||appExitCode(e)!=7||out.Len()!=0||!strings.Contains(errOut.String(),"fixture failure"){t.Fatal(e,out.String(),errOut.String())}
 runs,e:=os.ReadDir(filepath.Join(snap,".runs"));if e!=nil||len(runs)!=0{t.Fatal(runs,e)}
}
func TestWaitProtectsSnapshot(t *testing.T){
 m,s,snap:=appCallFixture(t)
 session:=filepath.Join(t.TempDir(),"中文 space.json")
 done:=make(chan error,1)
 go func(){done<-m.callApp(appCall{id:"observer",wait:true,args:[]string{"--session",session}},os.Stdout,os.Stderr)}()
 defer os.WriteFile(session+".finish",nil,0600)
 until:=time.Now().Add(4*time.Second)
 for{
  if _,e:=os.Stat(session);e==nil{break}
  select{case e:=<-done:t.Fatalf("app prematurely exited: %v",e);default:}
  if time.Now().After(until){t.Fatal("fixture did not start")};time.Sleep(20*time.Millisecond)
 }
 if e:=m.uninstall(s.ID);e!=nil{t.Fatal(e)}
 if _,e:=os.Stat(snap);e!=nil{t.Fatal("running snapshot deleted",e)}
 if e:=os.WriteFile(session+".finish",nil,0600);e!=nil{t.Fatal(e)}
 select{case e:=<-done:if e!=nil{t.Fatal(e)};case <-time.After(5*time.Second):t.Fatal("wait did not return")}
 runs,e:=os.ReadDir(filepath.Join(snap,".runs"));if e!=nil||len(runs)!=0{t.Fatal(runs,e)}
 if e=m.clean();e!=nil{t.Fatal(e)}
 if _,e=os.Stat(snap);!os.IsNotExist(e){t.Fatal("unused snapshot not cleaned",e)}
}
func TestQueryOutputBound(t *testing.T){
 var b queryBuffer
 p:=bytes.Repeat([]byte("x"),maxQueryOutput+1)
 if n,e:=b.Write(p);n!=len(p)||e!=nil||!b.overflow||b.b.Len()!=maxQueryOutput{t.Fatal(n,e,b.overflow,b.b.Len())}
}
