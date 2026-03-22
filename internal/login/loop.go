package login

import (
	"fmt"
	"time"

	"github.com/iitj/iitj-lan-autologin/internal/creds"
	"github.com/iitj/iitj-lan-autologin/internal/detect"
)

const checkInterval = 300 * time.Second

// RunLoop is the daemon entry point. It loads config + credentials, then
// loops: check portal → login if needed → sleep 300s → repeat.
func RunLoop() error {
	cfg, err := creds.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	c, err := creds.LoadCredentials()
	if err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	ifaceName := cfg.Interface
	ifaceIP := cfg.InterfaceIP

	// Refresh interface IP at start in case DHCP changed it.
	if ip, err := detect.GetInterfaceIP(ifaceName); err == nil {
		ifaceIP = ip
	}

	fmt.Printf("[iitj-login] started — interface: %s (%s)\n", ifaceName, ifaceIP)

	for {
		FlushDNSCache()

		// Re-check interface IP each loop in case it changed.
		if ip, err := detect.GetInterfaceIP(ifaceName); err == nil {
			ifaceIP = ip
		}

		result, err := CheckAndLogin(ifaceName, ifaceIP, c.Username, c.Password)
		if err != nil {
			fmt.Printf("[%s] error: %v\n", timestamp(), err)
		} else if result.NeedsLogin {
			fmt.Printf("[%s] captive portal detected — %s\n", timestamp(), result.Message)
		} else {
			fmt.Printf("[%s] already authenticated\n", timestamp())
		}

		time.Sleep(checkInterval)
	}
}

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
