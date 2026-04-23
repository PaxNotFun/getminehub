// GetMineHub — Gestor de Servidores Minecraft
// Migrado de Fyne a Wails v2
// Versión 3.4.0
package main

import (
	"embed"
	"log"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"getminehub/config"
	dbpkg "getminehub/services/database"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Iniciando GetMineHub", "version", config.CurrentVersion)

	if err := config.EnsureConfigExists(); err != nil {
		log.Fatalf("Error inicializando configuración: %v", err)
	}

	// InitDatabase centralizado en services/database
	if err := dbpkg.InitDatabase(); err != nil {
		slog.Warn("InitDatabase retornó error (posiblemente ya inicializada)", "error", err)
	}

	// Migrar desde JSON legado si todavía existe (idempotente + backup automático)
	if _, err := os.Stat(config.ServersFilePath); err == nil {
		if err := dbpkg.MigrateFromJSON(config.ServersFilePath); err != nil {
			slog.Warn("error en migración desde JSON", "error", err)
		}
	}

	app := NewApp()

	if err := wails.Run(&options.App{
		Title:     "GetMineHub",
		Width:     1280,
		Height:    745,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:         &options.RGBA{R: 10, G: 10, B: 15, A: 255},
		OnStartup:                app.startup,
		OnShutdown:               app.shutdown,
		Bind:                     []interface{}{app},
		EnableDefaultContextMenu: false,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "GetMineHub",
				Message: "Gestor de Servidores Minecraft v" + config.CurrentVersion,
			},
		},
		Linux: &linux.Options{
			WindowIsTranslucent: false,
		},
	}); err != nil {
		log.Fatal(err)
	}
}
