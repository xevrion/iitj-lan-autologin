package fix

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

const portalIP = "172.17.0.3"

// FixRouting ensures that the FortiGate portal IP (172.17.0.3) routes via the
// ethernet interface and not through WiFi or a virtual bridge.
//
// On Linux this uses nmcli to add a persistent static route.
// On macOS/Windows it adds a temporary route (survives until reboot).
func FixRouting(ifaceName, gateway, nmConn string) error {
	if gateway == "" {
		fmt.Println("  [route] gateway not detected — skipping routing fix")
		return nil
	}

	// Check if the portal IP already routes correctly.
	current := currentRouteDev()
	if current == ifaceName {
		fmt.Printf("  [route] %s already routes via %s — OK\n", portalIP, ifaceName)
		return nil
	}

	fmt.Printf("  [route] %s routes via %q — pinning to %s\n", portalIP, current, ifaceName)

	switch runtime.GOOS {
	case "linux":
		return fixRoutingLinux(ifaceName, gateway, nmConn)
	case "darwin":
		return fixRoutingDarwin(ifaceName, gateway)
	case "windows":
		return fixRoutingWindows(ifaceName, gateway)
	}
	return nil
}

func currentRouteDev() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	out, err := exec.Command("ip", "route", "get", portalIP).Output()
	if err != nil {
		return ""
	}
	for _, field := range strings.Fields(string(out)) {
		// "ip route get" output: ... dev <ifaceName> ...
		// We look for the word after "dev".
		if field == "dev" {
			continue
		}
	}
	// Parse: "172.17.0.3 dev eth0 src ..."
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "dev" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}

func fixRoutingLinux(ifaceName, gateway, nmConn string) error {
	if nmConn != "" {
		if _, err := exec.LookPath("nmcli"); err == nil {
			// Add persistent route via NetworkManager.
			route := fmt.Sprintf("%s/32 %s", portalIP, gateway)
			out, err := exec.Command("nmcli", "connection", "modify", nmConn,
				"+ipv4.routes", route).CombinedOutput()
			if err != nil {
				return fmt.Errorf("nmcli add route: %w: %s", err, strings.TrimSpace(string(out)))
			}
			// Bounce the connection to apply.
			exec.Command("nmcli", "connection", "down", nmConn).Run()
			exec.Command("nmcli", "connection", "up", nmConn).Run()
			fmt.Printf("  [route] added persistent route: %s/32 via %s (%s)\n", portalIP, gateway, nmConn)
			return nil
		}
	}

	// Fallback: add temporary route via ip command.
	out, err := exec.Command("ip", "route", "add",
		portalIP+"/32", "via", gateway, "dev", ifaceName).CombinedOutput()
	if err != nil && !strings.Contains(string(out), "File exists") {
		return fmt.Errorf("ip route add: %w: %s", err, strings.TrimSpace(string(out)))
	}
	fmt.Printf("  [route] added temporary route: %s/32 via %s\n", portalIP, gateway)
	return nil
}

func fixRoutingDarwin(ifaceName, gateway string) error {
	out, err := exec.Command("route", "add", "-host", portalIP, gateway).CombinedOutput()
	if err != nil && !strings.Contains(string(out), "File exists") {
		return fmt.Errorf("route add: %w: %s", err, strings.TrimSpace(string(out)))
	}
	fmt.Printf("  [route] added route: %s via %s (%s)\n", portalIP, gateway, ifaceName)
	return nil
}

func fixRoutingWindows(ifaceName, gateway string) error {
	// Get the interface index for the ethernet adapter.
	script := fmt.Sprintf(
		`(Get-NetAdapter -Name '%s').ifIndex`,
		ifaceName,
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return fmt.Errorf("get interface index: %w", err)
	}
	ifIndex := strings.TrimSpace(string(out))

	// Delete any existing route first.
	exec.Command("route", "delete", portalIP).Run()

	addOut, addErr := exec.Command("route", "add", portalIP, "mask", "255.255.255.255",
		gateway, "if", ifIndex).CombinedOutput()
	if addErr != nil {
		return fmt.Errorf("route add: %w: %s", addErr, strings.TrimSpace(string(addOut)))
	}
	fmt.Printf("  [route] added route: %s via %s\n", portalIP, gateway)
	return nil
}

// IsConflicting returns true if any local virtual interface owns 172.17.0.0/16,
// which would shadow the FortiGate portal IP.
func IsConflicting() (bool, string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false, ""
	}
	portalNet := &net.IPNet{
		IP:   net.ParseIP("172.17.0.0").To4(),
		Mask: net.CIDRMask(16, 32),
	}
	portalAddr := net.ParseIP(portalIP)

	for _, iface := range ifaces {
		// Skip ethernet-like interfaces.
		if isEthernetLike(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if portalNet.Contains(ipnet.IP) || ipnet.Contains(portalAddr) {
				return true, iface.Name
			}
		}
	}
	return false, ""
}

func isEthernetLike(name string) bool {
	for _, p := range []string{"eth", "enp", "ens", "eno", "em", "en"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
