package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Service is the interface for platform-specific daemon management.
type Service interface {
	Install(execPath string) error
	Uninstall() error
	Start() error
	Stop() error
	Status() (string, error)
	StatusInfo() (StatusInfo, error)
	RecentLogs(lines int) ([]string, error)
	IsInstalled() (bool, error)
}

// New returns the appropriate Service implementation for the current platform.
func New() Service {
	switch runtime.GOOS {
	case "linux":
		return &SystemdService{}
	case "darwin":
		return &LaunchdService{}
	case "windows":
		return &WindowsTaskService{}
	default:
		return &UnsupportedService{os: runtime.GOOS}
	}
}

// ExecPath returns the absolute path of the current executable.
func ExecPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine executable path: %w", err)
	}
	return filepath.Abs(exe)
}

// UnsupportedService is returned for unknown platforms.
type UnsupportedService struct{ os string }

func (u *UnsupportedService) Install(_ string) error     { return u.err() }
func (u *UnsupportedService) Uninstall() error           { return u.err() }
func (u *UnsupportedService) Start() error               { return u.err() }
func (u *UnsupportedService) Stop() error                { return u.err() }
func (u *UnsupportedService) Status() (string, error)    { return "", u.err() }
func (u *UnsupportedService) StatusInfo() (StatusInfo, error) {
	return StatusInfo{}, u.err()
}
func (u *UnsupportedService) RecentLogs(_ int) ([]string, error) { return nil, u.err() }
func (u *UnsupportedService) IsInstalled() (bool, error) { return false, u.err() }
func (u *UnsupportedService) err() error {
	return fmt.Errorf("service management not supported on %s", u.os)
}
