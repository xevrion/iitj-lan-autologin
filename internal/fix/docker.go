package fix

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// CheckDockerConflict detects if Docker's bridge network occupies 172.17.0.0/16,
// which shadows the FortiGate portal IP (172.17.0.3).
// Returns (conflicting, dockerBridgeIP, fix instructions).
func CheckDockerConflict() (bool, string, string) {
	if _, err := exec.LookPath("docker"); err != nil {
		return false, "", "" // Docker not installed
	}

	iface, err := net.InterfaceByName("docker0")
	if err != nil {
		return false, "", "" // docker0 not present
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return false, "", ""
	}

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP.To4()
		if ip == nil {
			continue
		}
		// Check if docker0 is in the 172.16.0.0/12 range (which includes 172.17.x.x).
		if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
			bridgeIP := ip.String()
			fix := buildDockerFix()
			return true, bridgeIP, fix
		}
	}

	return false, "", ""
}

func buildDockerFix() string {
	return strings.TrimSpace(`
Docker bridge (docker0) is using the 172.17.x.x range, which conflicts with
FortiGate's captive portal IP (172.17.0.3). The kernel will route portal
traffic into Docker's local bridge instead of sending it to FortiGate.

Fix (requires sudo):

  sudo mkdir -p /etc/docker
  sudo tee /etc/docker/daemon.json <<'EOF'
  { "default-address-pools": [{ "base": "10.200.0.0/16", "size": 24 }] }
  EOF
  sudo systemctl restart docker && docker network prune -f

After this, re-run the installer so the routing fix uses the correct portal IP.
`)
}

// PrintDockerWarning prints the Docker conflict warning if applicable.
func PrintDockerWarning() {
	conflict, bridgeIP, fix := CheckDockerConflict()
	if !conflict {
		return
	}
	fmt.Printf("\nWARNING: Docker bridge (docker0) is at %s\n", bridgeIP)
	fmt.Println(fix)
}
