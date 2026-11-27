package main

import "golang.org/x/sys/windows"

func openGuide(file string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	value, err := windows.UTF16PtrFromString(file)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, value, nil, nil, 1)
}
