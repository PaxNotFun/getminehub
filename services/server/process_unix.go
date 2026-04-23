//go:build !windows && !linux

package server

import (
	"os/exec"
	"syscall"
)

// setProcAttr en macOS y otros Unix: nuevo grupo de procesos.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcess mata el grupo de procesos completo.
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cmd.Process.Kill()
	}
}

// initJobObject — no aplicable en macOS/otros Unix.
func initJobObject(sm *ServerManager) {}

// assignJobObject — no aplicable en macOS/otros Unix.
func assignJobObject(sm *ServerManager, pid int) {}
