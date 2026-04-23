package java

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"getminehub/config"
	"getminehub/services/downloader"
	"getminehub/utils"
)

// javaExecName es el nombre del ejecutable de Java según la plataforma
func javaExecName() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}

// GetJavaInfoForMinecraft obtiene la información de Java requerida para una versión de Minecraft
func GetJavaInfoForMinecraft(mcVersion string) *config.JavaEntry {
	for _, req := range config.JavaMapping {
		if utils.IsVersionGreaterOrEqual(mcVersion, req.MCVersion) {
			entry := req
			return &entry
		}
	}
	return nil
}

// FindPrivateJavaExecutable busca una versión específica de Java en el directorio de runtimes
func FindPrivateJavaExecutable(requiredMajorVersion int) string {
	javaDir := config.JavaRuntimesDir
	if _, err := os.Stat(javaDir); os.IsNotExist(err) {
		return ""
	}

	entries, err := os.ReadDir(javaDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		javaExe := filepath.Join(javaDir, entry.Name(), "bin", javaExecName())
		if _, err := os.Stat(javaExe); os.IsNotExist(err) {
			continue
		}

		major := getJavaMajorVersion(javaExe)
		if major == requiredMajorVersion {
			return javaExe
		}
	}
	return ""
}

// getJavaMajorVersion ejecuta java -version y extrae la versión mayor.
// Usa un timeout para evitar bloqueos y captura stderr (donde Java escribe la versión).
func getJavaMajorVersion(javaExe string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, javaExe, "-version")
	// java -version escribe SOLO en stderr (estándar JVM)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// stdout se descarta
	cmd.Stdout = nil
	// Evitar que Windows abra una ventana de consola al detectar la versión de Java.
	utils.HideWindowAttr(cmd)

	if err := cmd.Run(); err != nil {
		// Si el contexto venció, loguear y retornar
		if ctx.Err() != nil {
			slog.Warn("timeout detectando versión de Java", "exe", javaExe)
		}
		return -1
	}

	return parseJavaMajorVersion(stderr.String())
}

// parseJavaMajorVersion extrae la versión mayor del output de "java -version".
// Formatos soportados:
//   - java version "21.0.4"   (OpenJDK/Oracle moderno)
//   - java version "1.8.0_422" (Java 8 legacy)
//   - openjdk version "17.0.12" (OpenJDK)
//   - java version "11.0.2"   (Java 11)
func parseJavaMajorVersion(output string) int {
	// Normalizar: coger la primera línea que contenga "version"
	var versionLine string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(strings.ToLower(line), "version") {
			versionLine = line
			break
		}
	}
	if versionLine == "" {
		return -1
	}

	// Buscar la versión entre comillas: version "X..." o version "1.X..."
	reQuoted := regexp.MustCompile(`version\s+"([^"]+)"`)
	m := reQuoted.FindStringSubmatch(versionLine)
	if len(m) < 2 {
		return -1
	}
	verStr := m[1]

	// Formato legacy: "1.8.0_422" → major = 8
	if strings.HasPrefix(verStr, "1.") {
		parts := strings.Split(verStr, ".")
		if len(parts) >= 2 {
			if v, err := strconv.Atoi(parts[1]); err == nil {
				return v
			}
		}
		return -1
	}

	// Formato moderno: "21.0.4", "17.0.12", "11.0.2", etc.
	firstDot := strings.Index(verStr, ".")
	if firstDot < 0 {
		firstDot = len(verStr)
	}
	major, err := strconv.Atoi(verStr[:firstDot])
	if err != nil {
		return -1
	}
	return major
}

// DownloadAndInstallJava descarga y extrae una versión de Java
func DownloadAndInstallJava(jInfo *config.JavaEntry, progress downloader.ProgressCallback) (string, error) {
	tmpDir, err := os.MkdirTemp("", "getminehub_java_*")
	if err != nil {
		return "", fmt.Errorf("no se pudo crear directorio temporal: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archiveName := filepath.Base(jInfo.URL)
	archivePath := filepath.Join(tmpDir, archiveName)

	if err := downloader.DownloadFileWithProgress(jInfo.URL, archivePath, progress); err != nil {
		return "", fmt.Errorf("fallo al descargar %s: %w", jInfo.Name, err)
	}

	if err := downloader.ExtractArchive(archivePath, config.JavaRuntimesDir); err != nil {
		return "", fmt.Errorf("fallo al extraer el archivo de Java: %w", err)
	}

	javaExe := FindPrivateJavaExecutable(jInfo.JavaVersion)
	if javaExe == "" {
		return "", fmt.Errorf("Java no encontrado después de la instalación (versión %d)", jInfo.JavaVersion)
	}

	if runtime.GOOS != "windows" {
		fixJavaBinPermissions(javaExe)
	}

	return javaExe, nil
}

// fixJavaBinPermissions da permisos de ejecución a todos los binarios de Java
func fixJavaBinPermissions(javaExe string) {
	binDir := filepath.Dir(javaExe)
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		path := filepath.Join(binDir, e.Name())
		if info, err := e.Info(); err == nil && !info.IsDir() {
			os.Chmod(path, 0755)
		}
	}
}

// RuntimeInfo contiene información sobre un runtime de Java instalado
type RuntimeInfo struct {
	Name    string
	Version int
	Path    string
}

// GetAvailableJavaRuntimes escanea el directorio de runtimes y retorna los Javas disponibles
func GetAvailableJavaRuntimes() []RuntimeInfo {
	var runtimes []RuntimeInfo

	javaDir := config.JavaRuntimesDir
	if _, err := os.Stat(javaDir); os.IsNotExist(err) {
		return runtimes
	}

	entries, err := os.ReadDir(javaDir)
	if err != nil {
		return runtimes
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		javaExe := filepath.Join(javaDir, entry.Name(), "bin", javaExecName())
		if _, err := os.Stat(javaExe); os.IsNotExist(err) {
			continue
		}

		major := getJavaMajorVersion(javaExe)
		if major > 0 {
			runtimes = append(runtimes, RuntimeInfo{
				Name:    fmt.Sprintf("Java %d", major),
				Version: major,
				Path:    javaExe,
			})
		}
	}

	return runtimes
}
