//go:build linux

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

	launchers := []struct {
		name string
		args []string
	}{
		{name: "x-terminal-emulator", args: []string{"-e", "sh", "-lc", commandLine}},
		{name: "gnome-terminal", args: []string{"--", "sh", "-lc", commandLine}},
		{name: "konsole", args: []string{"-e", "sh", "-lc", commandLine}},
		{name: "xfce4-terminal", args: []string{"-e", commandLine}},
		{name: "xterm", args: []string{"-e", commandLine}},
	}

	for _, launcher := range launchers {
		path, err := exec.LookPath(launcher.name)
		if err != nil {
			continue
		}

		cmd := exec.Command(path, launcher.args...)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("could not find a supported Linux terminal emulator")
}
