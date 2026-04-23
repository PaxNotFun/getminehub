package utils

import (
	"encoding/json"

	"getminehub/config"
	httpclient "getminehub/services/http"
)

type versionResponse struct {
	Version string `json:"version"`
}

// GetLatestVersion obtiene la última versión disponible desde el repositorio.
// Retorna "" si no se puede contactar el servidor o parsear la respuesta.
func GetLatestVersion() string {
	body, err := httpclient.Get(config.VersionJSONURL)
	if err != nil {
		return ""
	}
	var vr versionResponse
	if err := json.Unmarshal(body, &vr); err != nil {
		return ""
	}
	return vr.Version
}

// IsUpdateAvailable retorna (latestVersion, isAvailable).
// Respeta la configuración check_for_updates del usuario.
func IsUpdateAvailable() (string, bool) {
	s := config.LoadAllSettings()
	if !s.CheckForUpdates {
		return "", false
	}
	latest := GetLatestVersion()
	if latest == "" {
		return "", false
	}
	if IsVersionGreater(latest, config.CurrentVersion) {
		return latest, true
	}
	return latest, false
}
