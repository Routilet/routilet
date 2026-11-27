package main
import("testing";"net/http";"io";"strings";"context")
func TestChannelSelection(t *testing.T){
 s:=managerSpec
 for _,c:=range []struct{channel,latest,next,want string}{
 {"stable","1.2.0","1.3.0-beta.1","1.2.0"},
 {"test","1.2.0","1.1.9-beta.1","1.2.0"},
 {"test","1.2.0","1.2.0-beta.9","1.2.0"},
 {"test","1.2.0","1.3.0-beta.1","1.3.0-beta.1"},
 {"test","1.2.0","1.2.0","1.2.0"},
 {"test","1.2.0","","1.2.0"},
 {"test","","1.3.0-beta.1","1.3.0-beta.1"},
 {"stable","","1.3.0-beta.1",""},
 {"test","1.2.0","1.3.0-beta.01","1.2.0"},
 {"test","1.2.0+build.1","1.2.0+build.2","1.2.0+build.1"},
 {"test","1.9.0","1.10.0-beta.1","1.10.0-beta.1"},
 {"test","1.2.0","1.99999999999999999999999.0-beta.1","1.99999999999999999999999.0-beta.1"},
 }{if got:=chooseRelease(s,c.channel,c.latest,c.next);got!=c.want{t.Fatalf("%+v: got %s",c,got)}}
 observer,_:=spec("observer")
 if v:=chooseRelease(observer,"stable","0.1.2","0.2.0-beta.1");v!=""{t.Fatal("legacy TUI selected",v)}
 if v:=chooseRelease(observer,"test","0.1.2","0.2.0-beta.1");v!="0.2.0-beta.1"{t.Fatal(v)}
 companion,_:=spec("companion");if companion.Package!="@routilet/companion"{t.Fatal(companion)}
 if v:=chooseRelease(companion,"stable","0.0.0","");v!=""{t.Fatal("placeholder selected",v)}
 if v:=chooseRelease(companion,"test","0.0.0","0.1.0-beta.1");v!="0.1.0-beta.1"{t.Fatal(v)}
}
func TestChannelPersistence(t *testing.T){
 m:=&Manager{root:t.TempDir()}
 if v,e:=m.channel();e!=nil||v!="stable"{t.Fatal(v,e)}
 if e:=m.setChannel("test");e!=nil{t.Fatal(e)}
 if v,e:=m.channel();e!=nil||v!="test"{t.Fatal(v,e)}
 if e:=m.setChannel("invalid");e==nil{t.Fatal("invalid accepted")}
 if e:=m.setChannel("stable");e!=nil{t.Fatal(e)}
 if v,e:=m.channel();e!=nil||v!="stable"{t.Fatal(v,e)}
}
type tagTransport func(*http.Request)(*http.Response,error)
func(f tagTransport)RoundTrip(r *http.Request)(*http.Response,error){return f(r)}
func TestResolveTags(t *testing.T){
 m:=&Manager{root:t.TempDir()}
 var tags []string
 m.client=&http.Client{Transport:tagTransport(func(r *http.Request)(*http.Response,error){
  tag:=r.URL.Path[strings.LastIndex(r.URL.Path,"/")+1:];tags=append(tags,tag)
  body:=`{"version":"1.2.0"}`;if tag=="next"{body=`{"version":"1.2.0-beta.9"}`}
  return &http.Response{StatusCode:200,Body:io.NopCloser(strings.NewReader(body)),Header:make(http.Header)},nil
 })}
 if v,e:=m.resolveRelease(context.Background(),managerSpec,"stable");e!=nil||v!="1.2.0"||len(tags)!=1{t.Fatal(v,e,tags)}
 tags=nil
 if v,e:=m.resolveRelease(context.Background(),managerSpec,"test");e!=nil||v!="1.2.0"||len(tags)!=2{t.Fatal(v,e,tags)}
 m.client=&http.Client{Transport:tagTransport(func(r *http.Request)(*http.Response,error){return &http.Response{StatusCode:503,Body:io.NopCloser(strings.NewReader("")),Header:make(http.Header)},nil})}
 if _,e:=m.resolveRelease(context.Background(),managerSpec,"test");e==nil{t.Fatal("accepted unknown latest")}
}
