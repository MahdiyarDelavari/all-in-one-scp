//go:build darwin

package app

import (
	"fmt"
	"os/exec"
)

func OpenBrowser(url string) error {
	cmd := exec.Command("open", url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser on macOS: %w", err)
	}

	return nil
}
