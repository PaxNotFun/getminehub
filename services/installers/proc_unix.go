//go:build !windows

package installers

import (
	"os/exec"
	"syscall"
)

func hiddenWindowAttr() *syscall.SysProcAttr { return nil }
func setHidden(cmd *exec.Cmd)                { _ = cmd }
