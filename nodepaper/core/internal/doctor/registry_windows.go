package doctor

import (
	"strings"
	"syscall"
	"unsafe"
)

// readRegistryString reads a string value from HKEY_CURRENT_USER and reports
// whether it was present. A missing key, a missing value or a value of another
// type are all "not present" and are never surfaced as an error: this is a
// probe of an optional installation record, and an absent record is the normal
// state for a machine that uses only one channel.
//
// The registry is read through syscall rather than by running reg.exe. reg.exe
// prints in the console code page, so an installation directory with non-ASCII
// characters - the Chinese paths this project is built for - comes back
// mojibake and would be reported to the user that way, or compared against a
// correctly spelled path and found different. The API returns UTF-16, which
// leaves nothing to guess, and it also keeps doctor from spawning a process for
// a single value.
func readRegistryString(subKey, valueName string) (string, bool) {
	subKeyPtr, err := syscall.UTF16PtrFromString(subKey)
	if err != nil {
		return "", false
	}
	var handle syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, subKeyPtr, 0, syscall.KEY_QUERY_VALUE, &handle); err != nil {
		return "", false
	}
	defer syscall.RegCloseKey(handle)

	valueNamePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return "", false
	}
	var valueType, size uint32
	if err := syscall.RegQueryValueEx(handle, valueNamePtr, nil, &valueType, nil, &size); err != nil {
		return "", false
	}
	if valueType != syscall.REG_SZ && valueType != syscall.REG_EXPAND_SZ {
		return "", false
	}
	if size == 0 {
		return "", false
	}
	// size is in bytes and includes the terminating NUL; the extra element
	// keeps an unterminated value from running past the buffer.
	buffer := make([]uint16, size/2+1)
	if err := syscall.RegQueryValueEx(handle, valueNamePtr, nil, &valueType, (*byte)(unsafe.Pointer(&buffer[0])), &size); err != nil {
		return "", false
	}
	value := strings.TrimSpace(syscall.UTF16ToString(buffer))
	return value, value != ""
}
