package main
import "testing"
func TestDownloadProgressFromPinnedReporter(t *testing.T) {
 j:=&installJob{p:Progress{Stage:"解析与下载"}}
 j.line("Progress: resolved 1, reused 0, downloaded 0, added 0")
 if j.p.DownloadTotal!=0{t.Fatal("package counts must not create byte progress")}
 j.line("Downloading registry.npmjs.org/@routilet/companion/0.2.0: 0.00 B/20.97 MB")
 j.line("Downloading registry.npmjs.org/@routilet/companion/0.2.0: 5.76 MB/20.97 MB")
 if j.p.DownloadBytes!=5760000||j.p.DownloadTotal!=20970000{t.Fatalf("download: %+v",j.p)}
 j.line("Downloading registry.npmjs.org/other/1.0.0: 7.00 MB/10.00 MB")
 if j.p.DownloadBytes!=5760000{t.Fatal("interleaved transfer must not overwrite active transfer")}
 j.line("Downloading registry.npmjs.org/@routilet/companion/0.2.0: 20.97 MB/20.97 MB, done")
 if j.p.DownloadTotal!=0{t.Fatal("finished download must disappear")}
 j.stage("校验应用")
 j.line("Downloading x: 5.00 MB/10.00 MB")
 if j.p.DownloadTotal!=0{t.Fatal("stale output during validation")}
}
