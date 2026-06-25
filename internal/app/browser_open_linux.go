//go:build linux

package app

import (
	"fmt"
	"os/exec"
)

func OpenBrowser(url string) error {
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser on Linux: %w", err)
	}

	return nil
}
