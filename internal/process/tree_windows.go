//go:build windows

package process

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
)

func configureProcessTreeCancellation(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}

		// taskkill /T terminates descendants created by PowerShell, Pandoc,
		// latexmk and XeLaTeX. Process.Kill alone only stops the direct child.
		killer := exec.Command("taskkill.exe", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
		if err := killer.Run(); err == nil {
			return nil
		}

		// Fall back to the direct process so cancellation still makes progress
		// when taskkill is unavailable. The process-tree test detects this loss.
		err := cmd.Process.Kill()
		if errors.Is(err, os.ErrProcessDone) {
			return os.ErrProcessDone
		}
		return err
	}
}
