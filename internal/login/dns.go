package login

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	portalHostname = "gateway.iitj.ac.in"
	// FortiGate's captive portal IP, returned by its intercepted DNS.
	portalFallbackIP = "172.17.0.3"
)

// FlushDNSCache flushes the system DNS cache using the appropriate tool.
// Errors are non-fatal — a failed flush is logged but not returned.
func FlushDNSCache() {
	switch runtime.GOOS {
	case "linux":
		flushLinux()
	case "darwin":
		flushDarwin()
	case "windows":
		exec.Command("ipconfig", "/flushdns").Run()
	}
}

func flushLinux() {
	// Try systemd-resolved first (most common on modern distros).
	if run("resolvectl", "flush-caches") == nil {
		return
	}
	// nscd (older Debian/Ubuntu systems).
	if run("nscd", "-i", "hosts") == nil {
		return
	}
	// dnsmasq — send SIGHUP to reload.
	run("sh", "-c", `pidof dnsmasq | xargs -r kill -HUP`)
}

func flushDarwin() {
	run("dscacheutil", "-flushcache")
	run("killall", "-HUP", "mDNSResponder")
}

// ResolvePortalIP resolves gateway.iitj.ac.in by sending a DNS query that
// routes through the ethernet interface. FortiGate intercepts DNS at the
// packet level on ethernet and returns its portal IP (172.17.0.3).
//
// If the custom resolution fails, falls back to the system resolver
// (which works if /etc/hosts contains the entry added during install)
// and finally to the hardcoded fallback IP.
func ResolvePortalIP(ifaceIP string) string {
	// 1. Custom resolver bound to ethernet interface IP.
	if ifaceIP != "" {
		if ip := resolveViaInterface(portalHostname, ifaceIP); ip != "" {
			return ip
		}
	}

	// 2. System resolver — uses /etc/hosts if entry was added during install.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, portalHostname)
	if err == nil {
		for _, a := range addrs {
			if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
				return a
			}
		}
	}

	// 3. Hardcoded fallback.
	return portalFallbackIP
}

// resolveViaInterface sends a DNS query bound to ifaceIP so the kernel
// routes it through the ethernet interface, letting FortiGate intercept it.
func resolveViaInterface(hostname, ifaceIP string) string {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				LocalAddr: &net.UDPAddr{IP: net.ParseIP(ifaceIP), Port: 0},
				Timeout:   5 * time.Second,
			}
			return d.DialContext(ctx, "udp", address)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrs, err := r.LookupHost(ctx, hostname)
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
			return a
		}
	}
	return ""
}

// run executes a command, returning an error if it fails.
func run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}
