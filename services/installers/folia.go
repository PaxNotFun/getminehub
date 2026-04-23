package installers

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"getminehub/config"
	"getminehub/services/downloader"
)

type FoliaInstaller struct{ info *ServerInstallInfo }
func NewFoliaInstaller(info *ServerInstallInfo) *FoliaInstaller { return &FoliaInstaller{info: info} }
func (f *FoliaInstaller) GetInfo() *ServerInstallInfo { return f.info }

func (f *FoliaInstaller) InstallServer(progress downloader.ProgressCallback) error {
	buildsURL := fmt.Sprintf("%s/versions/%s/builds", config.FoliaAPIURL, f.info.Version)
	body, err := httpGet(buildsURL)
	if err != nil { return fmt.Errorf("error obteniendo builds de Folia: %w", err) }

	var data struct {
		Builds []struct{ Build int `json:"build"` } `json:"builds"`
	}
	if err := json.Unmarshal(body, &data); err != nil || len(data.Builds) == 0 {
		return fmt.Errorf("no se encontraron builds para Folia %s", f.info.Version)
	}

	latestBuild := data.Builds[len(data.Builds)-1].Build
	jarName := fmt.Sprintf("folia-%s-%d.jar", f.info.Version, latestBuild)
	downloadURL := fmt.Sprintf("%s/versions/%s/builds/%d/downloads/%s",
		config.FoliaAPIURL, f.info.Version, latestBuild, jarName)

	dest := filepath.Join(f.info.Path, "server.jar")
	if err := downloader.DownloadFileWithProgress(downloadURL, dest, progress); err != nil {
		return fmt.Errorf("la descarga del JAR de Folia falló: %w", err)
	}
	f.info.JarFile = "server.jar"
	return nil
}
