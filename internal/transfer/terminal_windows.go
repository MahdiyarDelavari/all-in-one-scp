//go:build windows

package transfer

import (
	"fmt"
	"os/exec"
	"strings"
)

func OpenInteractiveShell(config ConnectionConfig) error {
	parts, err := BuildSSHCommandParts(config)
	if err != nil {
		return err
	}

	// Windows does not have /dev/null, so adjust the insecure known-hosts path here.
	for index := 0; index < len(parts)-1; index++ {
		if parts[index] == "UserKnownHostsFile=/dev/null" {
			parts[index] = "UserKnownHostsFile=NUL"
		}
	}

	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, quoteForPowerShell(part))
	}

	commandLine := strings.Join(quoted, " ")
	cmd := exec.Command("powershell", "-NoExit", "-Command", commandLine)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start PowerShell SSH window: %w", err)
	}

	return nil
}
