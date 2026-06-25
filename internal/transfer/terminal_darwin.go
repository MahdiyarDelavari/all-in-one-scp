//go:build darwin

package transfer

import (
	"fmt"
	"os/exec"
)

func OpenInteractiveShell(config ConnectionConfig) error {
	parts, err := BuildSSHCommandParts(config)
	if err != nil {
		return err
	}

	commandLine := joinForPOSIXShell(parts)
	script := fmt.Sprintf(`tell application "Terminal"
activate
do script %q
end tell`, commandLine)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open Terminal.app SSH session: %w", err)
	}

	return nil
}
