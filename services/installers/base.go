// Instalador base: orquesta Java + descarga + server.properties + BD
package installers

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"getminehub/config"
	"getminehub/services/downloader"
	httpclient "getminehub/services/http"
	"getminehub/services/java"
	"getminehub/services/server"
	"getminehub/utils"
)

// ProgressStatus es el estado que se envía al callback de progreso de instalación.
type ProgressStatus struct {
	Text          string
	Progress      float64 // 0.0 – 1.0
	Error         string
	Success       bool
	ServerData    *server.ServerRecord
	WasReinstall  bool
	WasUpdate     bool
	ReinstallMode string
	Path          string
}

// ProgressCallback recibe actualizaciones de progreso.
type ProgressCallback func(ProgressStatus)

// ServerInstallInfo contiene los datos del servidor a instalar.
type ServerInstallInfo struct {
	Name            string
	Type            string
	Version         string
	Path            string
	JavaExecutable  string
	JarFile         string
	ForgeArgsFile   string
	ForgeLaunchType string
	ReinstallMode   string
	OriginalPath    string
}

// Installer define la interfaz de un instalador específico de tipo de servidor.
type Installer interface {
	InstallServer(progress downloader.ProgressCallback) error
	GetInfo() *ServerInstallInfo
}

// updater es un helper interno que empaqueta el callback de progreso con su contexto.
// Evita repetir los campos WasReinstall/WasUpdate/ReinstallMode en cada llamada.
type updater struct {
	cb           ProgressCallback
	isReinstall  bool
	isUpdate     bool
	reinstallMode string
	basePath     string
}

func newUpdater(cb ProgressCallback, info *ServerInstallInfo, basePath string, isReinstall, isUpdate bool) *updater {
	return &updater{
		cb:            cb,
		isReinstall:   isReinstall,
		isUpdate:      isUpdate,
		reinstallMode: info.ReinstallMode,
		basePath:      basePath,
	}
}

func (u *updater) send(text string, progress float64, errMsg string, success bool, srv *server.ServerRecord) {
	if u.cb == nil {
		return
	}
	u.cb(ProgressStatus{
		Text:          text,
		Progress:      progress,
		Error:         errMsg,
		Success:       success,
		ServerData:    srv,
		WasReinstall:  u.isReinstall && !u.isUpdate,
		WasUpdate:     u.isUpdate,
		ReinstallMode: u.reinstallMode,
		Path:          u.basePath,
	})
}

func (u *updater) fail(tempBackup, msg string) {
	if tempBackup == "" {
		u.send("", 0, msg, false, nil)
		return
	}
	u.send("Error detectado, restaurando archivos originales...", 0.99, "", false, nil)
	if err := server.RestoreFromTempReinstall(u.basePath, tempBackup); err != nil {
		u.send("", 0, msg+"\n\n⚠️ No se pudieron restaurar completamente los archivos originales.", false, nil)
	} else {
		u.send("", 0, msg+"\n\n✅ Rollback completado: tus archivos originales han sido restaurados.", false, nil)
	}
}

// ─── RunInstallation ──────────────────────────────────────────────────────────

// RunInstallation ejecuta el proceso completo de instalación/reinstalación/actualización.
// Está dividido en tres fases privadas para mantener cada función bajo 50 líneas:
//  1. preparePhase  — backup/limpieza del directorio existente
//  2. javaPhase     — resolver y/o descargar el JDK correcto
//  3. persistPhase  — registrar el servidor en la base de datos
func RunInstallation(inst Installer, basePath string, progressCB ProgressCallback, isReinstall, isUpdate bool) {
	info := inst.GetInfo()
	upd := newUpdater(progressCB, info, basePath, isReinstall, isUpdate)

	if !checkInternetConn() {
		upd.send("", 0, "No se puede continuar sin conexión a internet.", false, nil)
		return
	}

	tempBackup, ok := preparePhase(upd, basePath, info, isReinstall, isUpdate)
	if !ok {
		return
	}

	jExe, ok := javaPhase(upd, info, tempBackup)
	if !ok {
		return
	}
	info.JavaExecutable = jExe

	// Descargar/instalar el servidor (0.50 → 0.90)
	action := "Descargando"
	if isUpdate {
		action = "Actualizando"
	}
	upd.send(fmt.Sprintf("%s %s %s...", action, info.Type, info.Version), 0.5, "", false, nil)
	if err := inst.InstallServer(func(p float64) {
		upd.send("", 0.5+(p/100)*0.4, "", false, nil)
	}); err != nil {
		upd.fail(tempBackup, fmt.Sprintf("Error instalando %s: %v", info.Type, err))
		return
	}

	upd.send("Configurando archivos finales...", 0.98, "", false, nil)

	if !(isReinstall || isUpdate) || info.ReinstallMode == "total" {
		downloadServerProperties(basePath)
	}

	srvData := persistPhase(upd, info, basePath, jExe, tempBackup, isReinstall, isUpdate)
	upd.send("¡Completado!", 1.0, "", true, srvData)
}

// preparePhase realiza el backup/limpieza antes de instalar.
// Retorna (tempBackupPath, ok). ok=false indica que se emitió el error y hay que abortar.
func preparePhase(upd *updater, basePath string, info *ServerInstallInfo, isReinstall, isUpdate bool) (string, bool) {
	if !isReinstall && !isUpdate {
		return "", true
	}
	verb := "actualización"
	if !isUpdate {
		verb = "reinstalación"
	}
	upd.send(fmt.Sprintf("Preparando %s...", verb), 0.0, "", false, nil)

	tp, msg, err := server.PrepareReinstall(basePath, info.ReinstallMode)
	if err != nil {
		upd.send("", 0, fmt.Sprintf("Error preparando %s: %v", verb, err), false, nil)
		return "", false
	}
	upd.send(msg, 0.02, "", false, nil)
	return tp, true
}

// javaPhase resuelve el JDK necesario y lo descarga si no está instalado.
// Retorna (javaExecutablePath, ok). ok=false indica que se emitió el error y hay que abortar.
func javaPhase(upd *updater, info *ServerInstallInfo, tempBackup string) (string, bool) {
	upd.send("Verificando Java...", 0.05, "", false, nil)

	jInfo := java.GetJavaInfoForMinecraft(info.Version)
	if jInfo == nil {
		upd.fail(tempBackup, fmt.Sprintf("No se encontró Java compatible para MC %s", info.Version))
		return "", false
	}

	jExe := java.FindPrivateJavaExecutable(jInfo.JavaVersion)
	if jExe != "" {
		upd.send("Java listo.", 0.45, "", false, nil)
		return jExe, true
	}

	upd.send(fmt.Sprintf("Descargando %s...", jInfo.Name), 0.10, "", false, nil)
	var err error
	jExe, err = java.DownloadAndInstallJava(jInfo, func(p float64) {
		upd.send("", 0.10+(p/100)*0.30, "", false, nil)
	})
	if err != nil {
		upd.fail(tempBackup, fmt.Sprintf("Java no se pudo instalar: %v", err))
		return "", false
	}

	upd.send("Java listo.", 0.45, "", false, nil)
	return jExe, true
}

// persistPhase registra o actualiza el servidor en la base de datos.
// Retorna el ServerRecord resultante (puede ser nil si hay error de BD).
func persistPhase(upd *updater, info *ServerInstallInfo, basePath, jExe, tempBackup string, isReinstall, isUpdate bool) *server.ServerRecord {
	if isReinstall || isUpdate {
		return persistUpdate(upd, info, basePath, jExe, tempBackup)
	}
	return persistNew(upd, info, basePath, jExe)
}

func persistUpdate(upd *updater, info *ServerInstallInfo, basePath, jExe, tempBackup string) *server.ServerRecord {
	existing, _ := server.GetServerByPath(basePath)
	if existing != nil {
		if err := server.UpdateServer(existing.UUID, map[string]interface{}{
			"java_executable":   jExe,
			"type":              info.Type,
			"version":           info.Version,
			"jar_file":          info.JarFile,
			"forge_args_file":   info.ForgeArgsFile,
			"forge_launch_type": info.ForgeLaunchType,
		}); err != nil {
			slog.Warn("error actualizando servidor en BD", "path", basePath, "error", err)
		}
	} else {
		rec := serverRecordFrom(info, basePath, jExe)
		if err := server.AddServer(rec); err != nil {
			slog.Warn("error agregando servidor en BD durante update", "path", basePath, "error", err)
		}
	}
	srv, _ := server.GetServerByPath(basePath)
	server.CleanupTempReinstall(tempBackup)
	return srv
}

func persistNew(upd *updater, info *ServerInstallInfo, basePath, jExe string) *server.ServerRecord {
	rec := serverRecordFrom(info, basePath, jExe)
	if err := server.AddServer(rec); err != nil {
		upd.fail("", fmt.Sprintf("Error al guardar en base de datos: %v", err))
		return nil
	}
	srv, _ := server.GetServerByPath(basePath)
	if srv != nil {
		if err := server.UpdateServerConfig(srv.UUID, "2G", "4G", "", false); err != nil {
			slog.Warn("error aplicando config RAM por defecto", "uuid", srv.UUID, "error", err)
		}
		srv, _ = server.GetServerByPath(basePath)
	}
	return srv
}

// serverRecordFrom construye un ServerRecord a partir de ServerInstallInfo.
func serverRecordFrom(info *ServerInstallInfo, basePath, jExe string) *server.ServerRecord {
	return &server.ServerRecord{
		Name:            info.Name,
		Path:            basePath,
		Type:            info.Type,
		Version:         info.Version,
		JavaExecutable:  jExe,
		JarFile:         info.JarFile,
		ForgeArgsFile:   info.ForgeArgsFile,
		ForgeLaunchType: info.ForgeLaunchType,
	}
}

// ─── Helpers de red ───────────────────────────────────────────────────────────

// downloadServerProperties descarga el server.properties por defecto.
// Si falla (red caída), escribe un archivo vacío — Minecraft lo regenera en el
// primer arranque.
func downloadServerProperties(serverPath string) {
	body, err := httpclient.Get(config.ServerPropertiesURL)
	if err != nil {
		body = []byte{}
	}
	if err := os.WriteFile(filepath.Join(serverPath, "server.properties"), body, 0644); err != nil {
		slog.Warn("error escribiendo server.properties", "path", serverPath, "error", err)
	}
}

// checkInternetConn verifica conectividad delegando en utils.CheckInternetConnection.
func checkInternetConn() bool {
	return utils.CheckInternetConnection()
}
