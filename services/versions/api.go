package versions

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"getminehub/config"
	httpclient "getminehub/services/http"
)

func fetchFromAPI(serverType string) ([]string, error) {
	switch serverType {
	case "Vanilla":
		return fetchVanilla()
	case "PaperMC":
		return fetchPaperMC()
	case "Folia":
		return fetchFolia()
	case "Fabric":
		return fetchFabric()
	case "Forge":
		return fetchForge()
	}
	return nil, fmt.Errorf("tipo desconocido: %s", serverType)
}

func fetchVanilla() ([]string, error) {
	body, err := httpclient.Get(config.VanillaAPIURL)
	if err != nil {
		return nil, fmt.Errorf("obteniendo versiones Vanilla: %w", err)
	}
	var data struct {
		Versions []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parseando versiones Vanilla: %w", err)
	}
	var out []string
	for _, v := range data.Versions {
		if v.Type == "release" {
			out = append(out, v.ID)
		}
	}
	return out, nil
}

func fetchPaperMC() ([]string, error) {
	body, err := httpclient.Get(config.PaperMCAPIURL)
	if err != nil {
		return nil, fmt.Errorf("obteniendo versiones PaperMC: %w", err)
	}
	var data struct {
		Versions []string `json:"versions"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parseando versiones PaperMC: %w", err)
	}
	v := data.Versions
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
	return v, nil
}

func fetchFolia() ([]string, error) {
	body, err := httpclient.Get(config.FoliaAPIURL)
	if err != nil {
		return nil, fmt.Errorf("obteniendo versiones Folia: %w", err)
	}
	var data struct {
		Versions []string `json:"versions"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parseando versiones Folia: %w", err)
	}
	v := data.Versions
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
	return v, nil
}

func fetchFabric() ([]string, error) {
	body, err := httpclient.Get(config.FabricMetaURL + "/versions/game")
	if err != nil {
		return nil, fmt.Errorf("obteniendo versiones Fabric: %w", err)
	}
	var data []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parseando versiones Fabric: %w", err)
	}
	var out []string
	for _, v := range data {
		if v.Stable {
			out = append(out, v.Version)
		}
	}
	return out, nil
}

func fetchForge() ([]string, error) {
	body, err := httpclient.Get(config.ForgeAPIURL)
	if err != nil {
		return nil, fmt.Errorf("obteniendo versiones Forge: %w", err)
	}
	var data struct {
		Promos map[string]string `json:"promos"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parseando versiones Forge: %w", err)
	}

	seen := map[string]bool{}
	var versions []string
	for key := range data.Promos {
		parts := strings.Split(key, "-")
		if len(parts) < 2 {
			continue
		}
		mcVer := strings.Join(parts[:len(parts)-1], "-")
		if !seen[mcVer] {
			seen[mcVer] = true
			versions = append(versions, mcVer)
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		return compareVersionTuples(versions[i], versions[j]) > 0
	})
	return versions, nil
}

func compareVersionTuples(a, b string) int {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	max := len(pa)
	if len(pb) > max {
		max = len(pb)
	}
	for i := 0; i < max; i++ {
		var ai, bi int
		if i < len(pa) {
			fmt.Sscan(pa[i], &ai)
		}
		if i < len(pb) {
			fmt.Sscan(pb[i], &bi)
		}
		if ai != bi {
			if ai > bi {
				return 1
			}
			return -1
		}
	}
	return 0
}

// IsVersionValid verifica si una versión es válida para un tipo de servidor.
func IsVersionValid(serverType, mcVersion string) bool {
	if mcVersion == "" {
		return false
	}
	versions := GetVersions(serverType)
	for _, v := range versions {
		if v == mcVersion {
			return true
		}
	}
	return false
}
