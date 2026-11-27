//go:build !windows

package main

import (
	"os/exec"
	"runtime"
)

func openGuide(file string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "/usr/bin/open"
	}
	return exec.Command(command, file).Run()
}
