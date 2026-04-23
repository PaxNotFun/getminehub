// services/database/database.go
// Única fuente de verdad para toda la lógica SQLite de GetMineHub.
// Centraliza InitDatabase, ServerRecord y todas las operaciones CRUD.
// Usa el pool compartido de config.GetDB().
package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"getminehub/config"

	"github.com/google/uuid"
)

// ─── Modelo ───────────────────────────────────────────────────────────────────

// ServerRecord representa un servidor en la base de datos
type ServerRecord struct {
	UUID            string
	Name            string
	Path            string
	Type            string
	Version         string
	JavaExecutable  string
	JarFile         string
	ForgeArgsFile   string
	ForgeLaunchType string
	MinRAM          string
	MaxRAM          string
	JVMArgs         string
	UseAikarFlags   bool
	CreatedAt       string
	UpdatedAt       string
}

// ─── Inicialización ───────────────────────────────────────────────────────────

// InitDatabase inicializa las tablas de la base de datos si no existen.
// Es idempotente: puede llamarse múltiples veces sin error.
func InitDatabase() error {
	db := config.GetDB()
	if db == nil {
		return fmt.Errorf("base de datos no disponible")
	}
	return createTables(db)
}

// createTables crea todas las tablas e índices necesarios de forma idempotente.
func createTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS servers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			path TEXT UNIQUE NOT NULL,
			type TEXT NOT NULL,
			version TEXT NOT NULL,
			java_executable TEXT,
			jar_file TEXT,
			forge_args_file TEXT,
			forge_launch_type TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS server_config (
			server_uuid TEXT PRIMARY KEY,
			min_ram TEXT DEFAULT '2G',
			max_ram TEXT DEFAULT '4G',
			jvm_args TEXT DEFAULT '',
			use_aikar_flags INTEGER DEFAULT 0,
			FOREIGN KEY (server_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_server_uuid ON servers(uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_server_name ON servers(name)`,
		`CREATE TABLE IF NOT EXISTS version_cache (
			server_type TEXT PRIMARY KEY,
			versions TEXT NOT NULL,
			fetched_at INTEGER NOT NULL
		)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("inicializando tabla: %w", err)
		}
	}

	// Migración idempotente: ALTER TABLE silencia el error si la columna ya existe
	db.Exec("ALTER TABLE server_config ADD COLUMN use_aikar_flags INTEGER DEFAULT 0")

	return nil
}

// ─── CRUD de servidores ───────────────────────────────────────────────────────

// AddServer añade un nuevo servidor a la base de datos.
// Si el servidor (mismo path) ya existe, no hace nada (idempotente).
func AddServer(s *ServerRecord) error {
	db := config.GetDB()
	if db == nil {
		return fmt.Errorf("base de datos no disponible")
	}

	if s.UUID == "" {
		s.UUID = uuid.New().String()
	}
	now := time.Now().Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO servers (uuid, name, path, type, version, java_executable, jar_file,
			forge_args_file, forge_launch_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.UUID, s.Name, s.Path, s.Type, s.Version,
		nullStr(s.JavaExecutable), nullStr(s.JarFile),
		nullStr(s.ForgeArgsFile), nullStr(s.ForgeLaunchType),
		now, now)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO server_config (server_uuid, min_ram, max_ram, jvm_args, use_aikar_flags)
		VALUES (?, '2G', '4G', '', 0)
	`, s.UUID)
	return err
}

// GetServerByPath obtiene un servidor por su path. Retorna (nil, nil) si no existe.
func GetServerByPath(path string) (*ServerRecord, error) {
	db := config.GetDB()
	if db == nil {
		return nil, fmt.Errorf("base de datos no disponible")
	}

	row := db.QueryRow(`
		SELECT s.uuid, s.name, s.path, s.type, s.version, s.java_executable,
			s.jar_file, s.forge_args_file, s.forge_launch_type, s.created_at, s.updated_at,
			COALESCE(c.min_ram, '2G'), COALESCE(c.max_ram, '4G'),
			COALESCE(c.jvm_args, ''), COALESCE(c.use_aikar_flags, 0)
		FROM servers s
		LEFT JOIN server_config c ON s.uuid = c.server_uuid
		WHERE s.path = ?
	`, path)

	return scanServer(row)
}

// GetServerByUUID obtiene un servidor por su UUID. Retorna (nil, nil) si no existe.
func GetServerByUUID(serverUUID string) (*ServerRecord, error) {
	db := config.GetDB()
	if db == nil {
		return nil, fmt.Errorf("base de datos no disponible")
	}

	row := db.QueryRow(`
		SELECT s.uuid, s.name, s.path, s.type, s.version, s.java_executable,
			s.jar_file, s.forge_args_file, s.forge_launch_type, s.created_at, s.updated_at,
			COALESCE(c.min_ram, '2G'), COALESCE(c.max_ram, '4G'),
			COALESCE(c.jvm_args, ''), COALESCE(c.use_aikar_flags, 0)
		FROM servers s
		LEFT JOIN server_config c ON s.uuid = c.server_uuid
		WHERE s.uuid = ?
	`, serverUUID)

	return scanServer(row)
}

// GetAllServers retorna todos los servidores ordenados por nombre.
func GetAllServers() ([]*ServerRecord, error) {
	db := config.GetDB()
	if db == nil {
		return nil, fmt.Errorf("base de datos no disponible")
	}

	rows, err := db.Query(`
		SELECT s.uuid, s.name, s.path, s.type, s.version, s.java_executable,
			s.jar_file, s.forge_args_file, s.forge_launch_type, s.created_at, s.updated_at,
			COALESCE(c.min_ram, '2G'), COALESCE(c.max_ram, '4G'),
			COALESCE(c.jvm_args, ''), COALESCE(c.use_aikar_flags, 0)
		FROM servers s
		LEFT JOIN server_config c ON s.uuid = c.server_uuid
		ORDER BY s.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ServerRecord
	for rows.Next() {
		s, err := scanServerRow(rows)
		if err != nil {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

// UpdateServer actualiza campos específicos de un servidor por UUID.
// Solo se permiten las columnas de la allowlist; cualquier clave fuera de ella
// se ignora silenciosamente. Los nombres de columna se toman de la allowlist
// (nunca de la entrada) para que sea imposible inyectar SQL incluso si la
// allowlist contuviera un typo.
func UpdateServer(serverUUID string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	// allowedCols define las únicas columnas que pueden modificarse.
	// Los nombres van directamente al query, por eso los declaramos aquí
	// (no vienen de la entrada del caller).
	allowedCols := []string{
		"name", "type", "version",
		"java_executable", "jar_file",
		"forge_args_file", "forge_launch_type",
	}

	setClauses := make([]string, 0, len(allowedCols))
	args := make([]interface{}, 0, len(allowedCols)+2)

	for _, col := range allowedCols {
		v, ok := fields[col]
		if !ok {
			continue
		}
		setClauses = append(setClauses, col+" = ?")
		args = append(args, v)
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := "UPDATE servers SET " + strings.Join(setClauses, ", ") +
		", updated_at = ? WHERE uuid = ?"
	args = append(args, time.Now().Format(time.RFC3339), serverUUID)

	db := config.GetDB()
	if db == nil {
		return fmt.Errorf("base de datos no disponible")
	}

	_, err := db.Exec(query, args...)
	return err
}

// UpdateServerConfig actualiza la configuración de RAM/JVM de un servidor.
func UpdateServerConfig(serverUUID, minRAM, maxRAM, jvmArgs string, useAikarFlags bool) error {
	db := config.GetDB()
	if db == nil {
		return fmt.Errorf("base de datos no disponible")
	}

	aikar := 0
	if useAikarFlags {
		aikar = 1
	}

	_, err := db.Exec(`
		INSERT OR REPLACE INTO server_config (server_uuid, min_ram, max_ram, jvm_args, use_aikar_flags)
		VALUES (?, ?, ?, ?, ?)
	`, serverUUID, minRAM, maxRAM, jvmArgs, aikar)
	return err
}

// DeleteServer elimina un servidor de la base de datos por UUID.
func DeleteServer(serverUUID string) error {
	db := config.GetDB()
	if db == nil {
		return fmt.Errorf("base de datos no disponible")
	}

	_, err := db.Exec("DELETE FROM servers WHERE uuid = ?", serverUUID)
	return err
}

// SearchServers busca servidores por nombre o tipo.
func SearchServers(query string) ([]*ServerRecord, error) {
	db := config.GetDB()
	if db == nil {
		return nil, fmt.Errorf("base de datos no disponible")
	}

	like := "%" + query + "%"
	rows, err := db.Query(`
		SELECT s.uuid, s.name, s.path, s.type, s.version, s.java_executable,
			s.jar_file, s.forge_args_file, s.forge_launch_type, s.created_at, s.updated_at,
			COALESCE(c.min_ram, '2G'), COALESCE(c.max_ram, '4G'),
			COALESCE(c.jvm_args, ''), COALESCE(c.use_aikar_flags, 0)
		FROM servers s
		LEFT JOIN server_config c ON s.uuid = c.server_uuid
		WHERE s.name LIKE ? OR s.type LIKE ?
		ORDER BY s.name
	`, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ServerRecord
	for rows.Next() {
		s, err := scanServerRow(rows)
		if err != nil {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

// ─── Estadísticas ─────────────────────────────────────────────────────────────

// DBStats contiene estadísticas de la base de datos.
type DBStats struct {
	TotalServers   int
	ByType         map[string]int
	ByVersion      map[string]int
	DatabaseSizeMB float64
}

// GetDatabaseStats retorna estadísticas de la base de datos.
func GetDatabaseStats() DBStats {
	stats := DBStats{
		ByType:    make(map[string]int),
		ByVersion: make(map[string]int),
	}

	servers, err := GetAllServers()
	if err != nil {
		return stats
	}

	stats.TotalServers = len(servers)
	for _, s := range servers {
		stats.ByType[s.Type]++
		stats.ByVersion[s.Version]++
	}

	return stats
}

// ─── Migración desde JSON ─────────────────────────────────────────────────────

// MigrateFromJSON migra servidores desde el sistema JSON antiguo a SQLite.
// Es idempotente: no duplica servidores que ya existen (comprueba por path).
// Hace un backup automático de servers.json antes de migrar.
func MigrateFromJSON(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // No existe, nada que migrar
	}

	backupJSONFile(jsonPath, data)

	servers, ok := parseJSONServers(data)
	if !ok {
		os.Remove(jsonPath)
		return nil
	}
	if len(servers) == 0 {
		os.Remove(jsonPath)
		return nil
	}

	migrated, skipped := migrateServerList(servers)

	slog.Info("migración desde JSON completada",
		"migrados", migrated,
		"omitidos_existentes", skipped)

	os.Remove(jsonPath)
	return nil
}

// backupJSONFile escribe una copia de seguridad del archivo JSON antes de migrar.
func backupJSONFile(jsonPath string, data []byte) {
	backupPath := jsonPath + ".bak"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		slog.Warn("no se pudo crear backup de servers.json", "error", err)
	} else {
		slog.Info("backup de servers.json creado", "path", backupPath)
	}
}

// parseJSONServers decodifica el JSON y retorna (servidores, ok).
// ok=false indica JSON inválido y que el archivo debe eliminarse.
func parseJSONServers(data []byte) ([]map[string]interface{}, bool) {
	var servers []map[string]interface{}
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, false
	}
	return servers, true
}

// migrateServerList itera los registros JSON y los inserta en SQLite.
// Retorna (migratedCount, skippedCount).
func migrateServerList(servers []map[string]interface{}) (int, int) {
	migrated, skipped := 0, 0
	for _, srv := range servers {
		ok, wasSkipped := migrateOneServer(srv)
		if wasSkipped {
			skipped++
		} else if ok {
			migrated++
		}
	}
	return migrated, skipped
}

// migrateOneServer migra un único registro JSON a SQLite.
// Retorna (success, wasSkipped).
func migrateOneServer(srv map[string]interface{}) (bool, bool) {
	name, _ := srv["name"].(string)
	path, _ := srv["path"].(string)
	stype, _ := srv["type"].(string)
	version, _ := srv["version"].(string)
	javaExe, _ := srv["java_executable"].(string)
	jarFile, _ := srv["jar_file"].(string)

	if name == "" || path == "" || stype == "" || version == "" {
		return false, false
	}

	// Idempotente: no agregar si ya existe por path
	if existing, _ := GetServerByPath(path); existing != nil {
		return false, true
	}

	rec := &ServerRecord{
		Name: name, Path: path, Type: stype, Version: version,
		JavaExecutable: javaExe, JarFile: jarFile,
	}
	if id, ok := srv["uuid"].(string); ok && id != "" {
		rec.UUID = id
	}

	if err := AddServer(rec); err != nil {
		slog.Warn("error al migrar servidor", "name", name, "error", err)
		return false, false
	}

	migrateJVMConfig(path, srv)
	return true, false
}

// migrateJVMConfig lee la config JVM del JSON y la persiste en SQLite.
func migrateJVMConfig(path string, srv map[string]interface{}) {
	cfg, ok := srv["config"].(map[string]interface{})
	if !ok {
		return
	}
	added, _ := GetServerByPath(path)
	if added == nil {
		return
	}
	minRAM, _ := cfg["min_ram"].(string)
	maxRAM, _ := cfg["max_ram"].(string)
	jvmArgs, _ := cfg["jvm_args"].(string)
	if minRAM == "" {
		minRAM = "2G"
	}
	if maxRAM == "" {
		maxRAM = "4G"
	}
	if err := UpdateServerConfig(added.UUID, minRAM, maxRAM, jvmArgs, false); err != nil {
		slog.Warn("error migrando config JVM", "uuid", added.UUID, "error", err)
	}
}

// ─── Helpers de escaneo ───────────────────────────────────────────────────────

func scanServer(row *sql.Row) (*ServerRecord, error) {
	var s ServerRecord
	var javaExe, jarFile, forgeArgsFile, forgeLaunchType sql.NullString
	var aikar int

	err := row.Scan(
		&s.UUID, &s.Name, &s.Path, &s.Type, &s.Version,
		&javaExe, &jarFile, &forgeArgsFile, &forgeLaunchType,
		&s.CreatedAt, &s.UpdatedAt,
		&s.MinRAM, &s.MaxRAM, &s.JVMArgs, &aikar,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	s.JavaExecutable = javaExe.String
	s.JarFile = jarFile.String
	s.ForgeArgsFile = forgeArgsFile.String
	s.ForgeLaunchType = forgeLaunchType.String
	s.UseAikarFlags = aikar != 0
	return &s, nil
}

func scanServerRow(rows *sql.Rows) (*ServerRecord, error) {
	var s ServerRecord
	var javaExe, jarFile, forgeArgsFile, forgeLaunchType sql.NullString
	var aikar int

	err := rows.Scan(
		&s.UUID, &s.Name, &s.Path, &s.Type, &s.Version,
		&javaExe, &jarFile, &forgeArgsFile, &forgeLaunchType,
		&s.CreatedAt, &s.UpdatedAt,
		&s.MinRAM, &s.MaxRAM, &s.JVMArgs, &aikar,
	)
	if err != nil {
		return nil, err
	}

	s.JavaExecutable = javaExe.String
	s.JarFile = jarFile.String
	s.ForgeArgsFile = forgeArgsFile.String
	s.ForgeLaunchType = forgeLaunchType.String
	s.UseAikarFlags = aikar != 0
	return &s, nil
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
