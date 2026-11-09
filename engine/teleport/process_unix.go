//go:build !windows

package teleport

import (
	"os/exec"
	"syscall"
	"time"
)

func configureProxyProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func stopProxyProcess(cmd *exec.Cmd) {
	pgid, _ := syscall.Getpgid(cmd.Process.Pid)
	if pgid <= 0 {
		pgid = cmd.Process.Pid
	}
	// Try graceful shutdown before terminating the entire process group.
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	time.AfterFunc(500*time.Millisecond, func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		_ = cmd.Process.Kill()
	})
}
