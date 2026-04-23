package installers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"getminehub/config"
	"getminehub/services/downloader"
)

type FabricInstaller struct{ info *ServerInstallInfo }

func NewFabricInstaller(info *ServerInstallInfo) *FabricInstaller { return &FabricInstaller{info: info} }
func (f *FabricInstaller) GetInfo() *ServerInstallInfo            { return f.info }

func (f *FabricInstaller) InstallServer(progress downloader.ProgressCallback) error {
	latestInstaller, latestLoader, err := f.resolveFabricVersions()
	if err != nil {
		return err
	}

	installerPath := filepath.Join(f.info.Path, "fabric-installer.jar")
	installerURL := fmt.Sprintf("%s%s/fabric-installer-%s.jar",
		config.FabricMavenURL, latestInstaller, latestInstaller)

	if err := downloader.DownloadFileWithProgress(installerURL, installerPath, func(p float64) {
		progress(p * 0.5)
	}); err != nil {
		return fmt.Errorf("descarga del instalador de Fabric falló: %w", err)
	}

	if err := f.runFabricInstaller(installerPath, latestLoader); err != nil {
		return err
	}

	launcherPath := filepath.Join(f.info.Path, "fabric-server-launch.jar")
	if _, err := os.Stat(launcherPath); os.IsNotExist(err) {
		return fmt.Errorf("no se encontró 'fabric-server-launch.jar' después de la instalación")
	}

	f.info.JarFile = "fabric-server-launch.jar"
	return nil
}

// resolveFabricVersions obtiene la versión más reciente del instalador y del loader
// para la versión de MC solicitada.
func (f *FabricInstaller) resolveFabricVersions() (installerVer, loaderVer string, err error) {
	instBody, err := httpGet(config.FabricMetaURL + "/versions/installer")
	if err != nil {
		return "", "", fmt.Errorf("obteniendo versiones del instalador Fabric: %w", err)
	}
	var installerVersions []struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(instBody, &installerVersions); err != nil || len(installerVersions) == 0 {
		return "", "", fmt.Errorf("no se encontraron versiones del instalador Fabric")
	}

	loaderBody, err := httpGet(fmt.Sprintf("%s/versions/loader/%s", config.FabricMetaURL, f.info.Version))
	if err != nil {
		return "", "", fmt.Errorf("obteniendo loader de Fabric: %w", err)
	}
	var loaderVersions []struct {
		Loader struct {
			Version string `json:"version"`
		} `json:"loader"`
	}
	if err := json.Unmarshal(loaderBody, &loaderVersions); err != nil || len(loaderVersions) == 0 {
		return "", "", fmt.Errorf("no se encontró loader para Fabric %s", f.info.Version)
	}

	return installerVersions[0].Version, loaderVersions[0].Loader.Version, nil
}

// runFabricInstaller ejecuta el JAR del instalador Fabric y lo elimina al terminar.
func (f *FabricInstaller) runFabricInstaller(installerPath, loaderVersion string) error {
	javaExe := f.info.JavaExecutable
	if javaExe == "" {
		javaExe = "java"
	}

	cmd := exec.Command(javaExe, "-jar", installerPath,
		"server", "-mcversion", f.info.Version,
		"-loader", loaderVersion, "-downloadMinecraft")
	cmd.Dir = f.info.Path
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = hiddenWindowAttr()
	}

	out, err := cmd.CombinedOutput()
	os.Remove(installerPath) // limpiar independientemente del resultado
	if err != nil {
		return fmt.Errorf("el instalador de Fabric falló:\n%s", string(out))
	}
	return nil
}
