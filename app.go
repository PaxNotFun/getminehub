// app.go — Controlador principal de GetMineHub (Wails v2)
// Responsabilidad: bootstrap de la app + conexión entre frontend y servicios.
// Las secciones largas de lógica de negocio viven en sus paquetes respectivos.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"

	"getminehub/config"
	"getminehub/services/installers"
	javapkg "getminehub/services/java"
	srvpkg "getminehub/services/server"
	vcs "getminehub/services/versions"
	"getminehub/utils"
)

// ─── Tipos de datos para el frontend ─────────────────────────────────────────

type DashboardData struct {
	TotalServers int     `json:"totalServers"`
	UsedGB       float64 `json:"usedGB"`
	MostType     string  `json:"mostType"`
	MostVersion  string  `json:"mostVersion"`
}

type JVMConfig struct {
	MinRAM   string `json:"minRAM"`
	MaxRAM   string `json:"maxRAM"`
	JVMArgs  string `json:"jvmArgs"`
	UseAikar bool   `json:"useAikar"`
	JavaExe  string `json:"javaExe"`
}

type UpdateCheckResult struct {
	Available bool   `json:"available"`
	Latest    string `json:"latest"`
	Current   string `json:"current"`
}

type InstallProgressEvent struct {
	Progress     float64 `json:"progress"`
	Text         string  `json:"text"`
	Error        string  `json:"error,omitempty"`
	Success      bool    `json:"success"`
	WasUpdate    bool    `json:"wasUpdate"`
	WasReinstall bool    `json:"wasReinstall"`
	ServerName   string  `json:"serverName,omitempty"`
}

type DeleteResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type PropertyEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// JavaRuntimeInfo contiene información sobre un Java instalado localmente.
type JavaRuntimeInfo struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	Path    string `json:"path"`
}

// JavaDownloadOption representa una versión de Java disponible para descargar.
type JavaDownloadOption struct {
	Name        string `json:"name"`
	JavaVersion int    `json:"javaVersion"`
	MCVersion   string `json:"mcVersion"`
	Installed   bool   `json:"installed"`
}

// JavaDownloadProgress es el evento de progreso de descarga de Java.
type JavaDownloadProgress struct {
	Progress    float64 `json:"progress"`
	Text        string  `json:"text"`
	Error       string  `json:"error,omitempty"`
	Success     bool    `json:"success"`
	JavaVersion int     `json:"javaVersion"`
	JavaPath    string  `json:"javaPath,omitempty"`
}

// ─── App struct (ligado a Wails) ──────────────────────────────────────────────

type App struct {
	ctx            context.Context
	activeServer   *srvpkg.ServerRecord
	serverManager  *srvpkg.ServerManager
	restartPending bool
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go func() {
		time.Sleep(300 * time.Millisecond)
		vcs.PrefetchAllInBackground(nil)
	}()
}

// shutdown se llama cuando Wails está cerrando la app.
// Mata el servidor activo y espera brevemente a que el proceso termine.
func (a *App) shutdown(_ context.Context) {
	if a.serverManager == nil || !a.serverManager.IsRunning() {
		return
	}
	a.serverManager.Kill()
	waitForStop(a.serverManager, 2*time.Second)
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

func (a *App) GetDashboardData() DashboardData {
	stats := srvpkg.GetDatabaseStats()

	mostType := mostFrequent(stats.ByType)
	mostVer := mostFrequent(stats.ByVersion)

	servers, _ := srvpkg.GetAllServers()
	var totalBytes int64
	for _, s := range servers {
		totalBytes += srvpkg.GetDirectorySize(s.Path)
	}

	return DashboardData{
		TotalServers: stats.TotalServers,
		UsedGB:       float64(totalBytes) / (1024 * 1024 * 1024),
		MostType:     mostType,
		MostVersion:  mostVer,
	}
}

func (a *App) GetRecentServers() []*srvpkg.ServerRecord {
	return srvpkg.GetRecentServers(3)
}

func (a *App) GetAllServers() ([]*srvpkg.ServerRecord, error) {
	return srvpkg.LoadServers()
}

// ─── Gestión de servidores ────────────────────────────────────────────────────

func (a *App) OpenServer(path string) (*srvpkg.ServerRecord, error) {
	servers, err := srvpkg.LoadServers()
	if err != nil {
		return nil, fmt.Errorf("cargando servidores: %w", err)
	}
	for _, s := range servers {
		if s.Path != path {
			continue
		}
		a.activeServer = s
		config.SaveSetting("last_server_path", s.Path)
		if a.serverManager == nil || a.serverManager.ServerInfo.Path != s.Path {
			a.serverManager = a.newServerManager(s)
		}
		return s, nil
	}
	return nil, fmt.Errorf("servidor no encontrado: %s", path)
}

func (a *App) GetActiveServer() *srvpkg.ServerRecord { return a.activeServer }

func (a *App) GetLastServerPath() string { return config.GetLastServerPath() }

// ─── Control del servidor ─────────────────────────────────────────────────────

func (a *App) StartServer() error {
	if a.serverManager == nil {
		return fmt.Errorf("no hay servidor activo")
	}
	if a.serverManager.IsRunning() {
		return fmt.Errorf("el servidor ya está encendido")
	}
	go a.serverManager.Start()
	return nil
}

func (a *App) StopServer() error {
	if a.serverManager == nil || !a.serverManager.IsRunning() {
		return fmt.Errorf("el servidor no está encendido")
	}
	s := config.LoadAllSettings()
	if s.NotificationsEnabled && a.activeServer != nil {
		utils.SendNotification("Servidor Apagado",
			fmt.Sprintf("El servidor '%s' se está apagando.", a.activeServer.Name))
	}
	go a.serverManager.Stop()
	return nil
}

func (a *App) RestartServer() error {
	if a.serverManager == nil {
		return fmt.Errorf("no hay servidor activo")
	}
	if a.serverManager.IsRunning() {
		a.restartPending = true
		go a.serverManager.Stop()
	} else {
		go a.serverManager.Start()
	}
	return nil
}

func (a *App) SendCommand(cmd string) error {
	if a.serverManager == nil || !a.serverManager.IsRunning() {
		return fmt.Errorf("el servidor no está corriendo")
	}
	a.serverManager.SendCommand(cmd)
	return nil
}

func (a *App) GetConsoleHistory() string {
	if a.serverManager == nil {
		return ""
	}
	return a.serverManager.GetHistory()
}

func (a *App) IsServerRunning() bool {
	return a.serverManager != nil && a.serverManager.IsRunning()
}

// CloseServer cierra el servidor activo de forma segura antes de volver al menú.
func (a *App) CloseServer() {
	if a.serverManager != nil && a.serverManager.IsRunning() {
		a.serverManager.Kill()
		waitForStop(a.serverManager, 1500*time.Millisecond)
	}
	a.serverManager = nil
	a.activeServer = nil
	a.restartPending = false
}

// ─── Instalación ─────────────────────────────────────────────────────────────

func (a *App) GetVersions(serverType string) []string {
	if cached := vcs.GetVersionsCached(serverType); cached != nil {
		return cached
	}
	return vcs.GetVersions(serverType)
}

func (a *App) CheckInternetConnection() bool {
	return utils.CheckInternetConnection()
}

// InstallServer instala un servidor nuevo. Emite eventos install:progress.
func (a *App) InstallServer(name, stype, version string) {
	serverPath := filepath.Join(config.GetServersBaseDir(), srvpkg.GenerateServerFolderName())
	a.runInstallation(name, stype, version, serverPath, "partial", false, false)
}

// ReinstallServer reinstala el servidor activo. Emite eventos install:progress.
func (a *App) ReinstallServer(mode string) {
	if a.activeServer == nil {
		wailsrt.EventsEmit(a.ctx, "install:progress", InstallProgressEvent{
			Error: "No hay servidor activo para reinstalar.",
		})
		return
	}
	a.runInstallation(a.activeServer.Name, a.activeServer.Type,
		a.activeServer.Version, a.activeServer.Path, mode, true, false)
}

// UpdateServer actualiza el servidor activo a la versión indicada. Emite eventos install:progress.
func (a *App) UpdateServer(targetVersion string) {
	if a.activeServer == nil {
		wailsrt.EventsEmit(a.ctx, "install:progress", InstallProgressEvent{
			Error: "No hay servidor activo para actualizar.",
		})
		return
	}
	a.runInstallation(a.activeServer.Name, a.activeServer.Type,
		targetVersion, a.activeServer.Path, "partial", false, true)
}

func (a *App) runInstallation(name, stype, version, serverPath, reinstallMode string, isReinstall, isUpdate bool) {
	info := &installers.ServerInstallInfo{
		Name:          name,
		Type:          stype,
		Version:       version,
		Path:          serverPath,
		ReinstallMode: reinstallMode,
		OriginalPath:  serverPath,
	}

	inst, err := installers.GetInstaller(info)
	if err != nil {
		wailsrt.EventsEmit(a.ctx, "install:progress", InstallProgressEvent{Error: err.Error()})
		return
	}

	go installers.RunInstallation(inst, serverPath,
		func(status installers.ProgressStatus) {
			evt := InstallProgressEvent{
				Progress:     status.Progress,
				Text:         status.Text,
				Error:        status.Error,
				Success:      status.Success,
				WasUpdate:    status.WasUpdate,
				WasReinstall: status.WasReinstall,
			}
			if status.Success && status.ServerData != nil {
				evt.ServerName = status.ServerData.Name
				a.activeServer = status.ServerData
				if a.serverManager == nil || a.serverManager.ServerInfo.Path != status.ServerData.Path {
					a.serverManager = a.newServerManager(status.ServerData)
				}
				config.SaveSetting("last_server_path", status.ServerData.Path)
			}
			wailsrt.EventsEmit(a.ctx, "install:progress", evt)
		},
		isReinstall, isUpdate,
	)
}

// DeleteServer elimina el servidor activo.
func (a *App) DeleteServer(deleteFiles bool) DeleteResult {
	if a.activeServer == nil {
		return DeleteResult{false, "No hay servidor activo"}
	}
	if a.serverManager != nil && a.serverManager.IsRunning() {
		return DeleteResult{false, "Debes detener el servidor antes de eliminarlo."}
	}
	success, message := srvpkg.DeleteServerFiles(a.activeServer, deleteFiles)
	if success {
		a.serverManager = nil
		a.activeServer = nil
	}
	return DeleteResult{success, message}
}

// ─── Configuración JVM ────────────────────────────────────────────────────────

func (a *App) GetJVMConfig() JVMConfig {
	if a.activeServer == nil {
		return JVMConfig{MinRAM: config.DefaultMinRAM, MaxRAM: config.DefaultMaxRAM}
	}
	minRAM, maxRAM, jvmArgs, useAikar := srvpkg.LoadServerConfig(a.activeServer.Path)
	return JVMConfig{
		MinRAM:   minRAM,
		MaxRAM:   maxRAM,
		JVMArgs:  jvmArgs,
		UseAikar: useAikar,
		JavaExe:  a.activeServer.JavaExecutable,
	}
}

func (a *App) SaveJVMConfig(minRAM, maxRAM, jvmArgs string, useAikar bool) error {
	if a.activeServer == nil {
		return fmt.Errorf("no hay servidor activo")
	}
	return srvpkg.SaveServerConfig(a.activeServer.Path, minRAM, maxRAM, jvmArgs, useAikar)
}

// SaveServerJava guarda el ejecutable de Java para el servidor activo.
func (a *App) SaveServerJava(javaExe string) error {
	if a.activeServer == nil {
		return fmt.Errorf("no hay servidor activo")
	}
	if err := srvpkg.UpdateServer(a.activeServer.UUID, map[string]interface{}{
		"java_executable": javaExe,
	}); err != nil {
		return fmt.Errorf("actualizando java en BD: %w", err)
	}
	a.activeServer.JavaExecutable = javaExe
	if a.serverManager != nil {
		a.serverManager.ServerInfo.JavaExecutable = javaExe
	}
	return nil
}

// ─── server.properties ───────────────────────────────────────────────────────

func (a *App) GetServerProperties() ([]PropertyEntry, error) {
	if a.activeServer == nil {
		return nil, fmt.Errorf("no hay servidor activo")
	}
	propsPath := filepath.Join(a.activeServer.Path, "server.properties")
	f, err := os.Open(propsPath)
	if err != nil {
		return nil, fmt.Errorf("no se encontró server.properties: %w", err)
	}
	defer f.Close()

	var entries []PropertyEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			entries = append(entries, PropertyEntry{
				Key:   strings.TrimSpace(k),
				Value: strings.TrimSpace(v),
			})
		}
	}
	return entries, scanner.Err()
}

func (a *App) SaveServerProperties(props []PropertyEntry) error {
	if a.activeServer == nil {
		return fmt.Errorf("no hay servidor activo")
	}
	propsPath := filepath.Join(a.activeServer.Path, "server.properties")

	// Preservar los comentarios de cabecera del archivo original
	var headerComments []string
	if f, err := os.Open(propsPath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "#") || line == "" {
				headerComments = append(headerComments, line)
			} else {
				break
			}
		}
		f.Close()
	}

	f, err := os.Create(propsPath)
	if err != nil {
		return fmt.Errorf("creando server.properties: %w", err)
	}
	defer f.Close()

	for _, c := range headerComments {
		fmt.Fprintln(f, c)
	}
	if len(headerComments) > 0 {
		fmt.Fprintln(f)
	}
	for _, p := range props {
		fmt.Fprintf(f, "%s=%s\n", p.Key, p.Value)
	}
	return nil
}

// ─── Configuración global ─────────────────────────────────────────────────────

func (a *App) GetSettings() config.Settings  { return config.LoadAllSettings() }
func (a *App) SaveSettings(s config.Settings) error { return config.SaveAllSettings(s) }

// ─── Actualización ───────────────────────────────────────────────────────────

func (a *App) CheckForUpdates() UpdateCheckResult {
	latest, available := utils.IsUpdateAvailable()
	return UpdateCheckResult{
		Available: available,
		Latest:    latest,
		Current:   config.CurrentVersion,
	}
}

func (a *App) GetCurrentVersion() string { return config.CurrentVersion }
func (a *App) GetDownloadURL() string    { return config.DownloadURL }

// ─── Java ─────────────────────────────────────────────────────────────────────

// GetInstalledJavas retorna todos los runtimes de Java instalados localmente.
func (a *App) GetInstalledJavas() []JavaRuntimeInfo {
	runtimes := javapkg.GetAvailableJavaRuntimes()
	result := make([]JavaRuntimeInfo, 0, len(runtimes))
	for _, r := range runtimes {
		result = append(result, JavaRuntimeInfo{
			Name:    r.Name,
			Version: r.Version,
			Path:    r.Path,
		})
	}
	return result
}

// GetJavaDownloadOptions retorna las versiones de Java disponibles para descargar,
// indicando cuáles ya están instaladas.
func (a *App) GetJavaDownloadOptions() []JavaDownloadOption {
	installed := javapkg.GetAvailableJavaRuntimes()
	installedVersions := make(map[int]bool, len(installed))
	for _, r := range installed {
		installedVersions[r.Version] = true
	}

	result := make([]JavaDownloadOption, 0, len(config.JavaMapping))
	for _, entry := range config.JavaMapping {
		result = append(result, JavaDownloadOption{
			Name:        entry.Name,
			JavaVersion: entry.JavaVersion,
			MCVersion:   entry.MCVersion,
			Installed:   installedVersions[entry.JavaVersion],
		})
	}
	return result
}

// DownloadJava descarga e instala una versión de Java.
// Emite eventos "java:download-progress" con JavaDownloadProgress.
func (a *App) DownloadJava(javaVersion int) {
	jInfo := findJavaEntry(javaVersion)
	if jInfo == nil {
		wailsrt.EventsEmit(a.ctx, "java:download-progress", JavaDownloadProgress{
			Error:       fmt.Sprintf("Versión de Java %d no encontrada en la configuración.", javaVersion),
			JavaVersion: javaVersion,
		})
		return
	}

	go func() {
		emit := func(p float64, text string, err string, success bool, path string) {
			wailsrt.EventsEmit(a.ctx, "java:download-progress", JavaDownloadProgress{
				Progress:    p,
				Text:        text,
				Error:       err,
				Success:     success,
				JavaVersion: javaVersion,
				JavaPath:    path,
			})
		}

		emit(0, fmt.Sprintf("Descargando %s...", jInfo.Name), "", false, "")

		jExe, installErr := javapkg.DownloadAndInstallJava(jInfo, func(p float64) {
			emit(p/100.0, fmt.Sprintf("Descargando %s... %.0f%%", jInfo.Name, p), "", false, "")
		})

		if installErr != nil {
			emit(0, "", fmt.Sprintf("Error al instalar %s: %v", jInfo.Name, installErr), false, "")
			return
		}

		emit(1.0, fmt.Sprintf("✅ %s instalado correctamente.", jInfo.Name), "", true, jExe)
	}()
}

// ─── Utilidades del sistema ───────────────────────────────────────────────────

func (a *App) OpenServerFolder() error {
	if a.activeServer == nil {
		return fmt.Errorf("no hay servidor activo")
	}
	return openFolder(a.activeServer.Path)
}

func (a *App) SelectFolder() (string, error) {
	return wailsrt.OpenDirectoryDialog(a.ctx, wailsrt.OpenDialogOptions{
		Title: "Seleccionar Carpeta de Servidores",
	})
}

func (a *App) FilterNewerVersions(all []string, current string) []string {
	return utils.FilterNewerVersions(all, current)
}

// openFolder abre el explorador de archivos en el path dado (multiplataforma).

// newServerManager centraliza la construcción del ServerManager con sus callbacks de eventos.
func (a *App) newServerManager(s *srvpkg.ServerRecord) *srvpkg.ServerManager {
	return srvpkg.NewServerManager(s,
		func(text string) { wailsrt.EventsEmit(a.ctx, "console:output", text) },
		func(running bool) {
			wailsrt.EventsEmit(a.ctx, "server:status", running)
			if !running && a.restartPending {
				a.restartPending = false
				go a.serverManager.Start()
			}
		},
		func() { wailsrt.EventsEmit(a.ctx, "server:graceful-fail", nil) },
	)
}

// waitForStop espera como máximo `timeout` a que el serverManager deje de correr.
func waitForStop(sm *srvpkg.ServerManager, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) && sm.IsRunning() {
		time.Sleep(50 * time.Millisecond)
	}
}

// mostFrequent retorna la clave con mayor valor en un map[string]int,
// o "N/A" si el mapa está vacío.
// Compara contra bestCount para evitar depender de m[best] cuando best no es
// una clave válida, y para obtener un resultado determinista (primer máximo
// encontrado en la iteración, estable para un mapa de un solo elemento).
func mostFrequent(m map[string]int) string {
	best := "N/A"
	bestCount := -1
	for k, n := range m {
		if n > bestCount {
			best = k
			bestCount = n
		}
	}
	return best
}

// findJavaEntry busca una entrada en JavaMapping por versión de Java.
func findJavaEntry(javaVersion int) *config.JavaEntry {
	for i := range config.JavaMapping {
		if config.JavaMapping[i].JavaVersion == javaVersion {
			entry := config.JavaMapping[i]
			return &entry
		}
	}
	return nil
}

// openFolder abre el explorador de archivos en el path dado (multiplataforma).
func openFolder(path string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
