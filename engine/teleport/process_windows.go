package teleport

import (
	"context"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func configureProxyProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func stopProxyProcess(cmd *exec.Cmd) {
	// Windows has no Unix process groups. Stop tsh and its descendants together.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	kill := exec.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = kill.Run()
	_ = cmd.Process.Kill()
}
