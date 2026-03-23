package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/iitj/iitj-lan-autologin/internal/creds"
	"github.com/iitj/iitj-lan-autologin/internal/installer"
	"github.com/iitj/iitj-lan-autologin/internal/login"
	"github.com/iitj/iitj-lan-autologin/internal/manual"
	"github.com/iitj/iitj-lan-autologin/internal/service"
)

const version = "4.0.11"

const usage = `iitj-login — IITJ Ethernet Auto Login

Usage:
  iitj-login <command>

Commands:
  install    Setup wizard: detect interface, apply fixes, store credentials, install daemon
  uninstall  Remove daemon and stored credentials
  login      Run the login loop (used by the daemon service)
  start      Start the background daemon
  stop       Stop the background daemon
  status     Show daemon status
  version    Show version
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "install":
		if err := installer.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "install failed: %v\n", err)
			os.Exit(1)
		}

	case "uninstall":
		svc := service.New()
		if err := svc.Uninstall(); err != nil {
			printError("uninstall failed", err)
			os.Exit(1)
		}
		if err := creds.RemoveAll(); err != nil {
			printWarn("uninstall warning", fmt.Sprintf("remove data: %v", err))
		}
		if err := manual.Remove(); err != nil {
			printWarn("uninstall warning", fmt.Sprintf("remove man page: %v", err))
		}
		printSuccess("Uninstalled.")

	case "login":
		service.PrepareBackgroundProcess("login")
		if err := login.RunLoop(); err != nil {
			fmt.Fprintf(os.Stderr, "login loop error: %v\n", err)
			os.Exit(1)
		}

	case "start":
		svc := service.New()
		if err := svc.Start(); err != nil {
			printError("start failed", err)
			os.Exit(1)
		}
		printSuccess("Started.")

	case "stop":
		svc := service.New()
		if err := svc.Stop(); err != nil {
			printError("stop failed", err)
			os.Exit(1)
		}
		printSuccess("Stopped.")

	case "status":
		svc := service.New()
		info, err := svc.StatusInfo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
			os.Exit(1)
		}
		recentLogs, _ := svc.RecentLogs(8)
		out, err := service.StatusReport(version, info, recentLogs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(out)

	case "version":
		fmt.Printf("%s %s\n", color("1;36", "iitj-login"), color("2", "v"+version))

	default:
		printError("unknown command", os.Args[1])
		fmt.Fprintf(os.Stderr, "\n%s", usage)
		os.Exit(1)
	}
}

func printSuccess(msg string) {
	fmt.Println(color("32", msg))
}

func printWarn(label, msg string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", color("33", label), msg)
}

func printError(label string, err interface{}) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", color("31", label), err)
}

func color(code, s string) string {
	if !useColor() {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func useColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb" && !strings.EqualFold(term, "unknown")
}
