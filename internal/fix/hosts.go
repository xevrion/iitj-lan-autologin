package fix

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	hostsEntry    = "172.17.0.3 gateway.iitj.ac.in"
	hostsMarker   = "gateway.iitj.ac.in"
	hostsComment  = "# Added by iitj-login — FortiGate captive portal bypass"
)

// AddHostsEntry adds `172.17.0.3 gateway.iitj.ac.in` to the system hosts file.
// This bypasses DNS entirely for all processes (curl, browser, GNOME portal popup)
// preventing the WiFi/Ethernet DNS race condition.
//
// Requires elevated privileges on all platforms (sudo on Linux/macOS, Admin on Windows).
func AddHostsEntry() error {
	hostsPath := hostsFilePath()

	// Check if entry already exists.
	if entryExists(hostsPath) {
		fmt.Println("  [hosts] gateway.iitj.ac.in entry already present — skipping")
		return nil
	}

	fmt.Printf("  [hosts] adding %s to %s (requires sudo)...\n", hostsEntry, hostsPath)

	line := hostsComment + "\n" + hostsEntry + "\n"

	switch runtime.GOOS {
	case "windows":
		return appendHostsWindows(hostsPath, line)
	default:
		return appendHostsUnix(hostsPath, line)
	}
}

// RemoveHostsEntry removes the iitj-login entry from the hosts file.
func RemoveHostsEntry() error {
	hostsPath := hostsFilePath()
	if !entryExists(hostsPath) {
		return nil
	}

	f, err := os.Open(hostsPath)
	if err != nil {
		return fmt.Errorf("open hosts: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, hostsMarker) || strings.Contains(line, "iitj-login") {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	f.Close()

	content := strings.Join(lines, "\n") + "\n"
	return writeHostsFile(hostsPath, []byte(content))
}

func hostsFilePath() string {
	if runtime.GOOS == "windows" {
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot == "" {
			systemRoot = `C:\Windows`
		}
		return systemRoot + `\System32\drivers\etc\hosts`
	}
	return "/etc/hosts"
}

func entryExists(hostsPath string) bool {
	f, err := os.Open(hostsPath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), hostsMarker) {
			return true
		}
	}
	return false
}

// HostsEntryPresent reports whether the captive portal hosts entry exists.
func HostsEntryPresent() bool {
	return entryExists(hostsFilePath())
}

func appendHostsUnix(hostsPath, content string) error {
	// Try direct write first (works if running as root).
	f, err := os.OpenFile(hostsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		_, err = fmt.Fprint(f, "\n"+content)
		if err == nil {
			fmt.Println("  [hosts] entry added.")
			return nil
		}
	}

	// Fall back to sudo tee -a.
	cmd := exec.Command("sudo", "tee", "-a", hostsPath)
	cmd.Stdin = strings.NewReader("\n" + content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo tee hosts: %w", err)
	}
	fmt.Println("  [hosts] entry added.")
	return nil
}

func appendHostsWindows(hostsPath, content string) error {
	// Try direct append first.
	f, err := os.OpenFile(hostsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		_, err = fmt.Fprint(f, "\r\n"+strings.ReplaceAll(content, "\n", "\r\n"))
		if err == nil {
			fmt.Println("  [hosts] entry added.")
			return nil
		}
	}

	// Fall back to PowerShell with elevation prompt.
	backtick := "`"
	entry := strings.ReplaceAll(content, "\n", backtick+"r"+backtick+"n")
	psCmd := fmt.Sprintf(
		"Start-Process powershell -Verb RunAs -Wait -ArgumentList '-NoProfile -Command \"Add-Content -Path \\\"%s\\\" -Value \\\"%s\\\"\"'",
		hostsPath, entry,
	)
	script := psCmd
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("powershell add hosts entry: %w", err)
	}
	fmt.Println("  [hosts] entry added.")
	return nil
}

func writeHostsFile(hostsPath string, content []byte) error {
	// Try direct write.
	if err := os.WriteFile(hostsPath, content, 0644); err == nil {
		return nil
	}
	// Fall back to sudo.
	if runtime.GOOS != "windows" {
		cmd := exec.Command("sudo", "tee", hostsPath)
		cmd.Stdin = strings.NewReader(string(content))
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}
	return fmt.Errorf("cannot write %s — please run as Administrator", hostsPath)
}
