//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

func shellCommand(command string) *exec.Cmd {
	return exec.Command("/bin/sh", "-c", command)
}

// setProcessGroup puts the dev server in its own process group so stop reaches bun/node/go children.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateTree(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
}

func killTree(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
