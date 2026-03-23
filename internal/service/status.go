package service

import (
	"fmt"
	"os"
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
func StatusReport(version string, info StatusInfo, recentLogs []string) (string, error) {
	platform := detect.DetectPlatform()
	dataDir, _ := creds.DataDir()
	cfg, cfgErr := creds.LoadConfig()
	hasCreds := creds.HasCredentials()
	manPagePath, _ := manual.InstalledPath()
	ui := newPalette(useColor())

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n\n", ui.title("iitj-login"), ui.muted("v"+version))

	b.WriteString(ui.section("Overview"))
	writeField(&b, ui, "Platform", renderPlatform(platform))
	writeField(&b, ui, "Service", info.ServiceManager)
	writeField(&b, ui, "Service name", info.ServiceName)
	writeField(&b, ui, "Installed", installedValue(ui, info.Installed))
	writeField(&b, ui, "Running", runningValue(ui, info.Running))
	writeField(&b, ui, "Startup", fallback(info.Startup, "unknown"))

	if info.PID != "" {
		writeField(&b, ui, "PID", info.PID)
	}
	if info.LastExit != "" {
		writeField(&b, ui, "Last exit", ui.warn(info.LastExit))
	}

	b.WriteString("\n")
	b.WriteString(ui.section("Configuration"))
	writeField(&b, ui, "Data dir", fallback(dataDir, "unknown"))
	writeField(&b, ui, "Credentials", credentialsValue(ui, hasCreds))

	if cfgErr == nil {
		writeField(&b, ui, "Interface", cfg.Interface)
		writeField(&b, ui, "Interface IP", cfg.InterfaceIP)
		writeField(&b, ui, "Gateway", cfg.Gateway)
	} else {
		writeField(&b, ui, "Interface", ui.warn("not configured"))
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		writeField(&b, ui, "Man page", fallback(manPagePath, "not installed"))
	}

	b.WriteString("\n")
	b.WriteString(ui.section("Diagnostics"))
	if info.LogHint != "" {
		writeField(&b, ui, "Log source", info.LogHint)
	}
	if info.Note != "" {
		writeField(&b, ui, "Note", ui.warn(info.Note))
	}

	b.WriteString("\n")
	b.WriteString(ui.section("Recent logs"))
	if len(recentLogs) == 0 {
		b.WriteString("  (no recent logs available)\n")
	} else {
		for _, line := range recentLogs {
			b.WriteString("  ")
			b.WriteString(ui.logLine(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(ui.section("Commands"))
	b.WriteString("  iitj-login start\n")
	b.WriteString("  iitj-login stop\n")
	b.WriteString("  iitj-login uninstall\n")

	return b.String(), nil
}

func writeField(b *strings.Builder, ui palette, key, value string) {
	if value == "" {
		value = "unknown"
	}
	fmt.Fprintf(b, "  %-13s %s\n", ui.label(key+":"), value)
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

func installedValue(ui palette, v bool) string {
	if v {
		return ui.ok("configured")
	}
	return ui.warn("not installed")
}

func runningValue(ui palette, v bool) string {
	if v {
		return ui.ok("running")
	}
	return ui.bad("stopped")
}

func credentialsValue(ui palette, v bool) string {
	if v {
		return ui.ok("stored")
	}
	return ui.warn("missing")
}

type palette struct {
	enabled bool
}

func newPalette(enabled bool) palette {
	return palette{enabled: enabled}
}

func (p palette) wrap(code, s string) string {
	if !p.enabled {
		return s
	}
	return code + s + "\033[0m"
}

func (p palette) title(s string) string   { return p.wrap("\033[1;36m", s) }
func (p palette) section(s string) string { return p.wrap("\033[1;34m", s) + "\n" }
func (p palette) label(s string) string   { return p.wrap("\033[1m", s) }
func (p palette) ok(s string) string      { return p.wrap("\033[32m", s) }
func (p palette) warn(s string) string    { return p.wrap("\033[33m", s) }
func (p palette) bad(s string) string     { return p.wrap("\033[31m", s) }
func (p palette) muted(s string) string   { return p.wrap("\033[2m", s) }
func (p palette) logLine(s string) string {
	switch {
	case strings.Contains(strings.ToLower(s), "error"), strings.Contains(strings.ToLower(s), "failed"):
		return p.bad(s)
	case strings.Contains(strings.ToLower(s), "warning"):
		return p.warn(s)
	default:
		return s
	}
}

func useColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
