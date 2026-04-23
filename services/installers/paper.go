package installers

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"getminehub/config"
	"getminehub/services/downloader"
)

type PaperInstaller struct{ info *ServerInstallInfo }
func NewPaperInstaller(info *ServerInstallInfo) *PaperInstaller { return &PaperInstaller{info: info} }
func (p *PaperInstaller) GetInfo() *ServerInstallInfo { return p.info }

func (p *PaperInstaller) InstallServer(progress downloader.ProgressCallback) error {
	buildsURL := fmt.Sprintf("%s/versions/%s/builds", config.PaperMCAPIURL, p.info.Version)
	body, err := httpGet(buildsURL)
	if err != nil { return fmt.Errorf("error obteniendo builds de PaperMC: %w", err) }

	var data struct {
		Builds []struct{ Build int `json:"build"` } `json:"builds"`
	}
	if err := json.Unmarshal(body, &data); err != nil || len(data.Builds) == 0 {
		return fmt.Errorf("no se encontraron builds para PaperMC %s", p.info.Version)
	}

	latestBuild := data.Builds[len(data.Builds)-1].Build
	jarName := fmt.Sprintf("paper-%s-%d.jar", p.info.Version, latestBuild)
	downloadURL := fmt.Sprintf("%s/versions/%s/builds/%d/downloads/%s",
		config.PaperMCAPIURL, p.info.Version, latestBuild, jarName)

	dest := filepath.Join(p.info.Path, "server.jar")
	if err := downloader.DownloadFileWithProgress(downloadURL, dest, progress); err != nil {
		return fmt.Errorf("la descarga del JAR de PaperMC falló: %w", err)
	}
	p.info.JarFile = "server.jar"
	return nil
}
