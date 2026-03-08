//go:build !windows

package internal

import (
	"os/exec"
	"syscall"
)

func initProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessTree(pid int) {
	syscall.Kill(-pid, syscall.SIGTERM)
}

func killProcessTreeForce(pid int) {
	syscall.Kill(-pid, syscall.SIGKILL)
}
