package main

import (
 "context"
 "crypto/sha512"
 "encoding/base64"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

type mirrorTestTransport func(*http.Request)(*http.Response,error)
func(f mirrorTestTransport)RoundTrip(r *http.Request)(*http.Response,error){return f(r)}
func mirrorResponse(r *http.Request,status int,body string)*http.Response{
 return &http.Response{StatusCode:status,Header:make(http.Header),Body:io.NopCloser(strings.NewReader(body)),ContentLength:int64(len(body)),Request:r}
}
func mirrorSRI(s string)string{
 h:=sha512.Sum512([]byte(s))
 return "sha512-"+base64.StdEncoding.EncodeToString(h[:])
}

func TestVerifiedMirrorArchive(t *testing.T){
 const pkg="@routilet/companion"
 const version="0.2.0-beta.1"
 const archive="verified native archive bytes"
 for _,tc:=range []struct{name string;status int;mirrorHash,body,mirrorURL string;want bool;download bool;dependencies bool}{
  {"valid",200,mirrorSRI(archive),archive,"https://registry.npmmirror.com/tarball.tgz",true,true,false},
  {"missing version",404,"","","",false,false,false},
  {"different metadata hash",200,mirrorSRI("other"),archive,"https://registry.npmmirror.com/tarball.tgz",false,false,false},
  {"untrusted URL",200,mirrorSRI(archive),archive,"https://example.invalid/tarball.tgz",false,false,false},
  {"bad bytes",200,mirrorSRI(archive),"corrupt bytes","https://registry.npmmirror.com/tarball.tgz",false,true,false},
  {"has dependencies",200,mirrorSRI(archive),archive,"https://registry.npmmirror.com/tarball.tgz",false,false,true},
 }{
  t.Run(tc.name,func(t *testing.T){
   downloaded:=false;mirrorChecks:=0
   m:=&Manager{client:&http.Client{Transport:mirrorTestTransport(func(r *http.Request)(*http.Response,error){
    var data map[string]any
    switch r.URL.Host{
    case "registry.npmjs.org":
     data=map[string]any{"name":pkg,"version":version,"dist":map[string]string{"integrity":mirrorSRI(archive),"tarball":"https://registry.npmjs.org/a.tgz"}}
     if tc.dependencies{data["dependencies"]=map[string]string{"dep":"1.0.0"}}
    case "registry.npmmirror.com":
     if r.URL.Path=="/tarball.tgz"{downloaded=true;return mirrorResponse(r,200,tc.body),nil}
     mirrorChecks++
     if tc.status!=200{return mirrorResponse(r,tc.status,"not found"),nil}
     data=map[string]any{"name":pkg,"version":version,"dist":map[string]string{"integrity":tc.mirrorHash,"tarball":tc.mirrorURL}}
    default:return nil,fmt.Errorf("unexpected URL %s",r.URL)
    }
    b,_:=json.Marshal(data)
    return mirrorResponse(r,200,string(b)),nil
   })}}
   snap:=t.TempDir()
   file,e:=m.verifiedMirrorArchive(context.Background(),nil,snap,pkg,version)
   if (e==nil)!=tc.want||downloaded!=tc.download{t.Fatalf("file=%q err=%v downloaded=%v",file,e,downloaded)}
   if tc.dependencies&&mirrorChecks!=0{t.Fatal("dependency package should bypass mirror")}
   if tc.want{
    b,e:=os.ReadFile(file);if e!=nil||string(b)!=archive{t.Fatalf("archive bytes %q %v",b,e)}
   }else if _,e:=os.Stat(filepath.Join(snap,mirrorArchive));!os.IsNotExist(e){t.Fatalf("failed download left archive: %v",e)}
  })
 }
}
func TestMirrorRedirectRejected(t *testing.T){
 m:=&Manager{client:&http.Client{Transport:mirrorTestTransport(func(r *http.Request)(*http.Response,error){
  if r.URL.Host!="registry.npmmirror.com"{t.Fatalf("unexpected redirect %s",r.URL)}
  res:=mirrorResponse(r,302,"");res.Header.Set("Location","https://example.invalid/malicious")
  return res,nil
 })}}
 _,e:=m.tarballInfo(context.Background(),"@routilet/companion","0.2.0-beta.1",true)
 if e==nil{t.Fatal("untrusted redirect accepted")}
}
func TestDownloadSourceDefaultAndSetting(t *testing.T){
 m:=&Manager{root:t.TempDir()}
 if s,e:=m.downloadSource();e!=nil||s!="auto"{t.Fatalf("default %q %v",s,e)}
 if e:=m.setDownloadSource("official");e!=nil{t.Fatal(e)}
 if s,e:=m.downloadSource();e!=nil||s!="official"{t.Fatalf("stored %q %v",s,e)}
 if e:=m.setDownloadSource("arbitrary-url");e==nil{t.Fatal("arbitrary source accepted")}
 if e:=m.setDownloadSource("auto");e!=nil{t.Fatal(e)}
 if s,e:=m.downloadSource();e!=nil||s!="auto"{t.Fatalf("stored %q %v",s,e)}
}
func TestMirrorProgressUsesMeasuredBytes(t *testing.T){
 j:=&installJob{}
 w:=&mirrorCounter{job:j,total:20}
 if n,e:=w.Write([]byte("abc"));n!=3||e!=nil{t.Fatalf("%d %v",n,e)}
 if j.p.DownloadBytes!=3||j.p.DownloadTotal!=20{t.Fatalf("%+v",j.p)}
 j.stage("正在安装已验证的文件…")
 if j.p.DownloadBytes!=0||j.p.DownloadTotal!=0{t.Fatal("stale download progress")}
}
