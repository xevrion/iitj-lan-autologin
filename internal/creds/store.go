package creds

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// Credentials holds the IITJ LDAP login credentials.
type Credentials struct {
	Username string
	Password string
}

// Config holds non-secret runtime configuration saved during install.
type Config struct {
	Interface    string `json:"interface"`
	InterfaceIP  string `json:"interface_ip"`
	Gateway      string `json:"gateway"`
}

const (
	appName      = "iitj-login"
	credsFile    = "credentials.enc"
	keyFile      = "key.bin"
	configFile   = "config.json"
)

// DataDir returns the platform-specific data directory.
func DataDir() (string, error) {
	var base string
	switch runtime.GOOS {
	case "linux":
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			base = xdg
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share")
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "Library", "Application Support")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		base = appdata
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, appName), nil
}

// SaveCredentials encrypts and saves credentials to the data directory.
func SaveCredentials(creds Credentials) error {
	dir, err := DataDir()
	if err != nil {
		return fmt.Errorf("data dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Generate 32-byte AES-256 key.
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	kPath := filepath.Join(dir, keyFile)
	if err := os.WriteFile(kPath, key, 0600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}

	// Encode credentials as "username\npassword".
	plaintext := []byte(creds.Username + "\n" + creds.Password)

	ct, err := aesGCMEncrypt(key, plaintext)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	cPath := filepath.Join(dir, credsFile)
	if err := os.WriteFile(cPath, ct, 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}

	return nil
}

// LoadCredentials decrypts and returns stored credentials.
func LoadCredentials() (Credentials, error) {
	dir, err := DataDir()
	if err != nil {
		return Credentials{}, fmt.Errorf("data dir: %w", err)
	}

	key, err := os.ReadFile(filepath.Join(dir, keyFile))
	if err != nil {
		return Credentials{}, fmt.Errorf("read key: %w", err)
	}

	ct, err := os.ReadFile(filepath.Join(dir, credsFile))
	if err != nil {
		return Credentials{}, fmt.Errorf("read credentials: %w", err)
	}

	plaintext, err := aesGCMDecrypt(key, ct)
	if err != nil {
		return Credentials{}, fmt.Errorf("decrypt: %w", err)
	}

	// Split "username\npassword".
	parts := splitN(string(plaintext), "\n", 2)
	if len(parts) != 2 {
		return Credentials{}, fmt.Errorf("invalid credentials format")
	}

	return Credentials{Username: parts[0], Password: parts[1]}, nil
}

// SaveConfig writes the runtime config as JSON.
func SaveConfig(cfg Config) error {
	dir, err := DataDir()
	if err != nil {
		return fmt.Errorf("data dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, configFile), data, 0600)
}

// LoadConfig reads the runtime config.
func LoadConfig() (Config, error) {
	dir, err := DataDir()
	if err != nil {
		return Config{}, fmt.Errorf("data dir: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

// RemoveAll deletes all stored data for this application.
func RemoveAll() error {
	dir, err := DataDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// aesGCMEncrypt encrypts plaintext with AES-256-GCM.
// Returned format: nonce (12 bytes) || ciphertext.
func aesGCMEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// aesGCMDecrypt decrypts a nonce-prefixed ciphertext with AES-256-GCM.
func aesGCMDecrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ct := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

func splitN(s, sep string, n int) []string {
	result := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := indexOf(s, sep)
		if idx == -1 {
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	result = append(result, s)
	return result
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
