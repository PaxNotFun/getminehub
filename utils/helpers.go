package utils

import (
	"net"
	"strconv"
	"strings"
	"time"
)

// CheckInternetConnection verifica si hay conexión a internet via TCP a 8.8.8.8:53.
func CheckInternetConnection() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// ParseVersion convierte un string de versión a una tupla de enteros.
// Ej: "1.20.4" → [1, 20, 4]
func ParseVersion(v string) []int {
	parts := strings.Split(v, ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			result[i] = 0
		} else {
			result[i] = n
		}
	}
	return result
}

// CompareVersions compara dos versiones semánticamente.
// Retorna >0 si v1>v2, 0 si iguales, <0 si v1<v2.
func CompareVersions(v1, v2 string) int {
	t1 := ParseVersion(v1)
	t2 := ParseVersion(v2)

	maxLen := len(t1)
	if len(t2) > maxLen {
		maxLen = len(t2)
	}

	for i := 0; i < maxLen; i++ {
		var a, b int
		if i < len(t1) {
			a = t1[i]
		}
		if i < len(t2) {
			b = t2[i]
		}
		if a != b {
			if a > b {
				return 1
			}
			return -1
		}
	}
	return 0
}

// IsVersionGreater retorna true si v1 es estrictamente mayor que v2.
func IsVersionGreater(v1, v2 string) bool {
	return CompareVersions(v1, v2) > 0
}

// IsVersionGreaterOrEqual retorna true si v1 >= v2.
func IsVersionGreaterOrEqual(v1, v2 string) bool {
	return CompareVersions(v1, v2) >= 0
}

// FilterNewerVersions filtra de `versions` solo las estrictamente mayores a `currentVersion`.
func FilterNewerVersions(versions []string, currentVersion string) []string {
	var newer []string
	for _, v := range versions {
		if IsVersionGreater(v, currentVersion) {
			newer = append(newer, v)
		}
	}
	return newer
}
