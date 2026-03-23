//go:build !windows

package procutil

import "os/exec"

func Prepare(_ *exec.Cmd) {}
