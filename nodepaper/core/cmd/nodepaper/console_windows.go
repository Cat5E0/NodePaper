//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
)

// ownsConsoleWindow reports whether this process is the only process attached
// to its console. Windows then created that console window for nodepaper.exe
// itself, which happens when the executable is started from File Explorer
// instead of an existing terminal: the window is destroyed as soon as the
// process exits, so the output would never be readable.
//
// A terminal, shell script, pipeline or Start-menu launcher always keeps at
// least the parent shell attached, and a process without a console gets 0.
func ownsConsoleWindow() bool {
	var processIDs [8]uint32
	attached, _, _ := procGetConsoleProcessList.Call(
		uintptr(unsafe.Pointer(&processIDs[0])),
		uintptr(len(processIDs)),
	)
	return attached == 1
}
