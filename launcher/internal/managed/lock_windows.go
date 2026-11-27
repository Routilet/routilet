//go:build windows

package managed

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var lockFileEx = kernel32.NewProc("LockFileEx")
var unlockFileEx = kernel32.NewProc("UnlockFileEx")

func lockChannel(dir string) (func(), error) {
	return lockPath(filepath.Join(dir, "write.lock"), true)
}

func lockPath(path string, exclusive bool) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	var ov syscall.Overlapped
	// LOCKFILE_FAIL_IMMEDIATELY, optionally LOCKFILE_EXCLUSIVE_LOCK.
	flags := uintptr(1)
	if exclusive {
		flags |= 2
	}
	ok, _, err := lockFileEx.Call(f.Fd(), flags, 0, 1, 0, uintptr(unsafe.Pointer(&ov)))
	if ok == 0 {
		f.Close()
		return nil, err
	}
	return func() {
		var ov syscall.Overlapped
		_, _, _ = unlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&ov)))
		_ = f.Close()
	}, nil
}
