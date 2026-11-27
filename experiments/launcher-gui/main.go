package main

import (
 "embed"
 "fmt"
 "io/fs"
 "os"
 "os/exec"
 "runtime"
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
 if len(os.Args)==2 && os.Args[1]=="--self-test" { return selfTest() }
 if len(os.Args)!=1 {return fmt.Errorf("unsupported arguments")}
 m,err:=newManager();if err!=nil{return err}
 d:=&Desktop{manager:m}
 frontend,err:=fs.Sub(assets,"frontend");if err!=nil{return err}
 app:=application.New(application.Options{
  Name:"Routilet Preview", Description:"Application launcher preview",
  Services:[]application.Service{application.NewService(d)},
  Assets:application.AssetOptions{Handler:application.BundledAssetFileServer(frontend)},
  Mac:application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed:true},
  ShouldQuit:func()bool{return !m.progress().Active},
  OnShutdown:func(){m.cancelJob()},
 })
 d.native=app
 window:=app.Window.NewWithOptions(application.WebviewWindowOptions{Title:"Routilet",Width:420,Height:330,MinWidth:360,MinHeight:280,URL:"/"})
 window.RegisterHook(events.Common.WindowClosing,func(e *application.WindowEvent){if m.progress().Active{e.Cancel()}})
 return app.Run()
}
func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}
