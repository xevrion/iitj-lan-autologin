package fix

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// DisableMACRandomization disables MAC address randomization on the ethernet
// interface. FortiGate authenticates sessions by MAC address, so randomization
// breaks session persistence after re-connects.
//
// nmConn is the NetworkManager connection name (Linux only, may be "").
func DisableMACRandomization(ifaceName, nmConn string) error {
	switch runtime.GOOS {
	case "linux":
		return disableMACLinux(nmConn)
	case "darwin":
		return disableMACDarwin(ifaceName)
	case "windows":
		return disableMACWindows(ifaceName)
	}
	return nil
}

func disableMACLinux(nmConn string) error {
	if nmConn == "" {
		fmt.Println("  [MAC] nmcli connection not found — skipping MAC randomization fix")
		return nil
	}

	// Check if nmcli is available.
	if _, err := exec.LookPath("nmcli"); err != nil {
		fmt.Println("  [MAC] nmcli not found — skipping (not needed on non-NM distros)")
		return nil
	}

	out, err := exec.Command("nmcli", "connection", "modify", nmConn,
		"ethernet.cloned-mac-address", "permanent").CombinedOutput()
	if err != nil {
		return fmt.Errorf("nmcli modify MAC: %w: %s", err, strings.TrimSpace(string(out)))
	}
	fmt.Printf("  [MAC] randomization disabled for connection: %s\n", nmConn)
	return nil
}

func disableMACDarwin(ifaceName string) error {
	// macOS does not randomize MAC addresses on ethernet by default.
	// Wi-Fi randomization is handled separately and not relevant here.
	fmt.Printf("  [MAC] macOS ethernet (%s) does not randomize MACs — nothing to do\n", ifaceName)
	return nil
}

func disableMACWindows(ifaceName string) error {
	// Check if MAC randomization is enabled for this adapter.
	script := fmt.Sprintf(
		`(Get-NetAdapter -Name '%s' -ErrorAction SilentlyContinue).MacAddressSeed`,
		ifaceName,
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return nil // adapter not found or PowerShell failed — skip
	}
	seed := strings.TrimSpace(string(out))
	if seed == "" {
		fmt.Printf("  [MAC] Windows ethernet (%s) MAC randomization is already off\n", ifaceName)
		return nil
	}

	// Disable randomization via registry or Set-NetAdapterAdvancedProperty.
	disableScript := fmt.Sprintf(
		`Set-NetAdapterAdvancedProperty -Name '%s' -RegistryKeyword 'RandomizeMACAddress' -RegistryValue 0 -ErrorAction SilentlyContinue`,
		ifaceName,
	)
	exec.Command("powershell", "-NoProfile", "-Command", disableScript).Run()
	fmt.Printf("  [MAC] disabled MAC randomization for adapter: %s\n", ifaceName)
	return nil
}

// GetNMConnection returns the NetworkManager connection name for an interface.
func GetNMConnection(ifaceName string) string {
	if _, err := exec.LookPath("nmcli"); err != nil {
		return ""
	}
	out, err := exec.Command("nmcli", "-g", "NAME,DEVICE", "con", "show", "--active").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) == ifaceName {
			return strings.TrimSpace(parts[0])
		}
	}
	return ""
}
