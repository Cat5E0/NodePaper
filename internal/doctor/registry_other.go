//go:build !windows

package doctor

// readRegistryString has no meaning off Windows. checkInstallation returns
// before calling it there; this exists so the package still builds and its
// platform-independent tests still run on a Linux or macOS workstation.
func readRegistryString(subKey, valueName string) (string, bool) {
	return "", false
}
