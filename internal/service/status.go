package service

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/iitj/iitj-lan-autologin/internal/creds"
	"github.com/iitj/iitj-lan-autologin/internal/detect"
	"github.com/iitj/iitj-lan-autologin/internal/manual"
)

// StatusInfo is a user-facing service summary independent of the platform backend.
type StatusInfo struct {
	ServiceManager string
	ServiceName    string
	Installed      bool
	Running        bool
	Startup        string
	PID            string
	LastExit       string
	LogHint        string
	Note           string
}

// StatusReport formats the user-facing status output.
func StatusReport(version string, info StatusInfo) (string, error) {
	platform := detect.DetectPlatform()
	dataDir, _ := creds.DataDir()
	cfg, cfgErr := creds.LoadConfig()
	hasCreds := creds.HasCredentials()
	manPagePath, _ := manual.InstalledPath()

	var b strings.Builder
	fmt.Fprintf(&b, "iitj-login v%s\n\n", version)

	writeField(&b, "Platform", renderPlatform(platform))
	writeField(&b, "Service", info.ServiceManager)
	writeField(&b, "Service name", info.ServiceName)
	writeField(&b, "Installed", yesNo(info.Installed))
	writeField(&b, "Running", yesNo(info.Running))
	writeField(&b, "Startup", fallback(info.Startup, "unknown"))

	if info.PID != "" {
		writeField(&b, "PID", info.PID)
	}
	if info.LastExit != "" {
		writeField(&b, "Last exit", info.LastExit)
	}

	writeField(&b, "Data dir", fallback(dataDir, "unknown"))
	writeField(&b, "Credentials", yesNo(hasCreds))

	if cfgErr == nil {
		writeField(&b, "Interface", cfg.Interface)
		writeField(&b, "Interface IP", cfg.InterfaceIP)
		writeField(&b, "Gateway", cfg.Gateway)
	} else {
		writeField(&b, "Interface", "not configured")
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		writeField(&b, "Man page", fallback(manPagePath, "not installed"))
	}

	if info.LogHint != "" {
		writeField(&b, "Logs", info.LogHint)
	}
	if info.Note != "" {
		writeField(&b, "Note", info.Note)
	}

	b.WriteString("\nCommands:\n")
	b.WriteString("  iitj-login start\n")
	b.WriteString("  iitj-login stop\n")
	b.WriteString("  iitj-login uninstall\n")

	return b.String(), nil
}

func writeField(b *strings.Builder, key, value string) {
	if value == "" {
		value = "unknown"
	}
	fmt.Fprintf(b, "%-13s %s\n", key+":", value)
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func fallback(v, alt string) string {
	if strings.TrimSpace(v) == "" {
		return alt
	}
	return v
}

func renderPlatform(p detect.Platform) string {
	parts := []string{p.OS}
	if p.Distro != "" {
		parts = append(parts, p.Distro)
	}
	parts = append(parts, p.Arch)
	return strings.Join(parts, " / ")
}
