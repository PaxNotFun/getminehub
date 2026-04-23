//go:build linux

package server

import (
	"os/exec"
	"syscall"
)

// setProcAttr en Linux: nuevo grupo de procesos + SIGKILL automático si el padre muere.
// Pdeathsig garantiza que el servidor Java se cierre si la app se cierra inesperadamente.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGKILL,
	}
}

// killProcess mata el grupo de procesos completo (incluye hijos Java).
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cmd.Process.Kill()
	}
}

// initJobObject — no necesario en Linux (Pdeathsig cubre esto).
func initJobObject(sm *ServerManager) {}

// assignJobObject — no necesario en Linux.
func assignJobObject(sm *ServerManager, pid int) {}
