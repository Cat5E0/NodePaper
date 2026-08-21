//go:build !windows

package buildlock

import (
	"os"
	"syscall"
)

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks existence without sending an actual signal.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
