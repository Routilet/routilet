package main

import (
 "bytes"
 "context"
 "errors"
 "regexp"
 "strconv"
 "strings"
 "sync"
)

type Progress struct {
 ID string `json:"id"`
 Stage string `json:"stage"`
 Added int `json:"added"`
 Total int `json:"total"`
 Resolved int `json:"resolved"`
 Downloaded int `json:"downloaded"`
 DownloadBytes int64 `json:"downloadBytes"`
 DownloadTotal int64 `json:"downloadTotal"`
 DownloadName string `json:"downloadName"`
 Reused int `json:"reused"`
 Active bool `json:"active"`
 Cancellable bool `json:"cancellable"`
 Cancelled bool `json:"cancelled"`
}

type installJob struct {
 ctx context.Context
 cancel context.CancelFunc
 mu sync.Mutex
 p Progress
}

func (m *Manager) beginJob(id string)(*installJob,error){
 m.jobMu.Lock();defer m.jobMu.Unlock()
 if m.job!=nil{return nil,errors.New("已有安装任务")}
 ctx,cancel:=context.WithCancel(context.Background())
 j:=&installJob{ctx:ctx,cancel:cancel,p:Progress{ID:id,Stage:"检查版本",Active:true,Cancellable:true}}
 m.job=j
 return j,nil
}
func (m *Manager) endJob(j *installJob){
 j.cancel()
 m.jobMu.Lock();defer m.jobMu.Unlock()
 if m.job==j{m.job=nil}
}
func (m *Manager) progress() Progress{
 m.jobMu.Lock();defer m.jobMu.Unlock()
 if m.job==nil{return Progress{}}
 m.job.mu.Lock();defer m.job.mu.Unlock()
 return m.job.p
}
func (m *Manager) cancelJob()bool{
 m.jobMu.Lock();defer m.jobMu.Unlock()
 if m.job==nil{return false}
 j:=m.job;j.mu.Lock();defer j.mu.Unlock()
 if !j.p.Cancellable{return false}
 j.p.Cancelled=true;j.p.Cancellable=false;j.p.Stage="正在取消"
 j.cancel()
 return true
}
func (j *installJob) stage(s string){
 j.mu.Lock();defer j.mu.Unlock()
 if !j.p.Cancelled{j.p.Stage=s;j.p.Total=0;j.p.Added=0;j.p.Resolved=0;j.p.Downloaded=0;j.p.Reused=0;j.p.DownloadBytes=0;j.p.DownloadTotal=0;j.p.DownloadName=""}
}
// Serialize the cancellation decision with activation. Cancellation after
// this point is rejected; the UI must report completion, not cancellation.
func (j *installJob) commit()error{
 j.mu.Lock();defer j.mu.Unlock()
 if err:=j.ctx.Err();err!=nil{return err}
 j.p.Cancellable=false;j.p.Stage="激活版本";j.p.Total=0;j.p.DownloadBytes=0;j.p.DownloadTotal=0;j.p.DownloadName=""
 return nil
}
// pnpm 12.22.0 append-only emits decimal, rounded sizes for large tarballs.
var downloadPattern=regexp.MustCompile(`^Downloading ([^ ]+): ([0-9]+(?:\.[0-9]+)?) (B|kB|MB|GB)/([0-9]+(?:\.[0-9]+)?) (B|kB|MB|GB)(, done)?$`)
func downloadSize(number,unit string) int64 {
 n,e:=strconv.ParseFloat(number,64);if e!=nil||n<0{return -1}
 factor:=map[string]float64{"B":1,"kB":1000,"MB":1000000,"GB":1000000000}[unit]
 if factor==0||n*factor>1e15{return -1}
 return int64(n*factor)
}
var totalPattern=regexp.MustCompile(`^Packages:\s+\+([0-9]+)\s*$`)
var countPattern=regexp.MustCompile(`(?:^|[ ,])(?:resolved|reused|downloaded|added) [0-9]+`)
func (j *installJob) line(line string){
 j.mu.Lock();defer j.mu.Unlock()
 if j.p.Cancelled{return}
 line=strings.TrimSpace(line)
 if m:=downloadPattern.FindStringSubmatch(line);len(m)>0 {
  if j.p.Stage!="解析与下载"&&j.p.Stage!="下载与安装"{return}
  // Do not combine concurrent files or pretend to measure the entire install.
  if j.p.DownloadName!=""&&j.p.DownloadName!=m[1]{return}
  downloaded,total:=downloadSize(m[2],m[3]),downloadSize(m[4],m[5])
  if downloaded<0||total<=0||downloaded>total{return}
  if m[6]!=""{j.p.DownloadBytes=0;j.p.DownloadTotal=0;j.p.DownloadName="";return}
  j.p.DownloadName=m[1];j.p.DownloadBytes=downloaded;j.p.DownloadTotal=total
  j.p.Stage="下载与安装"
  return
 }
 if m:=totalPattern.FindStringSubmatch(line);len(m)>0{
  j.p.Total,_=strconv.Atoi(m[1]);return
 }
 if !strings.HasPrefix(line,"Progress:"){return}
 j.p.Stage="下载与安装"
 for _,part:=range countPattern.FindAllString(line,-1){
  f:=strings.Fields(strings.TrimLeft(part,", "));if len(f)!=2{continue}
  n,e:=strconv.Atoi(f[1]);if e!=nil{continue}
  switch f[0]{case "resolved":j.p.Resolved=n;case "reused":j.p.Reused=n;case "downloaded":j.p.Downloaded=n;case "added":j.p.Added=n}
 }
}
// os/exec serializes writes when stdout/stderr are the same comparable writer.
// Retain the tail for diagnostic errors, not an unbounded installation log.
type progressWriter struct{output *bytes.Buffer;pending []byte;job *installJob}
func(w *progressWriter) Write(b []byte)(int,error){
 n:=len(b)
 _,_=w.output.Write(b)
 if w.output.Len()>65536{w.output.Next(w.output.Len()-65536)}
 if w.job!=nil{
  w.pending=append(w.pending,b...)
  for{
   i:=bytes.IndexAny(w.pending,"\r\n");if i<0{break}
   w.job.line(string(w.pending[:i]));w.pending=w.pending[i+1:]
  }
  if len(w.pending)>65536{w.pending=nil}
 }
 return n,nil
}
func(w *progressWriter) finish(){if w.job!=nil&&len(w.pending)>0{w.job.line(string(w.pending))}}
