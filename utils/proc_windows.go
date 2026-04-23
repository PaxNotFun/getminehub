//go:build windows

package utils

import (
	"os/exec"
	"syscall"
)

// HideWindowAttr configura el proceso para que no abra una ventana de consola en Windows.
// Debe llamarse antes de cmd.Run() o cmd.Start().
func HideWindowAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
