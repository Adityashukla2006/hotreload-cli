package internal

import "runtime"

func shellArgs(cmdStr string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", cmdStr}
	}
	return "sh", []string{"-c", cmdStr}
}
