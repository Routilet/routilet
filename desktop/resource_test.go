package main
import("bytes";"encoding/json";"errors";"os";"path/filepath";"testing")
func TestResourceLifecycle(t *testing.T){
 m:=&Manager{root:t.TempDir()};s,_:=spec("c-exam-kit")
 if !s.resource()||!s.component()||s.adapter(){t.Fatal(s)}
 if _,e:=adapterSpec(s.ID);e==nil{t.Fatal("resource treated as adapter")}
 makeKit:=func(name,v string)string{
  snap:=filepath.Join(m.appDir(s),name);pkg:=filepath.Join(snap,"node_modules",filepath.FromSlash(s.Package))
  if e:=os.MkdirAll(filepath.Join(pkg,"preset"),0700);e!=nil{t.Fatal(e)}
  files:=map[string]string{"package.json":`{"name":"@routilet/c-exam-kit","version":"`+v+`"}`,"LICENSE":"license","preset/CC-BY-4.0.txt":"license","preset/manifest.json":`{"schemaVersion":2,"id":"exam-io-c99","language":"c","languageId":"c99","standard":"c99","headerFile":"kit.h","systemHeaders":[],"components":[{"id":"a","symbols":["a"],"requires":[],"code":"int a;"}]}`,"preset/reference.json":`{"schemaVersion":1,"version":"`+v+`","categories":[{}],"sections":[{}]}`}
  for n,b:=range files{if e:=os.WriteFile(filepath.Join(pkg,filepath.FromSlash(n)),[]byte(b),0600);e!=nil{t.Fatal(e)}}
  if e:=sealKit(snap,s,v);e!=nil{t.Fatal(e)}
  return pkg
 }
 old:=makeKit("release-old","0.6.10");makeKit("release-new","0.6.11")
 if e:=atomicJSON(filepath.Join(m.appDir(s),"activation.json"),activation{Snapshot:"release-new",Version:"0.6.11",Previous:"release-old",PreviousVersion:"0.6.10"});e!=nil{t.Fatal(e)}
 var b bytes.Buffer
 if e:=m.exportResource(s.ID,&b);e!=nil{t.Fatal(e)}
 var k kitExport;if e:=json.Unmarshal(b.Bytes(),&k);e!=nil||k.Version!="0.6.11"||len(k.Files)!=5{t.Fatal(k.Version,e)}
 if e:=m.rollback(s.ID);e!=nil{t.Fatal(e)}
 if k,e:=m.resourceData(s.ID);e!=nil||k.Version!="0.6.10"{t.Fatal(k.Version,e)}
 if e:=m.setDisabled(s.ID,true);e!=nil{t.Fatal(e)}
 if _,e:=m.resourceData(s.ID);!errors.Is(e,errAppDisabled){t.Fatal(e)}
 if e:=m.setDisabled(s.ID,false);e!=nil{t.Fatal(e)}
 if e:=os.WriteFile(filepath.Join(old,"LICENSE"),[]byte("modified"),0600);e!=nil{t.Fatal(e)}
 if _,e:=m.resourceData(s.ID);e==nil{t.Fatal("accepted modified resource")}
 if e:=m.launch(s.ID);e==nil{t.Fatal("resource launched")}
 if e:=m.uninstall(s.ID);e!=nil{t.Fatal(e)}
 if _,e:=m.resourceData(s.ID);e==nil{t.Fatal("export after uninstall")}
}
