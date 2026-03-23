package login

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	postURL       = "https://gateway.iitj.ac.in:1003/"
	triggerURL    = "http://neverssl.com"
	connectTimeout = 10 * time.Second
	requestTimeout = 15 * time.Second
)

// CheckResult is the outcome of a login attempt.
type CheckResult struct {
	NeedsLogin bool   // true if the captive portal was detected
	LoggedIn   bool   // true if login POST succeeded
	Message    string // human-readable status
}

// CheckAndLogin checks whether the captive portal is active and logs in if so.
func CheckAndLogin(ifaceName, ifaceIP string, username, password string) (CheckResult, error) {
	portalIP := ResolvePortalIP(ifaceIP)
	client := newPortalClient(ifaceIP, portalIP)

	// Step 1 — trigger check.
	token, err := triggerCheck(client)
	if err != nil {
		return CheckResult{}, fmt.Errorf("trigger check: %w", err)
	}
	if token == "" {
		return CheckResult{NeedsLogin: false, Message: "already authenticated"}, nil
	}

	// Step 2 — fetch fgtauth page to extract the actual magic value.
	magic := fetchMagic(client, token)

	// Step 3 — POST credentials.
	referer := fmt.Sprintf("https://gateway.iitj.ac.in:1003/login?%s", token)
	loggedIn, err := postCredentials(client, username, password, magic, referer)
	if err != nil {
		return CheckResult{NeedsLogin: true, LoggedIn: false, Message: err.Error()}, nil
	}

	msg := "login successful"
	if !loggedIn {
		msg = "login POST sent — no keepalive in response"
	}
	return CheckResult{NeedsLogin: true, LoggedIn: loggedIn, Message: msg}, nil
}

// triggerCheck sends an HTTP GET to neverssl.com via the ethernet interface.
// FortiGate intercepts and returns a JS redirect containing the fgtauth token.
// Returns ("", nil) if already authenticated.
func triggerCheck(client *http.Client) (string, error) {
	resp, err := client.Get(triggerURL)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", triggerURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", fmt.Errorf("read trigger response: %w", err)
	}

	return extractToken(string(body)), nil
}

// fetchMagic fetches the fgtauth page and extracts the hidden magic value.
// Falls back to using the token directly if the fetch fails.
func fetchMagic(client *http.Client, token string) string {
	fgtURL := fmt.Sprintf("https://gateway.iitj.ac.in:1003/fgtauth?%s", token)
	resp, err := client.Get(fgtURL)
	if err != nil {
		return token
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return token
	}

	if magic := extractMagic(string(body)); magic != "" {
		return magic
	}
	return token
}

// postCredentials POSTs the login form and checks the response for "keepalive?".
func postCredentials(client *http.Client, username, password, magic, referer string) (bool, error) {
	form := url.Values{
		"username": {username},
		"password": {password},
		"magic":    {magic},
		"4Tredir":  {referer},
	}

	req, err := http.NewRequest("POST", postURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", referer)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("POST: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return false, fmt.Errorf("read response: %w", err)
	}

	return strings.Contains(string(body), "keepalive?"), nil
}

// newPortalClient builds an HTTP client that:
//   - Binds TCP connections to the ethernet interface IP (forces traffic via ethernet)
//   - Overrides DNS for gateway.iitj.ac.in → portalIP (bypasses glibc/getaddrinfo race)
//   - Skips TLS verification (FortiGate uses a self-signed cert)
//   - Does NOT follow redirects (FortiGate uses JS redirects, not HTTP 302)
func newPortalClient(ifaceIP, portalIP string) *http.Client {
	var localTCPAddr *net.TCPAddr
	if ifaceIP != "" {
		if parsed := net.ParseIP(ifaceIP); parsed != nil {
			localTCPAddr = &net.TCPAddr{IP: parsed}
		}
	}

	baseDialer := &net.Dialer{
		LocalAddr: localTCPAddr,
		Timeout:   connectTimeout,
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Force gateway.iitj.ac.in to the portal IP we resolved via ethernet DNS.
		host, port, err := net.SplitHostPort(addr)
		if err == nil && host == portalHostname && portalIP != "" {
			addr = net.JoinHostPort(portalIP, port)
		}
		return baseDialer.DialContext(ctx, network, addr)
	}

	transport := &http.Transport{
		DialContext:         dialContext,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, //nolint:gosec — FortiGate self-signed cert
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: connectTimeout,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		// Do not follow redirects — FortiGate uses JS redirects, not HTTP 302.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// extractToken parses the fgtauth token from a FortiGate JS redirect body.
// FortiGate returns: window.location="https://gateway.iitj.ac.in:1003/fgtauth?TOKEN"
func extractToken(body string) string {
	const marker = "fgtauth?"
	idx := strings.Index(body, marker)
	if idx == -1 {
		return ""
	}
	rest := body[idx+len(marker):]
	// Token ends at the first quote, double-quote, or whitespace.
	end := strings.IndexAny(rest, "\"' \t\n\r")
	if end == -1 {
		return strings.TrimSpace(rest)
	}
	return rest[:end]
}

// extractMagic parses the hidden magic input value from the fgtauth HTML page.
// FortiGate HTML contains: <input name="magic" value="ACTUAL_MAGIC_VALUE">
func extractMagic(html string) string {
	const nameAttr = `name="magic"`
	idx := strings.Index(html, nameAttr)
	if idx == -1 {
		return ""
	}
	rest := html[idx+len(nameAttr):]

	const valAttr = `value="`
	vidx := strings.Index(rest, valAttr)
	if vidx == -1 {
		return ""
	}
	rest = rest[vidx+len(valAttr):]

	end := strings.Index(rest, `"`)
	if end == -1 {
		return ""
	}
	return rest[:end]
}
