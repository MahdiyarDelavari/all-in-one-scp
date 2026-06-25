package transfer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConnectionConfig stores one server login configuration.
type ConnectionConfig struct {
	Host        string
	Port        int
	User        string
	KeyPath     string
	Password    string
	PasswordEnv string
	Insecure    bool
}

func (c ConnectionConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c ConnectionConfig) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("host is required")
	}

	if strings.TrimSpace(c.User) == "" {
		return fmt.Errorf("username is required")
	}

	if c.Port <= 0 {
		return fmt.Errorf("port must be greater than zero")
	}

	if c.KeyPath == "" && c.Password == "" && c.PasswordEnv == "" {
		return fmt.Errorf("choose one authentication method")
	}

	return nil
}

func (c ConnectionConfig) ExpandedKeyPath() (string, error) {
	return expandPath(c.KeyPath)
}

func (c ConnectionConfig) PasswordFromEnv() string {
	if c.PasswordEnv == "" {
		return ""
	}

	return os.Getenv(c.PasswordEnv)
}

func expandPath(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	if value == "~" || strings.HasPrefix(value, "~/") || strings.HasPrefix(value, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home directory: %w", err)
		}

		return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(value, "~/"), "~\\")), nil
	}

	return value, nil
}
