//go:build windows

package export

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var getFinalPathNameByHandle = syscall.NewLazyDLL("kernel32.dll").NewProc("GetFinalPathNameByHandleW")

// canonicalExistingPath resolves both symbolic links and directory junctions.
// filepath.EvalSymlinks does not resolve every Windows reparse-point form, so
// use the final path of an open handle for containment decisions.
func canonicalExistingPath(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]uint16, 512)
	for {
		length, _, callErr := getFinalPathNameByHandle.Call(
			file.Fd(),
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(len(buffer)),
			0,
		)
		if length == 0 {
			return "", fmt.Errorf("GetFinalPathNameByHandleW: %w", callErr)
		}
		if length < uintptr(len(buffer)) {
			return ordinaryWindowsPath(syscall.UTF16ToString(buffer[:length])), nil
		}
		buffer = make([]uint16, int(length)+1)
	}
}

func ordinaryWindowsPath(path string) string {
	if strings.HasPrefix(path, `\\?\UNC\`) {
		return `\\` + strings.TrimPrefix(path, `\\?\UNC\`)
	}
	return strings.TrimPrefix(path, `\\?\`)
}
