package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
)

// CurrentVersion es la versión unificada de la aplicación.
const CurrentVersion = "5.0.10"

var (
	AppDir          string
	ServersFilePath string // Solo se mantiene para la migración desde JSON
	SettingsPath    string // Solo se mantiene para la migración desde JSON
	JavaRuntimesDir string
)

func init() {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			home, _ := os.UserHomeDir()
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		AppDir = filepath.Join(appdata, "GetMineHub")
	case "darwin":
		home, _ := os.UserHomeDir()
		AppDir = filepath.Join(home, "Library", "Application Support", "GetMineHub")
	default:
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			home, _ := os.UserHomeDir()
			xdgConfig = filepath.Join(home, ".config")
		}
		AppDir = filepath.Join(xdgConfig, "GetMineHub")
	}

	ServersFilePath = filepath.Join(AppDir, "servers.json")
	SettingsPath    = filepath.Join(AppDir, "settings.json")
	JavaRuntimesDir = filepath.Join(AppDir, "java_runtimes")
}

// Settings representa la configuración global de la aplicación.
type Settings struct {
	ServersBaseDir       string `json:"servers_base_dir"`
	MaxRAMLimit          int    `json:"max_ram_limit"`
	CheckForUpdates      bool   `json:"check_for_updates"`
	Timeout              int    `json:"timeout"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
}

// ─── Caché en memoria de Settings ────────────────────────────────────────────

var (
	settingsCache   *Settings
	settingsCacheMu sync.RWMutex
)

func invalidateSettingsCache() {
	settingsCacheMu.Lock()
	settingsCache = nil
	settingsCacheMu.Unlock()
}

// GetServersBaseDir obtiene el directorio base de los servidores desde SQLite.
func GetServersBaseDir() string {
	s := LoadAllSettings()
	if s.ServersBaseDir != "" {
		return s.ServersBaseDir
	}
	return filepath.Join(AppDir, "servers")
}

// EnsureConfigExists asegura que todos los directorios existan e inicializa la BD.
func EnsureConfigExists() error {
	dirs := []string{AppDir, JavaRuntimesDir}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	// Inicializar el pool de BD y la tabla de settings
	if err := InitSettingsDB(); err != nil {
		return err
	}

	// Asegurar que exista el directorio de servidores
	if err := os.MkdirAll(GetServersBaseDir(), 0755); err != nil {
		return err
	}

	// Insertar defaults solo si la tabla está vacía
	if _, ok := dbGetSetting("servers_base_dir"); !ok {
		defaults := Settings{
			ServersBaseDir:       filepath.Join(AppDir, "servers"),
			MaxRAMLimit:          0,
			CheckForUpdates:      true,
			Timeout:              DefaultTimeout,
			NotificationsEnabled: true,
		}
		if err := SaveAllSettings(defaults); err != nil {
			return fmt.Errorf("guardando configuración por defecto: %w", err)
		}
	}

	return nil
}

// LoadAllSettings carga todas las configuraciones. Usa caché en memoria para
// evitar lecturas repetidas a SQLite.
func LoadAllSettings() Settings {
	settingsCacheMu.RLock()
	if settingsCache != nil {
		s := *settingsCache
		settingsCacheMu.RUnlock()
		return s
	}
	settingsCacheMu.RUnlock()

	s := loadSettingsFromDB()

	settingsCacheMu.Lock()
	settingsCache = &s
	settingsCacheMu.Unlock()
	return s
}

func loadSettingsFromDB() Settings {
	s := Settings{
		ServersBaseDir:       filepath.Join(AppDir, "servers"),
		MaxRAMLimit:          0,
		CheckForUpdates:      true,
		Timeout:              DefaultTimeout,
		NotificationsEnabled: true,
	}

	if v, ok := dbGetSetting("servers_base_dir"); ok && v != "" {
		s.ServersBaseDir = v
	}
	if v, ok := dbGetSetting("max_ram_limit"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			s.MaxRAMLimit = n
		}
	}
	if v, ok := dbGetSetting("check_for_updates"); ok {
		s.CheckForUpdates = (v != "false" && v != "0")
	}
	if v, ok := dbGetSetting("timeout"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.Timeout = n
		}
	}
	if v, ok := dbGetSetting("notifications_enabled"); ok {
		s.NotificationsEnabled = (v != "false" && v != "0")
	}
	return s
}

// SaveAllSettings guarda todas las configuraciones en SQLite e invalida la caché.
func SaveAllSettings(s Settings) error {
	checkUpdates := "false"
	if s.CheckForUpdates {
		checkUpdates = "true"
	}
	notifs := "false"
	if s.NotificationsEnabled {
		notifs = "true"
	}

	pairs := [][2]string{
		{"servers_base_dir", s.ServersBaseDir},
		{"max_ram_limit", strconv.Itoa(s.MaxRAMLimit)},
		{"check_for_updates", checkUpdates},
		{"timeout", strconv.Itoa(s.Timeout)},
		{"notifications_enabled", notifs},
	}
	for _, kv := range pairs {
		if err := dbSetSetting(kv[0], kv[1]); err != nil {
			return fmt.Errorf("guardando configuración: %w", err)
		}
	}

	// Actualizar caché en memoria
	settingsCacheMu.Lock()
	cp := s
	settingsCache = &cp
	settingsCacheMu.Unlock()
	return nil
}

// SaveSetting guarda una configuración específica en SQLite.
// Los errores se registran en el log en lugar de silenciarse,
// ya que esta función se llama en contextos donde no se puede retornar error.
func SaveSetting(key string, value any) {
	// last_server_path se guarda directamente sin pasar por SaveAllSettings
	if key == "last_server_path" {
		if v, ok := value.(string); ok {
			if err := dbSetSetting("last_server_path", v); err != nil {
				slog.Warn("error guardando last_server_path", "error", err)
			}
		}
		return
	}

	// El resto mutan un campo de Settings y llaman SaveAllSettings
	mutator := settingMutator(key, value)
	if mutator == nil {
		return // clave desconocida
	}

	s := LoadAllSettings()
	mutator(&s)
	if err := SaveAllSettings(s); err != nil {
		slog.Warn("error guardando setting", "key", key, "error", err)
	}
}

// settingMutator retorna una función que muta el campo correcto de Settings
// según la clave dada. Retorna nil si la clave no es reconocida o el tipo no coincide.
func settingMutator(key string, value any) func(*Settings) {
	switch key {
	case "servers_base_dir":
		if v, ok := value.(string); ok {
			return func(s *Settings) { s.ServersBaseDir = v }
		}
	case "check_for_updates":
		if v, ok := value.(bool); ok {
			return func(s *Settings) { s.CheckForUpdates = v }
		}
	case "notifications_enabled":
		if v, ok := value.(bool); ok {
			return func(s *Settings) { s.NotificationsEnabled = v }
		}
	case "timeout":
		if v, ok := value.(int); ok {
			return func(s *Settings) { s.Timeout = v }
		}
	case "max_ram_limit":
		if v, ok := value.(int); ok {
			return func(s *Settings) { s.MaxRAMLimit = v }
		}
	}
	return nil
}

// GetLastServerPath obtiene el último servidor abierto desde SQLite.
func GetLastServerPath() string {
	val, _ := dbGetSetting("last_server_path")
	return val
}
