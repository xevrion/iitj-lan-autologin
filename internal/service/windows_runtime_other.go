//go:build !windows

package service

import "fmt"

func PrepareBackgroundProcess(command string) {}

func backgroundLogPath() (string, error) {
	return "", fmt.Errorf("windows log path is only available on windows")
}
