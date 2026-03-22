package detect

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

// Platform holds detected OS, distro, and init system info.
type Platform struct {
	OS      string // "linux", "darwin", "windows"
	Distro  string // "fedora", "ubuntu", "arch", "debian", etc. (linux only)
	IDLike  string // ID_LIKE field from os-release (e.g. "rhel fedora")
	InitSys string // "systemd", "launchd", "openrc", "runit", "windows", "unknown"
	Arch    string // "amd64", "arm64", etc.
}

// DetectPlatform returns the current platform information.
func DetectPlatform() Platform {
	p := Platform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	switch p.OS {
	case "linux":
		p.Distro, p.IDLike = parseOSRelease()
		p.InitSys = detectInitSystem()
	case "darwin":
		p.InitSys = "launchd"
	case "windows":
		p.InitSys = "windows"
	default:
		p.InitSys = "unknown"
	}

	return p
}

// IsSystemd returns true if the platform uses systemd.
func (p Platform) IsSystemd() bool {
	return p.InitSys == "systemd"
}

// IsFedoraLike returns true for Fedora, RHEL, CentOS, etc.
func (p Platform) IsFedoraLike() bool {
	return strings.Contains(p.Distro, "fedora") ||
		strings.Contains(p.IDLike, "fedora") ||
		strings.Contains(p.IDLike, "rhel")
}

// HasNMCLI returns true if NetworkManager is likely present.
func (p Platform) HasNMCLI() bool {
	return p.OS == "linux" && (p.IsFedoraLike() ||
		strings.Contains(p.Distro, "ubuntu") ||
		strings.Contains(p.IDLike, "debian"))
}

func parseOSRelease() (id, idLike string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id = strings.ToLower(strings.Trim(strings.TrimPrefix(line, "ID="), `"`))
		} else if strings.HasPrefix(line, "ID_LIKE=") {
			idLike = strings.ToLower(strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), `"`))
		}
	}
	return id, idLike
}

func detectInitSystem() string {
	// Check /proc/1/comm — PID 1 name indicates init system.
	data, err := os.ReadFile("/proc/1/comm")
	if err != nil {
		return "unknown"
	}
	comm := strings.TrimSpace(string(data))
	switch comm {
	case "systemd":
		return "systemd"
	case "openrc-init", "openrc":
		return "openrc"
	case "runit":
		return "runit"
	case "s6-svscan":
		return "s6"
	default:
		return "unknown"
	}
}
