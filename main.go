package main

import (
	"fmt"
	"os"

	"github.com/iitj/iitj-lan-autologin/internal/creds"
	"github.com/iitj/iitj-lan-autologin/internal/installer"
	"github.com/iitj/iitj-lan-autologin/internal/login"
	"github.com/iitj/iitj-lan-autologin/internal/manual"
	"github.com/iitj/iitj-lan-autologin/internal/service"
)

const version = "4.0.1"

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
			fmt.Fprintf(os.Stderr, "uninstall failed: %v\n", err)
			os.Exit(1)
		}
		if err := creds.RemoveAll(); err != nil {
			fmt.Fprintf(os.Stderr, "uninstall warning: remove data: %v\n", err)
		}
		if err := manual.Remove(); err != nil {
			fmt.Fprintf(os.Stderr, "uninstall warning: remove man page: %v\n", err)
		}
		fmt.Println("Uninstalled.")

	case "login":
		if err := login.RunLoop(); err != nil {
			fmt.Fprintf(os.Stderr, "login loop error: %v\n", err)
			os.Exit(1)
		}

	case "start":
		svc := service.New()
		if err := svc.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "start failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Started.")

	case "stop":
		svc := service.New()
		if err := svc.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "stop failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Stopped.")

	case "status":
		svc := service.New()
		out, err := svc.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(out)

	case "version":
		fmt.Printf("iitj-login v%s\n", version)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
}
