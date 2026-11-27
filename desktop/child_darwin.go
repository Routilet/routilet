//go:build darwin

package main
import "os/exec"
func hideChild(cmd *exec.Cmd){}
