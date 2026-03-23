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

func (s *SystemdService) StatusInfo() (StatusInfo, error) {
	installed, err := s.IsInstalled()
	if err != nil {
		return StatusInfo{}, err
	}

	info := StatusInfo{
		ServiceManager: "systemd user service",
		ServiceName:    systemdServiceName,
		Installed:      installed,
		Startup:        "not installed",
		LogHint:        "systemd user journal",
	}
	if !installed {
		return info, nil
	}

	out, err := exec.Command(
		"systemctl", "--user", "show", systemdServiceName,
		"--property=ActiveState,SubState,UnitFileState,ExecMainPID,Result",
		"--no-pager",
	).Output()
	if err != nil {
		info.Startup = "installed"
		info.Note = "live systemd state is unavailable outside a normal user session"
		return info, nil
	}

	props := parseKeyValueOutput(string(out))
	active := props["ActiveState"]
	subState := props["SubState"]
	info.Running = active == "active"

	if unitFileState := props["UnitFileState"]; unitFileState != "" {
		info.Startup = unitFileState
	}
	if subState != "" && subState != active {
		info.Startup = info.Startup + ", " + subState
	}
	if pid := props["ExecMainPID"]; pid != "" && pid != "0" {
		info.PID = pid
	}
	if result := props["Result"]; result != "" && result != "success" {
		info.LastExit = result
	}

	return info, nil
}

func (s *SystemdService) RecentLogs(lines int) ([]string, error) {
	out, err := exec.Command(
		"journalctl", "--user", "-u", systemdServiceName,
		"-n", fmt.Sprintf("%d", lines),
		"--no-pager", "-o", "cat",
	).CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return nil, nil
		}
	}
	return trimLogLines(string(out), lines), nil
}

func (s *SystemdService) IsInstalled() (bool, error) {
	path, err := s.servicePath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	return err == nil, nil
}

func parseKeyValueOutput(s string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[key] = value
	}
	return out
}
