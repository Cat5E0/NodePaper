//go:build !windows

package process

import (
	"os"
	"syscall"
)

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	child, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return child.Signal(syscall.Signal(0)) == nil
}
