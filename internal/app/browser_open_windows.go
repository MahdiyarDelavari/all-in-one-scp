//go:build windows

package app

import (
	"fmt"
	"os/exec"
)

func OpenBrowser(url string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser on Windows: %w", err)
	}

	return nil
}
