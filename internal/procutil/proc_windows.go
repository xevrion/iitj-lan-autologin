//go:build windows

package procutil

import (
	"os/exec"
	"syscall"
)

func Prepare(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
