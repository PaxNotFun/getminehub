//go:build !windows

package utils

import "os/exec"

// HideWindowAttr es un no-op en plataformas que no son Windows.
func HideWindowAttr(cmd *exec.Cmd) {}
