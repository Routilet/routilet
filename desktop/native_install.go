package main

import (
 "fmt"
 "os"
 "path/filepath"
 "runtime"
 "strings"
)

// Native packages do not need pnpm's linked dependency topology on Windows.
// Keep a separate store so previously populated data is not reused by this
// new layout. Never delete existing caches or snapshots.
func prepareNativeInstall(snap,root string)(string,error){
 if runtime.GOOS!="windows"{return filepath.Join(root,"store"),nil}
 p:=filepath.Join(snap,"pnpm-workspace.yaml")
 config:="nodeLinker: hoisted\npackageImportMethod: copy\nverifyStoreIntegrity: true\nenableGlobalVirtualStore: false\n"
 if err:=os.WriteFile(p,[]byte(config),0600);err!=nil{return "",fmt.Errorf("写入原生安装配置 %s: %w",p,err)}
 return filepath.Join(root,"store-native"),nil
}

// Inspect only the candidate package's declared entry and its immediate
// ancestors. Keep this bounded and do not enumerate unrelated user files.
func describeEntryPath(entry string)string{
 st,e:=os.Stat(entry)
 if e==nil{return fmt.Sprintf("入口 Stat 成功（普通文件=%t，大小=%d），但路径解析失败",st.Mode().IsRegular(),st.Size())}
 dir:=filepath.Dir(entry)
 entries,dirErr:=os.ReadDir(dir)
 if dirErr!=nil{return fmt.Sprintf("入口 Stat: %v；父目录 %s: %v",e,dir,dirErr)}
 names:=make([]string,0,12)
 for _,item:=range entries{if len(names)==12{names=append(names,"…");break};names=append(names,item.Name())}
 return fmt.Sprintf("入口 Stat: %v；父目录存在，包含 [%s]",e,strings.Join(names,", "))
}
