package main

import (
 "fmt"
 "os"
 "path/filepath"
 "runtime"
)
func bundledPNPM(bootstrap string) string {
 dir:=filepath.Dir(bootstrap)
 if runtime.GOOS!="windows"{return filepath.Join(dir,"pnpm")}
 preferred:=filepath.Join(dir,"runtime","pnpm.exe")
 // Preserve existing flat distributions. Do not silently fall back from an
 // unreadable or malformed runtime file to an unrelated executable.
 if _,e:=os.Stat(preferred);!os.IsNotExist(e){return preferred}
 legacy:=filepath.Join(dir,"pnpm.exe")
 if _,e:=os.Stat(legacy);!os.IsNotExist(e){return legacy}
 return preferred
}
func checkPNPM(path string)error{
 st,e:=os.Stat(path);if e!=nil{return fmt.Errorf("准备安装：找不到或无法访问随附 pnpm（%s），请完整解压发行包: %w",path,e)}
 if !st.Mode().IsRegular(){return fmt.Errorf("准备安装：随附 pnpm 不是普通文件（%s）",path)}
 return nil
}
