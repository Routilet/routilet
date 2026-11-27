package main

import (
 "embed"
 "fmt"
 "io/fs"
 "os"
 "os/exec"
 "runtime"
 "time"
 "github.com/wailsapp/wails/v3/pkg/events"
 "github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/*
var assets embed.FS

type Desktop struct {
 manager *Manager
 native *application.App
}

func (d *Desktop) List() []AppState { return d.manager.list() }
func (d *Desktop) Check() []AppState { return d.manager.check() }
func (d *Desktop) Install(id string) error { return d.manager.install(id) }
func (d *Desktop) Repair(id string) error{return d.manager.installMode(id,true)}
func (d *Desktop) Rollback(id string) error{return d.manager.rollback(id)}
func (d *Desktop) Uninstall(id string) error{return d.manager.uninstall(id)}
func (d *Desktop) Shortcut(id string) error{return d.manager.shortcut(id)}
func (d *Desktop) Clean() error{return d.manager.clean()}
func (d *Desktop) SelfInfo() SelfInfo{return d.manager.selfInfo()}
func (d *Desktop) Channel()(string,error){return d.manager.channel()}
func (d *Desktop) SetChannel(channel string)error{return d.manager.setChannel(channel)}
func (d *Desktop) UpdateSelf() error{return d.manager.install("routilet")}
func (d *Desktop) Ready() error{if d.manager.ready==""{return nil};return os.WriteFile(d.manager.ready,[]byte("ready"),0600)}
func (d *Desktop) Launch(id string) error { return d.manager.launch(id) }
func (d *Desktop) Close() { if !d.manager.progress().Active{d.native.Quit()} }
func (d *Desktop) Progress() Progress {return d.manager.progress()}
func (d *Desktop) Cancel() bool {return d.manager.cancelJob()}
func (d *Desktop) OpenData() error {
 if err:=os.MkdirAll(d.manager.root,0700);err!=nil{return err}
 if runtime.GOOS=="darwin"{return exec.Command("/usr/bin/open",d.manager.root).Run()}
 cmd:=exec.Command("explorer.exe",d.manager.root);if err:=cmd.Start();err!=nil{return err};return cmd.Process.Release()
}

func run() error {
 if len(os.Args)==2&&os.Args[1]=="--app-call-probe"{fmt.Println(appCallProtocol);return nil}
 if len(os.Args)==2&&os.Args[1]=="--manager-probe"{fmt.Println("routilet-manager/1 "+buildVersion);return nil}
 if len(os.Args)==2 && os.Args[1]=="--self-test" { return selfTest() }
 if len(os.Args)==4&&os.Args[1]=="--self-update-test"{return selfUpdateTest(os.Args[2],os.Args[3])}
 m,err:=newManager();if err!=nil{return err}
 if len(os.Args)>1&&(os.Args[1]=="--open-app"||os.Args[1]=="--query-app"){
  c,e:=parseAppCall(os.Args[1:]);if e!=nil{return e}
  if p:=os.Getenv("ROUTILET_BOOTSTRAP");p!=""{if e=m.setBootstrap(p);e!=nil{return e}}
  return m.callApp(c,os.Stdout,os.Stderr)
 }
 if len(os.Args)>=6&&os.Args[1]=="--supervise"{
  var args []string
  if len(os.Args)>6{if os.Args[6]!="--"||len(os.Args)==7{return fmt.Errorf("invalid supervisor arguments")};args=os.Args[7:]}
  if p:=os.Getenv("ROUTILET_BOOTSTRAP");p!=""{if err=m.setBootstrap(p);err!=nil{return err}}
  return m.supervise(os.Args[2],os.Args[3],os.Args[4],os.Args[5],args...)
 }
 if len(os.Args)==4&&os.Args[1]=="--managed"{
  if err=m.setBootstrap(os.Args[2]);err!=nil{return err};m.ready=os.Args[3]
  if fixtureMode=="ready"{if err=os.WriteFile(m.ready,[]byte("ready"),0600);err!=nil{return err};time.Sleep(750*time.Millisecond);return nil}
 }else if len(os.Args)==1{if m.dispatch(){return nil}}else{return fmt.Errorf("unsupported arguments")}
 d:=&Desktop{manager:m}
 frontend,err:=fs.Sub(assets,"frontend");if err!=nil{return err}
 app:=application.New(application.Options{
  Name:"Routilet", Description:"Routilet application launcher",
  Services:[]application.Service{application.NewService(d)},
  Assets:application.AssetOptions{Handler:application.BundledAssetFileServer(frontend)},
  Mac:application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed:true},
  ShouldQuit:func()bool{return !m.progress().Active},
  OnShutdown:func(){m.cancelJob()},
 })
 if icon,e:=assets.ReadFile("frontend/icons/routilet.png");e==nil{app.SetIcon(icon)}
 d.native=app
 window:=app.Window.NewWithOptions(application.WebviewWindowOptions{Title:"Routilet",Width:420,Height:330,MinWidth:360,MinHeight:280,URL:"/"})
 window.RegisterHook(events.Common.WindowClosing,func(e *application.WindowEvent){if m.progress().Active{e.Cancel()}})
 return app.Run()
}
func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(appExitCode(err))}}
