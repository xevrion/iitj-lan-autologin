package service

import (
	"fmt"
	"os/exec"
	"strings"
)

const windowsTaskName = "IITJ-LAN-AutoLogin"

// WindowsTaskService manages a Windows Task Scheduler task.
type WindowsTaskService struct{}

// Install creates a scheduled task that runs at logon.
func (w *WindowsTaskService) Install(execPath string) error {
	// Delete any existing task first.
	exec.Command("schtasks", "/delete", "/tn", windowsTaskName, "/f").Run()

	cmd := exec.Command("schtasks", "/create",
		"/tn", windowsTaskName,
		"/tr", fmt.Sprintf(`"%s" login`, execPath),
		"/sc", "onlogon",
		"/rl", "limited",
		"/f",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks create: %w: %s", err, strings.TrimSpace(string(out)))
	}

	// Run immediately.
	return w.Start()
}

// Uninstall deletes the scheduled task.
func (w *WindowsTaskService) Uninstall() error {
	out, err := exec.Command("schtasks", "/delete", "/tn", windowsTaskName, "/f").CombinedOutput()
	if err != nil && !strings.Contains(string(out), "does not exist") {
		return fmt.Errorf("schtasks delete: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (w *WindowsTaskService) Start() error {
	out, err := exec.Command("schtasks", "/run", "/tn", windowsTaskName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks run: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (w *WindowsTaskService) Stop() error {
	out, err := exec.Command("schtasks", "/end", "/tn", windowsTaskName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks end: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (w *WindowsTaskService) Status() (string, error) {
	out, err := exec.Command("schtasks", "/query", "/tn", windowsTaskName, "/fo", "list").CombinedOutput()
	if err != nil {
		return fmt.Sprintf("task not found: %s", strings.TrimSpace(string(out))), nil
	}
	return string(out), nil
}

func (w *WindowsTaskService) IsInstalled() (bool, error) {
	out, err := exec.Command("schtasks", "/query", "/tn", windowsTaskName).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "does not exist") {
			return false, nil
		}
		return false, fmt.Errorf("schtasks query: %w", err)
	}
	return true, nil
}
