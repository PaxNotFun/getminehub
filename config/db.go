// config/db.go — Pool único de SQLite para toda la aplicación.
// Reemplaza el patrón de abrir/cerrar conexión por llamada.
package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	_ "modernc.org/sqlite"
)

// ─── Pool global ──────────────────────────────────────────────────────────────

var (
	dbOnce   sync.Once
	globalDB *sql.DB
)

// GetDB devuelve el pool compartido de SQLite.
// El pool se crea una sola vez (singleton) y se reutiliza durante toda la
// vida de la aplicación. SQLite con WAL admite múltiples lectores concurrentes
// y un solo escritor; SetMaxOpenConns(1) garantiza ese invariante.
func GetDB() *sql.DB {
	dbOnce.Do(func() {
		path := filepath.Join(AppDir, "servers.db")
		db, err := sql.Open("sqlite", path)
		if err != nil {
			slog.Error("no se pudo abrir la base de datos", "path", path, "error", err)
			return
		}
		// SQLite no soporta múltiples escritores simultáneos.
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		// Habilitar WAL y busy_timeout para mayor robustez.
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA busy_timeout=5000")
		db.Exec("PRAGMA foreign_keys=ON")

		globalDB = db
	})
	return globalDB
}

// ─── Tabla app_settings ───────────────────────────────────────────────────────

// InitSettingsDB crea la tabla app_settings y migra desde archivos JSON/txt si existen.
func InitSettingsDB() error {
	db := GetDB()
	if db == nil {
		return nil // No se pudo abrir la BD; la app seguirá con valores por defecto
	}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS app_settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		return err
	}

	// Migración desde settings.json (si todavía existe)
	migrateSettingsJSON(db)
	// Migración desde last_server.txt (si todavía existe)
	migrateLastServerTxt(db)

	return nil
}

// migrateSettingsJSON importa settings.json a SQLite y lo elimina.
func migrateSettingsJSON(db *sql.DB) {
	data, err := os.ReadFile(SettingsPath)
	if err != nil {
		return
	}

	var m map[string]interface{}
	if json.Unmarshal(data, &m) != nil {
		os.Remove(SettingsPath)
		return
	}

	for k, v := range m {
		var val string
		switch t := v.(type) {
		case string:
			val = t
		case bool:
			if t {
				val = "true"
			} else {
				val = "false"
			}
		case float64:
			val = strconv.FormatFloat(t, 'f', -1, 64)
		default:
			continue
		}
		db.Exec(`INSERT OR IGNORE INTO app_settings (key, value) VALUES (?, ?)`, k, val)
	}
	os.Remove(SettingsPath)
	slog.Info("migración desde settings.json completada")
}

// migrateLastServerTxt importa last_server.txt a SQLite y lo elimina.
func migrateLastServerTxt(db *sql.DB) {
	txtPath := filepath.Join(AppDir, "last_server.txt")
	data, err := os.ReadFile(txtPath)
	if err != nil {
		return
	}
	val := string(data)
	if val != "" {
		db.Exec(`INSERT OR IGNORE INTO app_settings (key, value) VALUES (?, ?)`, "last_server_path", val)
	}
	os.Remove(txtPath)
}

// ─── Helpers internos ─────────────────────────────────────────────────────────

// dbGetSetting obtiene un valor de la tabla app_settings. Devuelve ("", false) si no existe.
func dbGetSetting(key string) (string, bool) {
	db := GetDB()
	if db == nil {
		return "", false
	}

	var val string
	err := db.QueryRow("SELECT value FROM app_settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		return "", false
	}
	return val, true
}

// dbSetSetting guarda o actualiza un valor en la tabla app_settings.
func dbSetSetting(key, value string) error {
	db := GetDB()
	if db == nil {
		return fmt.Errorf("base de datos no disponible al guardar clave '%s'", key)
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO app_settings (key, value) VALUES (?, ?)`, key, value)
	if err != nil {
		return fmt.Errorf("guardando setting '%s': %w", key, err)
	}
	return nil
}
