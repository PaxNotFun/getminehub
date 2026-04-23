//go:build windows

package installers

import (
	"os/exec"
	"syscall"
)

func hiddenWindowAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
func setHidden(cmd *exec.Cmd) { cmd.SysProcAttr = hiddenWindowAttr() }
