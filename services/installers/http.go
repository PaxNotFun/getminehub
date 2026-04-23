package installers

import (
	"fmt"

	httpclient "getminehub/services/http"
)

// httpGet es un alias interno que delega al cliente HTTP centralizado.
// Mantiene compatibilidad con las llamadas existentes dentro del paquete
// sin duplicar la lógica de timeout, User-Agent ni manejo de errores.
func httpGet(url string) ([]byte, error) {
	body, err := httpclient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return body, nil
}

// GetInstaller retorna el instalador correcto según el tipo de servidor.
func GetInstaller(info *ServerInstallInfo) (Installer, error) {
	switch info.Type {
	case "Vanilla":
		return NewVanillaInstaller(info), nil
	case "PaperMC":
		return NewPaperInstaller(info), nil
	case "Folia":
		return NewFoliaInstaller(info), nil
	case "Forge":
		return NewForgeInstaller(info), nil
	case "Fabric":
		return NewFabricInstaller(info), nil
	}
	return nil, fmt.Errorf("tipo de servidor '%s' no soportado", info.Type)
}
