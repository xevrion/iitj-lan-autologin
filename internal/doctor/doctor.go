package doctor

import (
	"fmt"
	"strings"

	"github.com/iitj/iitj-lan-autologin/internal/creds"
	"github.com/iitj/iitj-lan-autologin/internal/detect"
	"github.com/iitj/iitj-lan-autologin/internal/fix"
	"github.com/iitj/iitj-lan-autologin/internal/login"
	"github.com/iitj/iitj-lan-autologin/internal/service"
)

type check struct {
	Name   string
	Status string
	Detail string
}

func Run() (string, error) {
	var checks []check

	platform := detect.DetectPlatform()
	checks = append(checks, check{
		Name:   "Platform",
		Status: "ok",
		Detail: renderPlatform(platform),
	})

	svc := service.New()
	info, err := svc.StatusInfo()
	if err != nil {
		checks = append(checks, check{Name: "Service", Status: "warn", Detail: err.Error()})
	} else {
		status := "ok"
		detail := fmt.Sprintf("%s (%s)", info.ServiceManager, info.ServiceName)
		switch {
		case !info.Installed:
			status = "warn"
			detail = "service is not installed"
		case !info.Running:
			status = "warn"
			detail = fmt.Sprintf("%s, currently stopped", detail)
		default:
			detail = fmt.Sprintf("%s, running", detail)
		}
		checks = append(checks, check{Name: "Service", Status: status, Detail: detail})
	}

	if !creds.HasCredentials() {
		checks = append(checks, check{Name: "Credentials", Status: "warn", Detail: "no stored credentials found"})
	} else if c, err := creds.LoadCredentials(); err != nil {
		checks = append(checks, check{Name: "Credentials", Status: "warn", Detail: fmt.Sprintf("stored credentials could not be read: %v", err)})
	} else {
		checks = append(checks, check{Name: "Credentials", Status: "ok", Detail: fmt.Sprintf("stored for %s", c.Username)})
	}

	cfg, cfgErr := creds.LoadConfig()
	if cfgErr != nil {
		checks = append(checks, check{Name: "Configuration", Status: "warn", Detail: "no saved interface configuration found"})
	} else {
		checks = append(checks, check{Name: "Configuration", Status: "ok", Detail: fmt.Sprintf("interface %s, gateway %s", cfg.Interface, cfg.Gateway)})

		if cfg.Interface == "" {
			checks = append(checks, check{Name: "Interface", Status: "warn", Detail: "saved interface is empty"})
		} else if currentIP, err := detect.GetInterfaceIP(cfg.Interface); err != nil {
			checks = append(checks, check{Name: "Interface", Status: "warn", Detail: fmt.Sprintf("%s is not currently ready: %v", cfg.Interface, err)})
		} else {
			detail := fmt.Sprintf("%s has IPv4 %s", cfg.Interface, currentIP)
			status := "ok"
			if cfg.InterfaceIP != "" && cfg.InterfaceIP != currentIP {
				status = "warn"
				detail = fmt.Sprintf("%s has IPv4 %s, saved config still says %s", cfg.Interface, currentIP, cfg.InterfaceIP)
			}
			checks = append(checks, check{Name: "Interface", Status: status, Detail: detail})
		}

		if cfg.InterfaceIP != "" {
			resolved := login.ResolvePortalIP(cfg.InterfaceIP)
			status := "ok"
			detail := fmt.Sprintf("gateway.iitj.ac.in resolves to %s on the Ethernet path", resolved)
			if resolved == "" {
				status = "warn"
				detail = "could not resolve gateway.iitj.ac.in on the Ethernet path"
			}
			checks = append(checks, check{Name: "Portal DNS", Status: status, Detail: detail})
		}
	}

	if fix.HostsEntryPresent() {
		checks = append(checks, check{Name: "Hosts entry", Status: "ok", Detail: "gateway.iitj.ac.in is pinned in hosts"})
	} else {
		checks = append(checks, check{Name: "Hosts entry", Status: "warn", Detail: "gateway.iitj.ac.in is not pinned in hosts"})
	}

	if conflict, bridgeIP, _ := fix.CheckDockerConflict(); conflict {
		checks = append(checks, check{Name: "Docker conflict", Status: "warn", Detail: fmt.Sprintf("docker bridge uses conflicting range near %s", bridgeIP)})
	} else {
		checks = append(checks, check{Name: "Docker conflict", Status: "ok", Detail: "no conflicting Docker bridge detected"})
	}

	if conflict, iface := fix.IsConflicting(); conflict {
		checks = append(checks, check{Name: "Local route conflict", Status: "warn", Detail: fmt.Sprintf("local interface %s overlaps the portal network", iface)})
	} else {
		checks = append(checks, check{Name: "Local route conflict", Status: "ok", Detail: "no overlapping local interface detected"})
	}

	state, err := creds.LoadRuntimeState()
	if err != nil {
		checks = append(checks, check{Name: "Runtime metadata", Status: "warn", Detail: err.Error()})
	} else if state.LastCheckAt == "" {
		checks = append(checks, check{Name: "Runtime metadata", Status: "warn", Detail: "no login loop health data recorded yet"})
	} else {
		status := "ok"
		detail := fmt.Sprintf("last check %s", state.LastCheckAt)
		if state.ConsecutiveFailures > 0 {
			status = "warn"
			detail = fmt.Sprintf("%s, %d consecutive failures, last error: %s", detail, state.ConsecutiveFailures, fallback(state.LastError, "unknown"))
		} else if state.LastSuccessAt != "" {
			detail = fmt.Sprintf("%s, last success %s", detail, state.LastSuccessAt)
		}
		checks = append(checks, check{Name: "Runtime metadata", Status: status, Detail: detail})
	}

	if logs, err := svc.RecentLogs(20); err != nil {
		checks = append(checks, check{Name: "Recent logs", Status: "warn", Detail: err.Error()})
	} else if line := findRecentProblem(logs); line != "" {
		checks = append(checks, check{Name: "Recent logs", Status: "warn", Detail: line})
	} else if len(logs) == 0 {
		checks = append(checks, check{Name: "Recent logs", Status: "warn", Detail: "no recent logs available"})
	} else {
		checks = append(checks, check{Name: "Recent logs", Status: "ok", Detail: "no recent error lines found"})
	}

	return renderReport(checks), nil
}

func renderPlatform(p detect.Platform) string {
	parts := []string{p.OS}
	if p.Distro != "" {
		parts = append(parts, p.Distro)
	}
	parts = append(parts, p.Arch)
	return strings.Join(parts, " / ")
}

func renderReport(checks []check) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iitj-login doctor\n\n")
	for _, c := range checks {
		fmt.Fprintf(&b, "%-20s [%s] %s\n", c.Name, strings.ToUpper(c.Status), c.Detail)
	}
	return b.String()
}

func findRecentProblem(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "timeout") {
			return line
		}
	}
	return ""
}

func fallback(v, alt string) string {
	if strings.TrimSpace(v) == "" {
		return alt
	}
	return v
}
