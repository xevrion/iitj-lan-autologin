package detect

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

// NetInterface holds the detected ethernet interface details.
type NetInterface struct {
	Name    string
	IP      string // primary IPv4 address
	Gateway string // default gateway via this interface
}

// DetectEthernetInterface auto-detects the active ethernet interface.
func DetectEthernetInterface() (NetInterface, error) {
	switch runtime.GOOS {
	case "linux":
		return detectLinux()
	case "darwin":
		return detectDarwin()
	case "windows":
		return detectWindows()
	default:
		return NetInterface{}, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// GetInterfaceIP returns the primary IPv4 address for a named interface.
func GetInterfaceIP(name string) (string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", fmt.Errorf("interface %s not found: %w", name, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
			return ip4.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 address on interface %s", name)
}

// ethernetPrefixes are known ethernet interface name prefixes on Linux.
var ethernetPrefixes = []string{"eth", "enp", "ens", "eno", "em"}

// excludePrefixes are virtual/wireless interfaces to skip on Linux.
var excludePrefixes = []string{"lo", "docker", "br-", "virbr", "wl", "ww", "veth", "tun", "tap"}

func detectLinux() (NetInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return NetInterface{}, fmt.Errorf("listing interfaces: %w", err)
	}

	for _, iface := range ifaces {
		name := iface.Name

		if !isEthernetLinux(name) {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		ip, err := GetInterfaceIP(name)
		if err != nil {
			continue // no IP yet, skip
		}

		gw := getGatewayLinux(name)
		return NetInterface{Name: name, IP: ip, Gateway: gw}, nil
	}

	return NetInterface{}, fmt.Errorf("no active ethernet interface found")
}

func isEthernetLinux(name string) bool {
	for _, ex := range excludePrefixes {
		if strings.HasPrefix(name, ex) {
			return false
		}
	}
	for _, prefix := range ethernetPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func getGatewayLinux(ifaceName string) string {
	out, err := exec.Command("ip", "route", "show", "dev", ifaceName).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "default" && fields[1] == "via" {
			return fields[2]
		}
	}
	// fallback: global default route
	out2, err := exec.Command("ip", "route").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out2), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "default" && fields[1] == "via" {
			return fields[2]
		}
	}
	return ""
}

func detectDarwin() (NetInterface, error) {
	// networksetup -listallhardwareports gives us:
	// Hardware Port: Ethernet
	// Device: en0
	out, err := exec.Command("networksetup", "-listallhardwareports").Output()
	if err != nil {
		return NetInterface{}, fmt.Errorf("networksetup failed: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Hardware Port:") {
			continue
		}
		portName := strings.TrimSpace(strings.TrimPrefix(line, "Hardware Port:"))
		// Only match wired ethernet ports, not Wi-Fi or Thunderbolt Bridge
		if !strings.Contains(strings.ToLower(portName), "ethernet") {
			continue
		}
		// Next line should be "Device: enX"
		if i+1 >= len(lines) {
			continue
		}
		devLine := strings.TrimSpace(lines[i+1])
		if !strings.HasPrefix(devLine, "Device:") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(devLine, "Device:"))

		ip, err := GetInterfaceIP(name)
		if err != nil {
			continue
		}
		gw := getGatewayDarwin(name)
		return NetInterface{Name: name, IP: ip, Gateway: gw}, nil
	}

	return NetInterface{}, fmt.Errorf("no active ethernet interface found")
}

func getGatewayDarwin(ifaceName string) string {
	out, err := exec.Command("netstat", "-rn", "-f", "inet").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		// Look for: default  <gateway>  ...  <ifaceName>
		if len(fields) >= 6 && fields[0] == "default" && fields[len(fields)-1] == ifaceName {
			return fields[1]
		}
	}
	return ""
}

func detectWindows() (NetInterface, error) {
	// Use PowerShell to find connected ethernet adapters
	script := `Get-NetAdapter | Where-Object { $_.Status -eq 'Up' -and $_.PhysicalMediaType -eq '802.3' } | Select-Object -First 1 -ExpandProperty Name`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return NetInterface{}, fmt.Errorf("powershell failed: %w", err)
	}

	name := strings.TrimSpace(string(out))
	if name == "" {
		return NetInterface{}, fmt.Errorf("no active ethernet adapter found")
	}

	ip, err := GetInterfaceIPWindows(name)
	if err != nil {
		return NetInterface{}, err
	}

	gw := getGatewayWindows(name)
	return NetInterface{Name: name, IP: ip, Gateway: gw}, nil
}

// GetInterfaceIPWindows resolves the IPv4 for a Windows adapter name via PowerShell.
func GetInterfaceIPWindows(adapterName string) (string, error) {
	script := fmt.Sprintf(
		`(Get-NetIPAddress -InterfaceAlias '%s' -AddressFamily IPv4 -ErrorAction SilentlyContinue | Select-Object -First 1).IPAddress`,
		adapterName,
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return "", fmt.Errorf("could not get IP for %s: %w", adapterName, err)
	}
	ip := strings.TrimSpace(string(out))
	if ip == "" {
		return "", fmt.Errorf("no IPv4 address on adapter %s", adapterName)
	}
	return ip, nil
}

func getGatewayWindows(adapterName string) string {
	script := fmt.Sprintf(
		`(Get-NetRoute -InterfaceAlias '%s' -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue | Select-Object -First 1).NextHop`,
		adapterName,
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
