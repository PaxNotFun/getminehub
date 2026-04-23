package installers

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"getminehub/config"
	"getminehub/services/downloader"
)

type VanillaInstaller struct{ info *ServerInstallInfo }

func NewVanillaInstaller(info *ServerInstallInfo) *VanillaInstaller {
	return &VanillaInstaller{info: info}
}
func (v *VanillaInstaller) GetInfo() *ServerInstallInfo { return v.info }

func (v *VanillaInstaller) InstallServer(progress downloader.ProgressCallback) error {
	// Obtener manifest de versiones de Mojang
	body, err := httpGet(config.VanillaAPIURL)
	if err != nil { return fmt.Errorf("error obteniendo manifest: %w", err) }

	var manifest struct {
		Versions []struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		return fmt.Errorf("error parseando manifest: %w", err)
	}

	var versionURL string
	for _, ver := range manifest.Versions {
		if ver.ID == v.info.Version {
			versionURL = ver.URL
			break
		}
	}
	if versionURL == "" {
		return fmt.Errorf("versión Vanilla '%s' no encontrada en el manifest de Mojang", v.info.Version)
	}

	// Obtener detalles de la versión
	vBody, err := httpGet(versionURL)
	if err != nil { return fmt.Errorf("error obteniendo datos de versión: %w", err) }

	var vData struct {
		Downloads struct {
			Server struct {
				URL string `json:"url"`
			} `json:"server"`
		} `json:"downloads"`
	}
	if err := json.Unmarshal(vBody, &vData); err != nil {
		return fmt.Errorf("error parseando datos de versión: %w", err)
	}

	downloadURL := vData.Downloads.Server.URL
	if downloadURL == "" {
		return fmt.Errorf("no hay JAR de servidor disponible para Vanilla '%s'", v.info.Version)
	}

	dest := filepath.Join(v.info.Path, "server.jar")
	if err := downloader.DownloadFileWithProgress(downloadURL, dest, progress); err != nil {
		return fmt.Errorf("la descarga del JAR de Vanilla falló: %w", err)
	}

	v.info.JarFile = "server.jar"
	return nil
}
