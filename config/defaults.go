package config

import "runtime"

// Valores por defecto
const DefaultTimeout = 15
const DefaultMinRAM = "1G"
const DefaultMaxRAM = "2G"
const DefaultJVMArgs = ""

// JavaEntry contiene la información de una versión de Java requerida
type JavaEntry struct {
	MCVersion   string
	JavaVersion int
	URL         string
	Name        string
}

// javaMappingAll tiene las URLs para todas las plataformas
var javaMappingAll = []struct {
	MCVersion   string
	JavaVersion int
	URLWindows  string
	URLLinux    string
	URLMacOS    string
	Name        string
}{
	{
		MCVersion:   "1.20.5",
		JavaVersion: 21,
		URLWindows:  "https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.4%2B7/OpenJDK21U-jdk_x64_windows_hotspot_21.0.4_7.zip",
		URLLinux:    "https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.4%2B7/OpenJDK21U-jdk_x64_linux_hotspot_21.0.4_7.tar.gz",
		URLMacOS:    "https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.4%2B7/OpenJDK21U-jdk_x64_mac_hotspot_21.0.4_7.tar.gz",
		Name:        "Temurin JDK 21",
	},
	{
		MCVersion:   "1.17.1",
		JavaVersion: 17,
		URLWindows:  "https://github.com/adoptium/temurin17-binaries/releases/download/jdk-17.0.12%2B7/OpenJDK17U-jdk_x64_windows_hotspot_17.0.12_7.zip",
		URLLinux:    "https://github.com/adoptium/temurin17-binaries/releases/download/jdk-17.0.12%2B7/OpenJDK17U-jdk_x64_linux_hotspot_17.0.12_7.tar.gz",
		URLMacOS:    "https://github.com/adoptium/temurin17-binaries/releases/download/jdk-17.0.12%2B7/OpenJDK17U-jdk_x64_mac_hotspot_17.0.12_7.tar.gz",
		Name:        "Temurin JDK 17",
	},
	{
		MCVersion:   "1.17.0",
		JavaVersion: 16,
		URLWindows:  "https://github.com/adoptium/temurin16-binaries/releases/download/jdk-16.0.2%2B7/OpenJDK16U-jdk_x64_windows_hotspot_16.0.2_7.zip",
		URLLinux:    "https://github.com/adoptium/temurin16-binaries/releases/download/jdk-16.0.2%2B7/OpenJDK16U-jdk_x64_linux_hotspot_16.0.2_7.tar.gz",
		URLMacOS:    "https://github.com/adoptium/temurin16-binaries/releases/download/jdk-16.0.2%2B7/OpenJDK16U-jdk_x64_mac_hotspot_16.0.2_7.tar.gz",
		Name:        "Temurin JDK 16",
	},
	{
		MCVersion:   "0.0.0",
		JavaVersion: 8,
		URLWindows:  "https://github.com/adoptium/temurin8-binaries/releases/download/jdk8u422-b05/OpenJDK8U-jdk_x64_windows_hotspot_8u422b05.zip",
		URLLinux:    "https://github.com/adoptium/temurin8-binaries/releases/download/jdk8u422-b05/OpenJDK8U-jdk_x64_linux_hotspot_8u422b05.tar.gz",
		URLMacOS:    "https://github.com/adoptium/temurin8-binaries/releases/download/jdk8u422-b05/OpenJDK8U-jdk_x64_mac_hotspot_8u422b05.tar.gz",
		Name:        "Temurin JDK 8",
	},
}

// JavaMapping contiene el mapeo de versiones para la plataforma actual
var JavaMapping []JavaEntry

func init() {
	for _, e := range javaMappingAll {
		url := e.URLLinux
		if runtime.GOOS == "windows" {
			url = e.URLWindows
		} else if runtime.GOOS == "darwin" {
			url = e.URLMacOS
		}
		JavaMapping = append(JavaMapping, JavaEntry{
			MCVersion:   e.MCVersion,
			JavaVersion: e.JavaVersion,
			URL:         url,
			Name:        e.Name,
		})
	}
}
