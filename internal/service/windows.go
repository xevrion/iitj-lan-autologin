package service

import (
	"fmt"
	"os"
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

func (w *WindowsTaskService) StatusInfo() (StatusInfo, error) {
	installed, err := w.IsInstalled()
	if err != nil {
		return StatusInfo{}, err
	}

	info := StatusInfo{
		ServiceManager: "Windows Task Scheduler",
		ServiceName:    windowsTaskName,
		Installed:      installed,
		Startup:        "not installed",
		LogHint:        "service log file",
	}
	if !installed {
		return info, nil
	}

	info.Startup = "at logon"
	out, err := exec.Command("schtasks", "/query", "/tn", windowsTaskName, "/fo", "list", "/v").CombinedOutput()
	if err != nil {
		return info, nil
	}

	props := parseWindowsList(string(out))
	status := strings.ToLower(props["Status"])
	info.Running = strings.Contains(status, "running")
	if logPath, err := backgroundLogPath(); err == nil {
		info.LogHint = logPath
	}
	if lastResult := props["Last Result"]; shouldShowWindowsLastResult(lastResult, info.Running) {
		info.LastExit = lastResult
	}

	return info, nil
}

func (w *WindowsTaskService) RecentLogs(lines int) ([]string, error) {
	path, err := backgroundLogPath()
	if err != nil {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	return trimLogLines(string(data), lines), nil
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

func parseWindowsList(s string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func trimLogLines(s string, lines int) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, lines)
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	if len(out) <= lines {
		return out
	}
	return out[len(out)-lines:]
}

func shouldShowWindowsLastResult(lastResult string, running bool) bool {
	lastResult = strings.TrimSpace(lastResult)
	if lastResult == "" || lastResult == "0" || lastResult == "The operation completed successfully." {
		return false
	}
	if running && (lastResult == "267009" || strings.EqualFold(lastResult, "0x41301")) {
		return false
	}
	return true
}
