package transfer

import (
	"fmt"
	"strconv"
	"strings"
)

// BuildSSHCommandParts returns a portable ssh command split into arguments.
// The OS-specific launcher can then decide how to feed these args to a terminal app.
func BuildSSHCommandParts(config ConnectionConfig) ([]string, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	parts := []string{"ssh"}

	if config.Insecure {
		parts = append(parts, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null")
	}

	if config.KeyPath != "" {
		keyPath, err := config.ExpandedKeyPath()
		if err != nil {
			return nil, err
		}

		parts = append(parts, "-i", keyPath)
	}

	parts = append(parts, "-p", strconv.Itoa(config.Port))
	parts = append(parts, fmt.Sprintf("%s@%s", config.User, config.Host))

	return parts, nil
}

func joinForPOSIXShell(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, quoteForPOSIXShell(part))
	}

	return strings.Join(quoted, " ")
}

func quoteForPOSIXShell(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func quoteForPowerShell(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
