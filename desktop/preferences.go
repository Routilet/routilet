package main

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
)

var errAppDisabled=errors.New("应用已禁用，请先在 Routilet 中启用")
func (m *Manager) disabled(s appSpec)(bool,error){
 if s.ID=="routilet"{return false,nil}
 p:=filepath.Join(m.appDir(s),"disabled.json")
 st,e:=os.Stat(p)
 if os.IsNotExist(e){return false,nil}
 if e!=nil{return false,fmt.Errorf("读取应用禁用状态 %s: %w",p,e)}
 if !st.Mode().IsRegular(){return false,fmt.Errorf("应用禁用记录无效: %s",p)}
 return true,nil
}
func (m *Manager) requireEnabled(s appSpec)error{
 disabled,e:=m.disabled(s);if e!=nil{return e};if disabled{return errAppDisabled};return nil
}
func (m *Manager) setDisabled(id string,disabled bool)error{
 if id=="routilet"{return errors.New("不能禁用 Routilet 自身")}
 s,e:=spec(id);if e!=nil{return e}
 m.mu.Lock();defer m.mu.Unlock()
 done,e:=m.operationLock();if e!=nil{return e};defer done()
 dir:=m.appDir(s);p:=filepath.Join(dir,"disabled.json")
 if disabled{
  if e=os.MkdirAll(dir,0700);e!=nil{return fmt.Errorf("创建应用偏好目录 %s: %w",dir,e)}
  return atomicJSON(p,true)
 }
 if e=os.Remove(p);e!=nil&&!os.IsNotExist(e){return fmt.Errorf("移除禁用记录 %s: %w",p,e)}
 return nil
}
