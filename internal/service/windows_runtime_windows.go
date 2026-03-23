//go:build windows

package service

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/iitj/iitj-lan-autologin/internal/creds"
)

const windowsServiceLogFile = "service.log"

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procFreeConsole      = kernel32.NewProc("FreeConsole")
	procShowWindow       = user32.NewProc("ShowWindow")
)

func PrepareBackgroundProcess(command string) {
	if command != "login" {
		return
	}

	redirectWindowsLogs()
	hideWindowsConsole()
}

func backgroundLogPath() (string, error) {
	dir, err := creds.DataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, windowsServiceLogFile), nil
}

func redirectWindowsLogs() {
	path, err := backgroundLogPath()
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	os.Stdout = f
	os.Stderr = f
}

func hideWindowsConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}

	const swHide = 0
	procShowWindow.Call(hwnd, swHide)
	procFreeConsole.Call()
}
