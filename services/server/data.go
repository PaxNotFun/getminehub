// services/server/data.go — Helpers de alto nivel para gestión de servidores.
// Toda la lógica de base de datos está en services/database/database.go.
// Este archivo solo contiene helpers de archivos/directorios y wrappers.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"getminehub/config"
	"getminehub/services/database"

	"github.com/google/uuid"
)

// ─── Alias de tipo: toda la app sigue usando server.ServerRecord ──────────────

// ServerRecord es un alias de database.ServerRecord.
// Esto mantiene compatibilidad con todo el código existente que usa srvpkg.ServerRecord.
type ServerRecord = database.ServerRecord

// ─── Wrappers sobre database.*  ──────────────────────────────────────────────

// EnsureDatabaseInitialized asegura que la base de datos esté inicializada.
func EnsureDatabaseInitialized() error {
	return database.InitDatabase()
}

// LoadServers carga todos los servidores desde la base de datos.
func LoadServers() ([]*ServerRecord, error) {
	if err := EnsureDatabaseInitialized(); err != nil {
		return nil, err
	}
	return database.GetAllServers()
}

// GetAllServers retorna todos los servidores ordenados por nombre.
func GetAllServers() ([]*ServerRecord, error) {
	return database.GetAllServers()
}

// GetServerByPath obtiene un servidor por su path.
func GetServerByPath(path string) (*ServerRecord, error) {
	return database.GetServerByPath(path)
}

// GetServerByUUID obtiene un servidor por su UUID.
func GetServerByUUID(id string) (*ServerRecord, error) {
	return database.GetServerByUUID(id)
}

// AddServer añade un servidor a la base de datos.
func AddServer(s *ServerRecord) error {
	return database.AddServer(s)
}

// UpdateServer actualiza campos de un servidor.
func UpdateServer(serverUUID string, fields map[string]interface{}) error {
	return database.UpdateServer(serverUUID, fields)
}

// UpdateServerConfig actualiza la configuración RAM/JVM.
func UpdateServerConfig(serverUUID, minRAM, maxRAM, jvmArgs string, useAikarFlags bool) error {
	return database.UpdateServerConfig(serverUUID, minRAM, maxRAM, jvmArgs, useAikarFlags)
}

// DeleteServer elimina un servidor de la base de datos.
func DeleteServer(serverUUID string) error {
	return database.DeleteServer(serverUUID)
}

// GetDatabaseStats retorna estadísticas de la base de datos.
func GetDatabaseStats() database.DBStats {
	return database.GetDatabaseStats()
}

// ─── Helpers de carpeta y configuración ──────────────────────────────────────

// GenerateServerFolderName genera un nombre único de carpeta basado en UUID.
func GenerateServerFolderName() string {
	return uuid.New().String()
}

// LoadServerConfig carga la configuración de JVM de un servidor desde la BD.
func LoadServerConfig(serverPath string) (minRAM, maxRAM, jvmArgs string, useAikarFlags bool) {
	s, err := database.GetServerByPath(serverPath)
	if err != nil || s == nil {
		return "2G", "4G", "", false
	}
	minRAM = s.MinRAM
	if minRAM == "" {
		minRAM = "2G"
	}
	maxRAM = s.MaxRAM
	if maxRAM == "" {
		maxRAM = "4G"
	}
	return minRAM, maxRAM, s.JVMArgs, s.UseAikarFlags
}

// SaveServerConfig guarda la configuración de JVM de un servidor.
func SaveServerConfig(serverPath, minRAM, maxRAM, jvmArgs string, useAikarFlags bool) error {
	s, err := database.GetServerByPath(serverPath)
	if err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("servidor no encontrado en la base de datos")
	}
	return database.UpdateServerConfig(s.UUID, minRAM, maxRAM, jvmArgs, useAikarFlags)
}

// AcceptEULA escribe eula=true en el servidor.
func AcceptEULA(serverPath string) error {
	eulaPath := filepath.Join(serverPath, "eula.txt")
	return os.WriteFile(eulaPath, []byte("eula=true\n"), 0644)
}

// DetectMinecraftWorlds detecta todos los mundos en el servidor.
func DetectMinecraftWorlds(serverPath string) []string {
	var worlds []string
	entries, err := os.ReadDir(serverPath)
	if err != nil {
		return worlds
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(serverPath, e.Name())
		if isMinecraftWorld(dir) {
			worlds = append(worlds, e.Name())
		}
	}
	return worlds
}

func isMinecraftWorld(dir string) bool {
	checks := []string{"level.dat", "level.dat_old", "session.lock"}
	for _, f := range checks {
		if info, err := os.Stat(filepath.Join(dir, f)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// ReinstallItems contiene los items detectados para preservar en reinstalación parcial.
type ReinstallItems struct {
	Worlds  []string
	Folders []string
	Files   []string
}

// GetItemsToPreserve obtiene los items a preservar según el modo de reinstalación.
func GetItemsToPreserve(serverPath string) ReinstallItems {
	result := ReinstallItems{}

	importantFolders := map[string]bool{
		"plugins":  true,
		"mods":     true,
		"config":   true,
		"versions": true,
	}

	importantFiles := map[string]bool{
		"server.properties":   true,
		"eula.txt":            true,
		"ops.json":            true,
		"whitelist.json":      true,
		"banned-players.json": true,
		"banned-ips.json":     true,
		"permissions.yml":     true,
		"bukkit.yml":          true,
		"spigot.yml":          true,
		"paper.yml":           true,
		"server-icon.png":     true,
		"user_cache.json":     true,
		"usernamecache.json":  true,
	}

	entries, err := os.ReadDir(serverPath)
	if err != nil {
		return result
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if isMinecraftWorld(filepath.Join(serverPath, name)) {
				result.Worlds = append(result.Worlds, name)
			} else if importantFolders[name] {
				result.Folders = append(result.Folders, name)
			}
		} else {
			if importantFiles[name] {
				result.Files = append(result.Files, name)
			}
		}
	}
	return result
}

// PrepareReinstall prepara el servidor para reinstalación o actualización.
// Delega en prepareTotal (modo "total") o preparePartial (modo "partial").
// Retorna (backupPath, messageForUI, error).
func PrepareReinstall(serverPath, reinstallMode string) (string, string, error) {
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("la ruta del servidor no existe")
	}
	if reinstallMode == "total" {
		return prepareTotal(serverPath)
	}
	return preparePartial(serverPath)
}

// prepareTotal hace un backup completo del directorio, lo vacía y guarda la config JVM.
func prepareTotal(serverPath string) (string, string, error) {
	parentDir := filepath.Dir(serverPath)
	baseName := filepath.Base(serverPath)
	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(parentDir, ".backup_"+timestamp+"_"+baseName)

	os.RemoveAll(backupDir)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", "", fmt.Errorf("creando directorio de backup: %w", err)
	}

	entries, err := os.ReadDir(serverPath)
	if err != nil {
		return "", "", fmt.Errorf("leyendo directorio del servidor: %w", err)
	}
	for _, e := range entries {
		src := filepath.Join(serverPath, e.Name())
		dst := filepath.Join(backupDir, e.Name())
		if err := copyPath(src, dst); err != nil {
			os.RemoveAll(backupDir)
			return "", "", fmt.Errorf("haciendo backup de %s: %w", e.Name(), err)
		}
	}

	// Persistir config JVM para restaurarla después de la reinstalación
	saveJVMConfigToBackup(serverPath, backupDir)

	os.RemoveAll(serverPath)
	os.MkdirAll(serverPath, 0755)

	return backupDir, fmt.Sprintf("Backup completo guardado en %s", filepath.Base(backupDir)), nil
}

// preparePartial mueve mundos/plugins/configs a un directorio temporal y limpia el resto.
func preparePartial(serverPath string) (string, string, error) {
	parentDir := filepath.Dir(serverPath)
	baseName := filepath.Base(serverPath)
	tempBackup := filepath.Join(parentDir, ".temp_reinstall_"+baseName)

	os.RemoveAll(tempBackup)
	if err := os.MkdirAll(tempBackup, 0755); err != nil {
		return "", "", fmt.Errorf("creando directorio temporal: %w", err)
	}

	items := GetItemsToPreserve(serverPath)
	allItems := append(append(items.Worlds, items.Folders...), items.Files...)
	if len(allItems) == 0 {
		os.RemoveAll(tempBackup)
		return "", "", fmt.Errorf("no se detectaron items para conservar")
	}

	backedUp, err := backupItems(serverPath, tempBackup, allItems)
	if err != nil {
		os.RemoveAll(tempBackup)
		return "", "", err
	}

	clearNonBackedUp(serverPath, backedUp)

	msg := fmt.Sprintf("Respaldados %d items (%d mundos detectados)", len(backedUp), len(items.Worlds))
	return tempBackup, msg, nil
}

// saveJVMConfigToBackup persiste la config JVM en local_config.json dentro del backup.
func saveJVMConfigToBackup(serverPath, backupDir string) {
	minRAM, maxRAM, jvmArgs, useAikarFlags := LoadServerConfig(serverPath)
	cfg := map[string]interface{}{
		"min_ram": minRAM, "max_ram": maxRAM,
		"jvm_args": jvmArgs, "use_aikar_flags": useAikarFlags,
	}
	if data, err := json.Marshal(cfg); err == nil {
		os.WriteFile(filepath.Join(backupDir, "local_config.json"), data, 0644)
	}
}

// backupItems copia cada item de la lista al directorio de backup.
// Retorna la lista de items copiados exitosamente.
func backupItems(serverPath, tempBackup string, items []string) ([]string, error) {
	var backedUp []string
	for _, itemName := range items {
		src := filepath.Join(serverPath, itemName)
		dst := filepath.Join(tempBackup, itemName)
		if err := copyPath(src, dst); err != nil {
			return nil, fmt.Errorf("copiando %s al backup: %w", itemName, err)
		}
		backedUp = append(backedUp, itemName)
	}
	return backedUp, nil
}

// clearNonBackedUp elimina del serverPath los archivos/dirs que no están en backedUp.
func clearNonBackedUp(serverPath string, backedUp []string) {
	backedUpSet := make(map[string]bool, len(backedUp))
	for _, b := range backedUp {
		backedUpSet[b] = true
	}
	entries, _ := os.ReadDir(serverPath)
	for _, e := range entries {
		if backedUpSet[e.Name()] {
			continue
		}
		p := filepath.Join(serverPath, e.Name())
		if e.IsDir() {
			os.RemoveAll(p)
		} else {
			os.Remove(p)
		}
	}
}

// RestoreFromTempReinstall restaura los archivos desde el backup temporal.
func RestoreFromTempReinstall(serverPath, tempBackupPath string) error {
	if _, err := os.Stat(tempBackupPath); os.IsNotExist(err) {
		return fmt.Errorf("no existe el backup temporal")
	}

	entries, err := os.ReadDir(tempBackupPath)
	if err != nil {
		return err
	}

	for _, e := range entries {
		src := filepath.Join(tempBackupPath, e.Name())
		dst := filepath.Join(serverPath, e.Name())
		os.RemoveAll(dst)
		if err := copyPath(src, dst); err != nil {
			return fmt.Errorf("error restaurando %s: %w", e.Name(), err)
		}
	}

	os.RemoveAll(tempBackupPath)
	return nil
}

// CleanupTempReinstall limpia el backup temporal. Los backups permanentes no se eliminan.
func CleanupTempReinstall(tempBackupPath string) {
	if tempBackupPath == "" {
		return
	}
	base := filepath.Base(tempBackupPath)
	if len(base) >= 8 && base[:8] == ".backup_" {
		return
	}
	os.RemoveAll(tempBackupPath)
}

// DeleteServerFiles elimina un servidor de la BD y opcionalmente sus archivos.
func DeleteServerFiles(s *ServerRecord, deleteFiles bool) (bool, string) {
	if err := database.DeleteServer(s.UUID); err != nil {
		return false, fmt.Sprintf("Error al eliminar de la base de datos: %v", err)
	}

	if deleteFiles {
		if err := os.RemoveAll(s.Path); err != nil {
			return true, fmt.Sprintf("Servidor eliminado de la lista, pero hubo un error al eliminar archivos: %v", err)
		}
		return true, fmt.Sprintf("Servidor '%s' eliminado completamente.", s.Name)
	}

	return true, fmt.Sprintf("Servidor '%s' eliminado de la lista.", s.Name)
}

// GetDirectorySize calcula el tamaño de un directorio en bytes.
func GetDirectorySize(path string) int64 {
	var total int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// GetRecentServers retorna los N servidores más recientes.
func GetRecentServers(n int) []*ServerRecord {
	all, err := database.GetAllServers()
	if err != nil || len(all) == 0 {
		return nil
	}

	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[i].UpdatedAt < all[j].UpdatedAt {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// GetOrCreateServerPath retorna una nueva ruta de servidor única.
func GetOrCreateServerPath() string {
	return filepath.Join(config.GetServersBaseDir(), GenerateServerFolderName())
}

// ─── Helpers de copia de archivos ────────────────────────────────────────────

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyPath(
			filepath.Join(src, e.Name()),
			filepath.Join(dst, e.Name()),
		); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
