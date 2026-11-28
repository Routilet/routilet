package main

import (
 "context"
 "errors"
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

func TestRegistryMirrorFirst(t *testing.T){
 m:=&Manager{root:t.TempDir()}
 var calls []string
 mirrorStatus:=200
 m.client=&http.Client{Transport:mirrorTestTransport(func(r *http.Request)(*http.Response,error){
  calls=append(calls,r.URL.Host)
  if r.URL.Host=="registry.npmmirror.com"{if mirrorStatus==0{return nil,fmt.Errorf("DNS unavailable")};return mirrorResponse(r,mirrorStatus,`{"name":"@routilet/companion","version":"0.2.0-beta.3"}`),nil}
  if r.URL.Host=="registry.npmjs.org"{return mirrorResponse(r,200,`{"name":"@routilet/companion","version":"0.2.0-beta.4"}`),nil}
  return nil,errors.New("unknown registry")
 })}
 s,_:=spec("companion")
 if v,e:=m.fetchTag(context.Background(),s,"latest");e!=nil||v!="0.2.0-beta.3"||strings.Join(calls,",")!="registry.npmmirror.com"{t.Fatal(v,e,calls)}
 for _,status:=range []int{0,404,503}{
  mirrorStatus=status;calls=nil
  if v,e:=m.fetchTag(context.Background(),s,"latest");e!=nil||v!="0.2.0-beta.4"||strings.Join(calls,",")!="registry.npmmirror.com,registry.npmjs.org"{t.Fatal(v,e,calls)}
 }
 if e:=m.setDownloadSource("official");e!=nil{t.Fatal(e)}
 calls=nil
 if v,e:=m.fetchTag(context.Background(),s,"latest");e!=nil||v!="0.2.0-beta.4"||strings.Join(calls,",")!="registry.npmjs.org"{t.Fatal(v,e,calls)}
}
func TestInstallSourceFallback(t *testing.T){
 for _,mode:=range []string{"mirror-success","fallback","cancel","both-fail"}{t.Run(mode,func(t *testing.T){
  ctx,cancel:=context.WithCancel(context.Background());defer cancel()
  snap:=t.TempDir();var calls []string
  install:=func(reg string)error{
   calls=append(calls,reg)
   if reg==mirrorRegistry{
    if mode=="mirror-success"{return nil}
    if mode=="cancel"{cancel()}
    if e:=os.MkdirAll(filepath.Join(snap,"node_modules"),0700);e!=nil{t.Fatal(e)}
    if e:=os.WriteFile(filepath.Join(snap,"pnpm-lock.yaml"),[]byte("mirror"),0600);e!=nil{t.Fatal(e)}
    return errors.New("mirror failure")
   }
   for _,p:=range []string{"node_modules","pnpm-lock.yaml"}{if _,e:=os.Stat(filepath.Join(snap,p));!os.IsNotExist(e){t.Fatal("stale install",p,e)}}
   if mode=="both-fail"{return errors.New("official failure")};return nil
  }
  e:=installFromSources(ctx,nil,snap,install)
  if (e!=nil)!=(mode=="cancel"||mode=="both-fail"){t.Fatal(e)}
  want:=1;if mode=="fallback"||mode=="both-fail"{want=2}
  if len(calls)!=want||calls[0]!=mirrorRegistry{t.Fatal(calls)}
 })}
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
