package main

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
)
func TestVersions(t *testing.T){
 for _,c:=range []struct{a,b string;want bool}{{"0.2.0-beta.10","0.2.0-beta.2",true},{"0.2.0","0.2.0-beta.9",true},{"0.1.2","0.2.0-beta.1",false},{"0.2.0","0.2.0",false}}{
  if newer(c.a,c.b)!=c.want{t.Fatalf("newer(%s,%s)",c.a,c.b)}
 }
}
func TestNativeManifest(t *testing.T){
 if target()==""{t.Skip("target platform only")}
 root:=t.TempDir();s,_:=spec("observer")
 pkg:=filepath.Join(root,"node_modules",filepath.FromSlash(s.Package))
 if err:=os.MkdirAll(pkg,0700);err!=nil{t.Fatal(err)}
 payload:=[]byte("test native bytes");sum:=sha256.Sum256(payload)
 path:=filepath.Join(pkg,"native");if err:=os.WriteFile(path,payload,0700);err!=nil{t.Fatal(err)}
 manifest:=map[string]any{"schemaVersion":1,"version":"0.2.0-beta.1","platforms":map[string]any{target():map[string]string{"executable":"native","sha256":hex.EncodeToString(sum[:])}}}
 b,_:=json.Marshal(manifest);if err:=os.WriteFile(filepath.Join(pkg,"manifest.json"),b,0600);err!=nil{t.Fatal(err)}
 if _,err:=nativeEntry(root,s,"0.2.0-beta.1");err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(path,[]byte("modified"),0700);err!=nil{t.Fatal(err)}
 if _,err:=nativeEntry(root,s,"0.2.0-beta.1");err==nil{t.Fatal("accepted modified executable")}
}
