package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	systemdServiceName = "iitj-login"
	systemdServiceFile = systemdServiceName + ".service"
)

// SystemdService manages a systemd user service (Linux).
type SystemdService struct{}

func (s *SystemdService) serviceDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user"), nil
}

func (s *SystemdService) servicePath() (string, error) {
	dir, err := s.serviceDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, systemdServiceFile), nil
}

// Install writes the systemd unit file and enables + starts the service.
func (s *SystemdService) Install(execPath string) error {
	dir, err := s.serviceDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create systemd user dir: %w", err)
	}

	unit := fmt.Sprintf(`[Unit]
Description=IITJ LAN Auto Login
After=network-online.target

[Service]
ExecStart=%s login
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
`, execPath)

	path, err := s.servicePath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(unit), 0644); err != nil {
		return fmt.Errorf("write service file: %w", err)
	}

	// Enable linger so the service runs without an active login session.
	exec.Command("loginctl", "enable-linger").Run()

	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := exec.Command("systemctl", "--user", "enable", systemdServiceName).Run(); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}
	if err := exec.Command("systemctl", "--user", "start", systemdServiceName).Run(); err != nil {
		return fmt.Errorf("start service: %w", err)
	}

	return nil
}

// Uninstall stops, disables, and removes the service unit file.
func (s *SystemdService) Uninstall() error {
	exec.Command("systemctl", "--user", "stop", systemdServiceName).Run()
	exec.Command("systemctl", "--user", "disable", systemdServiceName).Run()

	path, err := s.servicePath()
	if err != nil {
		return err
	}
	os.Remove(path)

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

func (s *SystemdService) Start() error {
	return exec.Command("systemctl", "--user", "start", systemdServiceName).Run()
}

func (s *SystemdService) Stop() error {
	return exec.Command("systemctl", "--user", "stop", systemdServiceName).Run()
}

func (s *SystemdService) Status() (string, error) {
	out, err := exec.Command("systemctl", "--user", "status", systemdServiceName, "--no-pager").CombinedOutput()
	// systemctl status exits 3 if inactive — still print output.
	if err != nil && len(out) == 0 {
		return "", err
	}
	return string(out), nil
}

func (s *SystemdService) IsInstalled() (bool, error) {
	out, err := exec.Command("systemctl", "--user", "list-unit-files", "--no-pager").Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), systemdServiceFile), nil
}
