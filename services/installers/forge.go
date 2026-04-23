package installers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"getminehub/config"
	"getminehub/services/downloader"
	"getminehub/utils"
)

type ForgeInstaller struct{ info *ServerInstallInfo }
func NewForgeInstaller(info *ServerInstallInfo) *ForgeInstaller { return &ForgeInstaller{info: info} }
func (fi *ForgeInstaller) GetInfo() *ServerInstallInfo { return fi.info }

func (fi *ForgeInstaller) InstallServer(progress downloader.ProgressCallback) error {
	// Obtener versión de Forge
	body, err := httpGet(config.ForgeAPIURL)
	if err != nil { return fmt.Errorf("error obteniendo versiones de Forge: %w", err) }

	var data struct{ Promos map[string]string `json:"promos"` }
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("error parseando datos de Forge: %w", err)
	}

	mcVer := fi.info.Version
	forgeVerPart := data.Promos[mcVer+"-recommended"]
	if forgeVerPart == "" { forgeVerPart = data.Promos[mcVer+"-latest"] }
	if forgeVerPart == "" {
		return fmt.Errorf("no se encontró versión de Forge para MC %s", mcVer)
	}

	fullForgeVer := mcVer + "-" + forgeVerPart
	installerURL := fmt.Sprintf("%s%s/forge-%s-installer.jar", config.ForgeMavenURL, fullForgeVer, fullForgeVer)
	instPath := filepath.Join(fi.info.Path, "installer.jar")

	if err := downloader.DownloadFileWithProgress(installerURL, instPath, progress); err != nil {
		return fmt.Errorf("descarga del instalador de Forge falló: %w", err)
	}

	// Ejecutar instalador
	javaExe := fi.info.JavaExecutable
	if javaExe == "" { javaExe = "java" }

	cmd := exec.Command(javaExe, "-jar", instPath, "--installServer")
	cmd.Dir = fi.info.Path
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = hiddenWindowAttr()
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("instalador de Forge falló:\n%s", string(out))
	}

	os.Remove(instPath)

	// Detectar tipo de lanzamiento
	launchType := fi.detectForgeLaunchType(mcVer)

	if launchType == "modern" {
		return fi.setupModernForge(forgeVerPart)
	}
	return fi.setupLegacyForge(mcVer)
}

func (fi *ForgeInstaller) detectForgeLaunchType(mcVersion string) string {
	// MC >= 1.17 siempre es moderno
	if utils.IsVersionGreaterOrEqual(mcVersion, "1.17.0") {
		return "modern"
	}

	// Buscar archivo de argumentos
	argsFilename := "unix_args.txt"
	_ = argsFilename
	if runtime.GOOS == "windows" { argsFilename = "win_args.txt" }

	forgePath := filepath.Join(fi.info.Path, "libraries", "net", "minecraftforge", "forge")
	if _, err := os.Stat(forgePath); err == nil {
		found := false
		filepath.Walk(forgePath, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && info.Name() == argsFilename {
				found = true
				return fmt.Errorf("stop")
			}
			return nil
		})
		if found { return "modern" }
	}

	// run.sh/run.bat indica moderno
	runScript := filepath.Join(fi.info.Path, "run.sh")
	if runtime.GOOS == "windows" { runScript = filepath.Join(fi.info.Path, "run.bat") }
	if _, err := os.Stat(runScript); err == nil { return "modern" }

	return "legacy"
}

func (fi *ForgeInstaller) setupModernForge(forgeVersion string) error {
	argsFilename := "unix_args.txt"
	if runtime.GOOS == "windows" { argsFilename = "win_args.txt" }

	forgePath := filepath.Join(fi.info.Path, "libraries", "net", "minecraftforge", "forge")
	var argsFilePath string

	if _, err := os.Stat(forgePath); err == nil {
		var found []string
		filepath.Walk(forgePath, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && info.Name() == argsFilename {
				found = append(found, path)
			}
			return nil
		})
		// Priorizar el que contiene la versión exacta
		for _, f := range found {
			if strings.Contains(f, forgeVersion) {
				argsFilePath = f
				break
			}
		}
		if argsFilePath == "" && len(found) > 0 {
			argsFilePath = found[0]
		}
	}

	if argsFilePath == "" {
		return fmt.Errorf("Forge moderno detectado pero no se encontró el archivo de argumentos")
	}

	rel, err := filepath.Rel(fi.info.Path, argsFilePath)
	if err != nil { return fmt.Errorf("error calculando ruta relativa: %w", err) }

	fi.info.ForgeLaunchType = "modern"
	fi.info.ForgeArgsFile = filepath.ToSlash(rel)
	fi.info.JarFile = ""
	return nil
}

func (fi *ForgeInstaller) setupLegacyForge(mcVersion string) error {
	pattern := regexp.MustCompile(fmt.Sprintf(`(?i)forge-.*%s-.*\.jar$`, regexp.QuoteMeta(mcVersion)))

	var serverJar string
	entries, _ := os.ReadDir(fi.info.Path)
	for _, e := range entries {
		if pattern.MatchString(e.Name()) {
			if strings.Contains(strings.ToLower(e.Name()), "universal") {
				serverJar = e.Name()
				break
			}
			if serverJar == "" { serverJar = e.Name() }
		}
	}

	if serverJar == "" {
		return fmt.Errorf("Forge legacy detectado pero no se encontró el JAR del servidor")
	}

	// Renombrar a server.jar
	src := filepath.Join(fi.info.Path, serverJar)
	dst := filepath.Join(fi.info.Path, "server.jar")
	if src != dst {
		if _, err := os.Stat(dst); err == nil { os.Remove(dst) }
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("error renombrando JAR de Forge: %w", err)
		}
	}

	fi.info.ForgeLaunchType = "legacy"
	fi.info.JarFile = "server.jar"
	fi.info.ForgeArgsFile = ""
	return nil
}
