package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	launchdLabel   = "ac.iitj.login"
	launchdPlist   = launchdLabel + ".plist"
	launchdLogFile = "/tmp/iitj-login.log"
)

// LaunchdService manages a launchd user agent (macOS).
type LaunchdService struct{}

func (l *LaunchdService) agentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

func (l *LaunchdService) plistPath() (string, error) {
	dir, err := l.agentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, launchdPlist), nil
}

// Install writes the launchd plist and loads it.
func (l *LaunchdService) Install(execPath string) error {
	dir, err := l.agentsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>login</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, launchdLabel, execPath, launchdLogFile, launchdLogFile)

	path, err := l.plistPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	// Unload first in case a stale agent exists.
	exec.Command("launchctl", "unload", path).Run()

	if err := exec.Command("launchctl", "load", path).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}

	return nil
}

// Uninstall unloads and removes the plist.
func (l *LaunchdService) Uninstall() error {
	path, err := l.plistPath()
	if err != nil {
		return err
	}
	exec.Command("launchctl", "unload", path).Run()
	os.Remove(path)
	return nil
}

func (l *LaunchdService) Start() error {
	return exec.Command("launchctl", "start", launchdLabel).Run()
}

func (l *LaunchdService) Stop() error {
	return exec.Command("launchctl", "stop", launchdLabel).Run()
}

func (l *LaunchdService) Status() (string, error) {
	out, err := exec.Command("launchctl", "list", launchdLabel).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("not loaded: %s", strings.TrimSpace(string(out))), nil
	}
	return string(out), nil
}

func (l *LaunchdService) IsInstalled() (bool, error) {
	path, err := l.plistPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	return err == nil, nil
}
