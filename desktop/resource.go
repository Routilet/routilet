package main

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "strings"
 "unicode/utf8"
)
const resourceProtocol="routilet-resource/1"
const kitMaxBytes=2<<20
type kitExport struct {
 Protocol string `json:"protocol"`
 ID string `json:"id"`
 Version string `json:"version"`
 Files map[string][]byte `json:"files"`
}
func readKitResource(snap string,s appSpec,version string)(kitExport,error){
 k:=kitExport{resourceProtocol,s.ID,version,map[string][]byte{}}
 if !s.resource(){return k,errors.New("不是资源组件")}
 root,e:=filepath.EvalSymlinks(snap);if e!=nil{return k,e}
 pkg,e:=filepath.EvalSymlinks(filepath.Join(root,"node_modules",filepath.FromSlash(s.Package)));if e!=nil{return k,e}
 if !within(root,pkg){return k,errors.New("资源路径超出安装目录")}
 total:=0
 for _,name:=range []string{"package.json","LICENSE","preset/manifest.json","preset/reference.json","preset/CC-BY-4.0.txt","preset/manual.zh-CN.md"}{
  path:=filepath.Join(pkg,filepath.FromSlash(name))
  resolved,e:=filepath.EvalSymlinks(path)
  if e!=nil{if name=="preset/manual.zh-CN.md"&&os.IsNotExist(e){continue};return k,e}
  if !within(pkg,resolved){return k,errors.New("资源文件超出软件包")}
  st,e:=os.Lstat(path);if e!=nil{return k,e};if !st.Mode().IsRegular(){return k,errors.New("资源不是普通文件: "+name)}
  f,e:=os.Open(resolved);if e!=nil{return k,e}
  b,e:=io.ReadAll(io.LimitReader(f,int64(kitMaxBytes-total+1)));f.Close()
  if e!=nil{return k,e}
  total+=len(b);if total>kitMaxBytes||len(b)==0||!utf8.Valid(b){return k,errors.New("资源大小或编码无效: "+name)}
  k.Files[name]=b
 }
 var meta struct{Name,Version string}
 if json.Unmarshal(k.Files["package.json"],&meta)!=nil||meta.Name!=s.Package||meta.Version!=version{return k,errors.New("资源包身份或版本不匹配")}
 var manifest struct{SchemaVersion int;ID,Language,LanguageID,Standard,HeaderFile string;SystemHeaders []string;Components []struct{ID,Code string;Symbols,Requires []string}}
 if json.Unmarshal(k.Files["preset/manifest.json"],&manifest)!=nil||manifest.SchemaVersion!=2||manifest.ID!="exam-io-c99"||manifest.Language!="c"||manifest.LanguageID!="c99"||manifest.Standard!="c99"||manifest.HeaderFile==""||manifest.SystemHeaders==nil||len(manifest.Components)==0||len(manifest.Components)>4096{return k,errors.New("Exam Kit manifest schema 2 不兼容")}
 deps:=map[string][]string{};symbols:=map[string]bool{}
 for _,c:=range manifest.Components{
  if c.ID==""||strings.TrimSpace(c.Code)==""||c.Symbols==nil||c.Requires==nil{return k,errors.New("资源组件不完整")}
  if _,ok:=deps[c.ID];ok{return k,errors.New("重复资源组件")}
  deps[c.ID]=c.Requires
  for _,sym:=range c.Symbols{if sym==""||symbols[sym]{return k,errors.New("重复或空资源符号")};symbols[sym]=true}
 }
 state:=map[string]int{};var visit func(string)error
 visit=func(id string)error{d,ok:=deps[id];if !ok||state[id]==1{return errors.New("无效资源依赖")};if state[id]==2{return nil};state[id]=1;for _,v:=range d{if e:=visit(v);e!=nil{return e}};state[id]=2;return nil}
 for id:=range deps{if e:=visit(id);e!=nil{return k,e}}
 var reference struct{SchemaVersion int;Version string;Categories,Sections []json.RawMessage}
 if json.Unmarshal(k.Files["preset/reference.json"],&reference)!=nil||reference.SchemaVersion!=1||reference.Version!=version||len(reference.Categories)==0||len(reference.Sections)==0{return k,errors.New("Exam Kit reference schema 1 不兼容")}
 return k,nil
}
func kitDigest(k kitExport)string{b,_:=json.Marshal(k);h:=sha256.Sum256(b);return hex.EncodeToString(h[:])}
func sealKit(snap string,s appSpec,version string)error{
 k,e:=readKitResource(snap,s,version);if e!=nil{return e}
 return atomicJSON(filepath.Join(snap,"resource-seal.json"),kitDigest(k))
}
func verifiedKit(snap string,s appSpec,version string)(kitExport,error){
 k,e:=readKitResource(snap,s,version);if e!=nil{return k,e}
 b,e:=os.ReadFile(filepath.Join(snap,"resource-seal.json"));if e!=nil{return k,e}
 var hash string
 if json.Unmarshal(b,&hash)!=nil||hash!=kitDigest(k){return k,errors.New("资源完整性校验失败，请修复安装")}
 return k,nil
}
// Copy a bounded, verified bundle under the same operation lock used by
// update and uninstall. Release the lock before writing to caller's pipe.
func(m *Manager) resourceData(id string)(kitExport,error){
 var k kitExport
 s,e:=spec(id);if e!=nil||!s.resource(){return k,errors.New("未知资源组件")}
 m.mu.Lock();defer m.mu.Unlock()
 done,e:=m.operationLock();if e!=nil{return k,e};defer done()
 if e=m.requireEnabled(s);e!=nil{return k,e}
 a,e:=m.active(s);if e!=nil{return k,fmt.Errorf("资源未安装或记录异常: %w",e)}
 return verifiedKit(filepath.Join(m.appDir(s),a.Snapshot),s,a.Version)
}
func(m *Manager) exportResource(id string,w io.Writer)error{k,e:=m.resourceData(id);if e!=nil{return e};return json.NewEncoder(w).Encode(k)}
