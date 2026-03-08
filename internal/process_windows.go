//go:build windows

package internal

import (
	"os/exec"
	"strconv"
)

func initProcess(cmd *exec.Cmd) {
	// Windows: no process group setup needed; taskkill /T handles the tree
}

func killProcessTree(pid int) {
	exec.Command("taskkill", "/T", "/PID", strconv.Itoa(pid)).Run()
}

func killProcessTreeForce(pid int) {
	exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
}
