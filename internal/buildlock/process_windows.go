//go:build windows

package buildlock

import "syscall"

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	// OpenProcess with minimal access right to check if the process exists.
	const processQueryLimitedInfo = 0x1000
	handle, err := syscall.OpenProcess(processQueryLimitedInfo, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(handle)
	return true
}
