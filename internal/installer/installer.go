package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/iitj/iitj-lan-autologin/internal/creds"
	"github.com/iitj/iitj-lan-autologin/internal/detect"
	"github.com/iitj/iitj-lan-autologin/internal/fix"
	"github.com/iitj/iitj-lan-autologin/internal/manual"
	"github.com/iitj/iitj-lan-autologin/internal/service"
)

// Run executes the interactive install wizard.
func Run() error {
	printBanner()

	platform := detect.DetectPlatform()
	fmt.Printf("Platform: %s", platform.OS)
	if platform.Distro != "" {
		fmt.Printf(" / %s", platform.Distro)
	}
	fmt.Printf(" (%s) — init: %s\n\n", platform.Arch, platform.InitSys)

	// Step 1 — detect or prompt for ethernet interface.
	iface, err := detectInterface()
	if err != nil {
		return err
	}
	fmt.Printf("Interface: %s (IP: %s, GW: %s)\n\n", iface.Name, iface.IP, iface.Gateway)

	// Step 2 — apply platform fixes.
	fmt.Println("Applying fixes...")

	// MAC randomization.
	nmConn := ""
	if platform.OS == "linux" {
		nmConn = fix.GetNMConnection(iface.Name)
	}
	if err := fix.DisableMACRandomization(iface.Name, nmConn); err != nil {
		fmt.Printf("  [MAC] warning: %v\n", err)
	}

	// Docker conflict check.
	fix.PrintDockerWarning()

	// /etc/hosts entry — must happen before routing fix so DNS works.
	if err := fix.AddHostsEntry(); err != nil {
		fmt.Printf("  [hosts] warning: %v\n", err)
		fmt.Println("  [hosts] you may need to add '172.17.0.3 gateway.iitj.ac.in' to your hosts file manually.")
	}

	// Static route for portal IP.
	if err := fix.FixRouting(iface.Name, iface.Gateway, nmConn); err != nil {
		fmt.Printf("  [route] warning: %v\n", err)
	}

	fmt.Println()

	// Step 3 — collect and store credentials.
	fmt.Println("Enter your IITJ LDAP credentials:")
	username, err := prompt("  Username: ", false)
	if err != nil {
		return fmt.Errorf("read username: %w", err)
	}
	password, err := prompt("  Password: ", true)
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	fmt.Println()

	if err := creds.SaveCredentials(creds.Credentials{
		Username: username,
		Password: password,
	}); err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}

	// Step 4 — save config.
	if err := creds.SaveConfig(creds.Config{
		Interface:   iface.Name,
		InterfaceIP: iface.IP,
		Gateway:     iface.Gateway,
	}); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Step 5 — install service.
	fmt.Println("Installing daemon service...")
	execPath, err := service.ExecPath()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	svc := service.New()
	if err := svc.Install(execPath); err != nil {
		return fmt.Errorf("install service: %w", err)
	}

	manPagePath := ""
	if path, err := manual.Install(); err != nil {
		fmt.Printf("  [man] warning: %v\n", err)
	} else {
		manPagePath = path
	}

	dataDir, _ := creds.DataDir()
	fmt.Printf("\nInstallation complete.\n")
	fmt.Printf("  Data dir : %s\n", dataDir)
	fmt.Printf("  Binary   : %s\n", execPath)
	if manPagePath != "" {
		fmt.Printf("  Man page : %s\n", manPagePath)
	}
	fmt.Printf("\nCommands: iitj-login status | start | stop | uninstall\n")

	return nil
}

func detectInterface() (detect.NetInterface, error) {
	iface, err := detect.DetectEthernetInterface()
	if err == nil {
		return iface, nil
	}

	fmt.Printf("Could not auto-detect ethernet interface (%v).\n", err)
	name, err := prompt("Enter interface name (e.g. eth0, enp7s0): ", false)
	if err != nil {
		return detect.NetInterface{}, err
	}

	ip, _ := detect.GetInterfaceIP(name)
	return detect.NetInterface{Name: name, IP: ip}, nil
}

func printBanner() {
	fmt.Println("==========================================")
	fmt.Println(" IITJ Ethernet Auto Login Installer v4.0")
	fmt.Println("==========================================")
	fmt.Println()
}

// prompt reads a line from stdin, optionally hiding input (for passwords).
func prompt(text string, secret bool) (string, error) {
	fmt.Print(text)

	if secret {
		return readPassword()
	}

	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	return strings.TrimSpace(line), err
}

// readPassword reads a password without echoing it to the terminal.
func readPassword() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return readPasswordWindows()
	default:
		return readPasswordUnix()
	}
}

func readPasswordUnix() (string, error) {
	// Disable echo via stty, which is available on all POSIX systems.
	sttyOff := exec.Command("stty", "-echo")
	sttyOff.Stdin = os.Stdin
	if err := sttyOff.Run(); err == nil {
		defer func() {
			sttyOn := exec.Command("stty", "echo")
			sttyOn.Stdin = os.Stdin
			sttyOn.Run()
			fmt.Println()
		}()
	}

	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	return strings.TrimSpace(line), err
}

func readPasswordWindows() (string, error) {
	// Use PowerShell Read-Host -AsSecureString to hide input.
	script := `$p = Read-Host -AsSecureString; ` +
		`[System.Runtime.InteropServices.Marshal]::PtrToStringAuto(` +
		`[System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($p))`

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err == nil {
		fmt.Println()
		return strings.TrimSpace(string(out)), nil
	}

	// Fallback to visible input.
	fmt.Print("[password will be visible] ")
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	fmt.Println()
	return strings.TrimSpace(line), err
}
