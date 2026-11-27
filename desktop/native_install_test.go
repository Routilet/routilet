package main

import("os";"path/filepath";"runtime";"strings";"testing")

func TestNativeInstallLayout(t *testing.T){
 root:=t.TempDir();snap:=filepath.Join(root,"release-fixture")
 if e:=os.MkdirAll(snap,0700);e!=nil{t.Fatal(e)}
 store,e:=prepareNativeInstall(snap,root);if e!=nil{t.Fatal(e)}
 if runtime.GOOS!="windows"{if store!=filepath.Join(root,"store"){t.Fatal(store)};return}
 if store!=filepath.Join(root,"store-native"){t.Fatal(store)}
 data,e:=os.ReadFile(filepath.Join(snap,"pnpm-workspace.yaml"));if e!=nil{t.Fatal(e)}
 for _,s:=range []string{"nodeLinker: hoisted","packageImportMethod: copy","verifyStoreIntegrity: true","enableGlobalVirtualStore: false"}{if !strings.Contains(string(data),s){t.Fatal(string(data))}}
}
func TestMissingNativeEntryDiagnostics(t *testing.T){
 dir:=t.TempDir();p:=filepath.Join(dir,"missing.exe")
 if e:=os.WriteFile(filepath.Join(dir,"manifest-fixture"),nil,0600);e!=nil{t.Fatal(e)}
 if got:=describeEntryPath(p);!strings.Contains(got,"manifest-fixture"){t.Fatal(got)}
 if got:=describeEntryPath(filepath.Join(dir,"missing-dir","missing.exe"));!strings.Contains(got,"父目录"){t.Fatal(got)}
}
