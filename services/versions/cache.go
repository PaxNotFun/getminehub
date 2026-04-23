// Servicio de caché de versiones con TTL 24h en SQLite + memoria.
// Usa el pool compartido de config.GetDB().
package versions

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"getminehub/config"
)

const cacheTTL = 24 * 60 * 60 // 24 h en segundos

var ServerTypes = []string{"Vanilla", "PaperMC", "Folia", "Forge", "Fabric"}

var (
	memCache     = make(map[string][]string)
	memCacheLock sync.RWMutex
)

func loadFromDB(serverType string) []string {
	db := config.GetDB()
	if db == nil {
		return nil
	}

	now := int64(time.Now().Unix())
	var vj string
	var ft int64
	err := db.QueryRow("SELECT versions, fetched_at FROM version_cache WHERE server_type=?", serverType).Scan(&vj, &ft)
	if err != nil || (now-ft) >= cacheTTL {
		return nil
	}

	var v []string
	if json.Unmarshal([]byte(vj), &v) != nil {
		return nil
	}
	return v
}

func saveToDB(serverType string, versions []string) {
	db := config.GetDB()
	if db == nil {
		return
	}

	data, err := json.Marshal(versions)
	if err != nil {
		return
	}
	db.Exec(`INSERT OR REPLACE INTO version_cache(server_type,versions,fetched_at) VALUES(?,?,?)`,
		serverType, string(data), int64(time.Now().Unix()))
}

// GetVersions: 1. memoria → 2. SQLite → 3. API (bloquea)
func GetVersions(serverType string) []string {
	memCacheLock.RLock()
	if v, ok := memCache[serverType]; ok {
		memCacheLock.RUnlock()
		return v
	}
	memCacheLock.RUnlock()

	if v := loadFromDB(serverType); v != nil {
		memCacheLock.Lock()
		memCache[serverType] = v
		memCacheLock.Unlock()
		return v
	}

	versions, err := fetchFromAPI(serverType)
	if err != nil {
		slog.Warn("no se pudieron obtener versiones de la API", "type", serverType, "error", err)
		return nil
	}
	if len(versions) == 0 {
		return nil
	}
	saveToDB(serverType, versions)
	memCacheLock.Lock()
	memCache[serverType] = versions
	memCacheLock.Unlock()
	return versions
}

// GetVersionsCached sólo desde memoria, no bloquea. Retorna nil si no hay.
func GetVersionsCached(serverType string) []string {
	memCacheLock.RLock()
	defer memCacheLock.RUnlock()
	return memCache[serverType]
}

// PrefetchAllInBackground precarga todos los tipos en goroutines paralelas.
// La tabla version_cache ya existe gracias a database.InitDatabase() que se
// llama durante el arranque; no es necesario recrearla aquí.
func PrefetchAllInBackground(onTypeReady func(string, []string)) {
	for _, st := range ServerTypes {
		go func(stype string) {
			// Intentar primero desde SQLite (sin red)
			if v := loadFromDB(stype); v != nil {
				memCacheLock.Lock()
				memCache[stype] = v
				memCacheLock.Unlock()
				if onTypeReady != nil {
					onTypeReady(stype, v)
				}
				return
			}
			// Si no hay caché válida, intentar la API
			v, err := fetchFromAPI(stype)
			if err != nil || len(v) == 0 {
				slog.Warn("prefetch fallido (sin red o error)", "type", stype, "error", err)
				return
			}
			saveToDB(stype, v)
			memCacheLock.Lock()
			memCache[stype] = v
			memCacheLock.Unlock()
			if onTypeReady != nil {
				onTypeReady(stype, v)
			}
		}(st)
	}
}

// Invalidate fuerza refresco. serverType="" borra todo.
func Invalidate(serverType string) {
	memCacheLock.Lock()
	if serverType != "" {
		delete(memCache, serverType)
	} else {
		memCache = make(map[string][]string)
	}
	memCacheLock.Unlock()

	db := config.GetDB()
	if db == nil {
		return
	}
	if serverType != "" {
		db.Exec("DELETE FROM version_cache WHERE server_type=?", serverType)
	} else {
		db.Exec("DELETE FROM version_cache")
	}
}
