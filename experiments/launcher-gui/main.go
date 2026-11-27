package main

import (
 "embed"
 "fmt"
 "io/fs"
 "os"
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
func (d *Desktop) Close() { d.native.Quit() }

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
 })
 d.native=app
 app.Window.NewWithOptions(application.WebviewWindowOptions{Title:"Routilet",Width:420,Height:330,MinWidth:360,MinHeight:280,URL:"/"})
 return app.Run()
}
func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}
