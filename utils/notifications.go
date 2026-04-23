package utils

import (
	"os/exec"
	"runtime"
)

// SendNotification envía una notificación de escritorio cross-platform
func SendNotification(title, message string) {
	go func() {
		switch runtime.GOOS {
		case "linux":
			// notify-send está disponible en la mayoría de distribuciones Linux con libnotify
			exec.Command("notify-send", "--app-name=GetMineHub", "--icon=dialog-information", title, message).Run()
		case "darwin":
			script := `display notification "` + message + `" with title "` + title + `"`
			exec.Command("osascript", "-e", script).Run()
		case "windows":
			// En Windows usamos PowerShell para notificaciones toast
			ps := `Add-Type -AssemblyName System.Windows.Forms; ` +
				`$notify = New-Object System.Windows.Forms.NotifyIcon; ` +
				`$notify.Icon = [System.Drawing.SystemIcons]::Information; ` +
				`$notify.Visible = $true; ` +
				`$notify.ShowBalloonTip(5000, "` + title + `", "` + message + `", [System.Windows.Forms.ToolTipIcon]::Info)`
			exec.Command("powershell", "-Command", ps).Run()
		}
	}()
}
