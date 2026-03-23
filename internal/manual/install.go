package manual

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed iitj-login.1
var page []byte

const pageName = "iitj-login.1"

// Install writes the man page to a directory visible to `man` when possible.
func Install() (string, error) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return "", nil
	}

	dir, err := installDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create man dir: %w", err)
	}

	path := filepath.Join(dir, pageName)
	if err := os.WriteFile(path, page, 0644); err != nil {
		return "", fmt.Errorf("write man page: %w", err)
	}
	return path, nil
}

// Remove deletes the installed man page if present.
func Remove() error {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil
	}

	path, err := installedPath()
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove man page: %w", err)
	}
	return nil
}

func installedPath() (string, error) {
	dirs, err := candidateDirs()
	if err != nil {
		return "", err
	}
	for _, dir := range dirs {
		path := filepath.Join(dir, pageName)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", nil
}

func installDir() (string, error) {
	dirs, err := candidateDirs()
	if err != nil {
		return "", err
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("could not find a writable man directory")
}

func candidateDirs() ([]string, error) {
	seen := map[string]struct{}{}
	var dirs []string

	add := func(base string) {
		if base == "" {
			return
		}
		dir := filepath.Join(base, "man1")
		if _, ok := seen[dir]; ok {
			return
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}

	for _, base := range manBasesFromEnv() {
		add(base)
	}

	for _, base := range manBasesFromCommand() {
		add(base)
	}

	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".local", "share", "man"))
	}

	return dirs, nil
}

func manBasesFromEnv() []string {
	manpath := os.Getenv("MANPATH")
	if manpath == "" {
		return nil
	}
	return splitPathList(manpath)
}

func manBasesFromCommand() []string {
	out, err := exec.Command("manpath").Output()
	if err != nil {
		return nil
	}
	return splitPathList(strings.TrimSpace(string(out)))
}

func splitPathList(s string) []string {
	parts := strings.Split(s, ":")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
