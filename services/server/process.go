package server

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const maxHistoryLines = 2000

// ServerManager gestiona el proceso del servidor Minecraft.
// IsRunning y startNotifSent usan atomic.Bool para evitar race conditions
// al leerlos fuera del mutex principal (por ejemplo, desde app.go).
type ServerManager struct {
	ServerInfo *ServerRecord

	consoleMu       sync.Mutex
	ConsoleCallback func(text string)
	StatusCallback  func(running bool)
	KillReadyCB     func()

	mu          sync.Mutex
	isRunning   atomic.Bool // acceso atómico, sin mutex
	startNotif  atomic.Bool // acceso atómico
	process     *exec.Cmd
	stdinWriter io.WriteCloser

	// historyMu protege history y historyLines.
	// history es un strings.Builder: WriteString es O(1) amortizado y
	// GetHistory devuelve la cadena acumulada sin ningún Join extra.
	// historyLines lleva la cuenta de líneas para respetar maxHistoryLines;
	// cuando se supera el límite se descarta la primera mitad del buffer.
	historyMu    sync.Mutex
	history      strings.Builder
	historyLines int

	killTimer *time.Timer

	// jobHandle es un Win32 Job Object (solo Windows).
	// En Linux esto lo cubre Pdeathsig en SysProcAttr.
	jobHandle uintptr
}

// IsRunning devuelve si el servidor está corriendo.
// Seguro para llamar desde cualquier goroutine sin el mutex.
func (sm *ServerManager) IsRunning() bool {
	return sm.isRunning.Load()
}

// NewServerManager crea un nuevo ServerManager
func NewServerManager(info *ServerRecord, consoleCB func(string), statusCB func(bool), killReadyCB func()) *ServerManager {
	sm := &ServerManager{
		ServerInfo:      info,
		ConsoleCallback: consoleCB,
		StatusCallback:  statusCB,
		KillReadyCB:     killReadyCB,
	}
	welcome := fmt.Sprintf("━━━ Panel de Control: %s ━━━\n\n", info.Name)
	sm.history.WriteString(welcome)
	sm.historyLines = 2 // la línea del banner + la línea vacía
	// Inicializar Job Object (Windows) o no-op en Linux/macOS.
	initJobObject(sm)
	return sm
}

func (sm *ServerManager) LogOutput(text string) {
	sm.historyMu.Lock()
	sm.history.WriteString(text)
	// Contar las líneas nuevas que trae este fragmento.
	sm.historyLines += strings.Count(text, "\n")
	// Si superamos el límite, reconstruimos el builder conservando solo la
	// segunda mitad de las líneas. Esto ocurre raramente (cada ~2000 líneas)
	// y evita el crecimiento ilimitado del buffer.
	if sm.historyLines > maxHistoryLines {
		sm.trimHistory()
	}
	sm.historyMu.Unlock()

	sm.consoleMu.Lock()
	cb := sm.ConsoleCallback
	sm.consoleMu.Unlock()
	if cb != nil {
		cb(text)
	}
}

// trimHistory descarta la primera mitad de las líneas del historial.
// Debe llamarse con historyMu tomado.
func (sm *ServerManager) trimHistory() {
	full := sm.history.String()
	// Encontrar el inicio de la línea que está en la posición maxHistoryLines/2.
	keep := maxHistoryLines / 2
	pos := 0
	for i := 0; i < len(full); i++ {
		if full[i] == '\n' {
			pos++
			if pos >= keep {
				// Conservar desde el siguiente carácter.
				remaining := full[i+1:]
				sm.history.Reset()
				sm.history.WriteString(remaining)
				sm.historyLines = strings.Count(remaining, "\n")
				return
			}
		}
	}
	// Si no encontramos el punto de corte (buffer casi vacío), dejarlo tal cual.
}

func (sm *ServerManager) SetConsoleCallback(cb func(string)) {
	sm.consoleMu.Lock()
	defer sm.consoleMu.Unlock()
	sm.ConsoleCallback = cb
}

func (sm *ServerManager) GetHistory() string {
	sm.historyMu.Lock()
	defer sm.historyMu.Unlock()
	return sm.history.String()
}

func (sm *ServerManager) Start() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning.Load() {
		sm.LogOutput("El servidor ya está encendido.\n")
		return
	}

	if err := AcceptEULA(sm.ServerInfo.Path); err != nil {
		sm.LogOutput(fmt.Sprintf("❌ Error EULA: %v\n", err))
		return
	}
	sm.LogOutput("EULA Aceptado.\n")

	minRAM, maxRAM, customArgs, useAikar := LoadServerConfig(sm.ServerInfo.Path)
	command := sm.buildStartCommand(minRAM, maxRAM, customArgs, useAikar)
	if len(command) == 0 {
		sm.LogOutput("❌ Error: No se pudo construir el comando de inicio.\n")
		return
	}

	sm.LogOutput(fmt.Sprintf("🚀 Ejecutando: %s\n\n", strings.Join(command, " ")))

	cmd, stdin, stdout, err := buildCmd(command, sm.ServerInfo.Path)
	if err != nil {
		sm.LogOutput(fmt.Sprintf("❌ %v\n", err))
		return
	}

	if cmd.Process != nil {
		assignJobObject(sm, cmd.Process.Pid)
	}

	sm.process = cmd
	sm.stdinWriter = stdin
	sm.isRunning.Store(true)
	sm.StatusCallback(true)

	go sm.readOutput(stdout)
}

// buildCmd construye el *exec.Cmd con stdin/stdout/stderr configurados y lo arranca.
// stdout y stderr se fusionan en un único pipe (io.Pipe) para que la consola
// muestre ambas salidas en orden de llegada, sin perder mensajes de error de la JVM.
func buildCmd(command []string, dir string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	setProcAttr(cmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creando stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, nil, nil, fmt.Errorf("creando stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, nil, nil, fmt.Errorf("creando stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, nil, nil, fmt.Errorf("iniciando proceso: %w", err)
	}

	// Fusionar stdout + stderr en un único ReadCloser para que el caller
	// reciba ambas salidas en un solo scanner, igual que antes.
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		io.Copy(pw, stdout) //nolint:errcheck
	}()
	go func() {
		io.Copy(pw, stderr) //nolint:errcheck
	}()

	return cmd, stdin, pr, nil
}

func (sm *ServerManager) readOutput(stdout io.ReadCloser) {
	defer stdout.Close()

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		sm.LogOutput(scanner.Text() + "\n")
	}

	if sm.process != nil {
		if err := sm.process.Wait(); err != nil {
			slog.Debug("proceso del servidor terminó", "error", err)
		}
	}
	sm.LogOutput("\n--- Servidor detenido ---\n")

	sm.mu.Lock()
	sm.isRunning.Store(false)
	sm.startNotif.Store(false)
	if sm.killTimer != nil {
		sm.killTimer.Stop()
		sm.killTimer = nil
	}
	if sm.stdinWriter != nil {
		sm.stdinWriter.Close()
		sm.stdinWriter = nil
	}
	sm.mu.Unlock()

	sm.StatusCallback(false)
}

func (sm *ServerManager) SendCommand(command string) {
	sm.mu.Lock()
	writer := sm.stdinWriter
	running := sm.isRunning.Load()
	sm.mu.Unlock()

	if !running || writer == nil {
		return
	}

	sm.LogOutput(fmt.Sprintf("> %s\n", command))
	writer.Write([]byte(command + "\n"))
}

func (sm *ServerManager) Stop() {
	if !sm.isRunning.Load() {
		return
	}

	sm.SendCommand("stop")

	sm.mu.Lock()
	sm.killTimer = time.AfterFunc(15*time.Second, func() {
		if sm.isRunning.Load() && sm.KillReadyCB != nil {
			sm.KillReadyCB()
		}
	})
	sm.mu.Unlock()
}

func (sm *ServerManager) Kill() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.process != nil && sm.process.Process != nil {
		killProcess(sm.process)
	}
	sm.isRunning.Store(false)
}

// ─── Comandos de construcción ─────────────────────────────────────────────────

func (sm *ServerManager) buildStartCommand(minRAM, maxRAM, customArgs string, useAikar bool) []string {
	info := sm.ServerInfo
	forgeLaunchType := info.ForgeLaunchType
	forgeArgsFile := info.ForgeArgsFile

	javaExe := info.JavaExecutable
	if javaExe == "" {
		javaExe = "java"
	}

	if info.Type == "Forge" && forgeLaunchType == "" {
		forgeLaunchType, forgeArgsFile = sm.detectForgeTypeAtRuntime()
	}

	if info.Type == "Forge" && forgeLaunchType == "modern" {
		return sm.buildModernForgeCommand(javaExe, minRAM, maxRAM, customArgs, forgeArgsFile, useAikar)
	}

	return sm.buildStandardJarCommand(javaExe, minRAM, maxRAM, customArgs, useAikar)
}

func (sm *ServerManager) buildModernForgeCommand(javaExe, minRAM, maxRAM, customArgs, forgeArgsFile string, useAikar bool) []string {
	if forgeArgsFile == "" {
		sm.LogOutput("❌ Error: Forge moderno detectado pero falta forge_args_file.\n")
		return nil
	}

	argsFilePath := filepath.Join(sm.ServerInfo.Path, forgeArgsFile)
	if _, err := os.Stat(argsFilePath); os.IsNotExist(err) {
		sm.LogOutput(fmt.Sprintf("❌ No se encuentra el archivo de argumentos: %s\n", forgeArgsFile))
		return nil
	}

	jvmArgsPath := filepath.Join(sm.ServerInfo.Path, "user_jvm_args.txt")
	var lines strings.Builder
	lines.WriteString(fmt.Sprintf("-Xms%s\n-Xmx%s\n", minRAM, maxRAM))
	if useAikar {
		for _, f := range getAikarFlags() {
			lines.WriteString(f + "\n")
		}
	}
	if customArgs != "" {
		for _, arg := range strings.Fields(customArgs) {
			lines.WriteString(arg + "\n")
		}
	}

	if err := os.WriteFile(jvmArgsPath, []byte(lines.String()), 0644); err != nil {
		sm.LogOutput(fmt.Sprintf("❌ Error al crear user_jvm_args.txt: %v\n", err))
		return nil
	}

	return []string{javaExe, "@user_jvm_args.txt", "@" + forgeArgsFile, "nogui"}
}

func (sm *ServerManager) buildStandardJarCommand(javaExe, minRAM, maxRAM, customArgs string, useAikar bool) []string {
	jarFile := sm.ServerInfo.JarFile
	if jarFile == "" {
		jarFile = sm.findServerJar()
		if jarFile == "" {
			sm.LogOutput("❌ Error: No se encontró ningún archivo JAR en el directorio.\n")
			return nil
		}
	}

	jarPath := filepath.Join(sm.ServerInfo.Path, jarFile)
	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		sm.LogOutput(fmt.Sprintf("❌ Error: No se encuentra el JAR: %s\n", jarFile))
		return nil
	}

	cmd := []string{javaExe, fmt.Sprintf("-Xms%s", minRAM), fmt.Sprintf("-Xmx%s", maxRAM)}
	if useAikar {
		cmd = append(cmd, getAikarFlags()...)
	}
	if customArgs != "" {
		cmd = append(cmd, strings.Fields(customArgs)...)
	}
	cmd = append(cmd, "-jar", jarFile, "nogui")
	return cmd
}

func (sm *ServerManager) detectForgeTypeAtRuntime() (string, string) {
	argsFilename := "unix_args.txt"
	if runtime.GOOS == "windows" {
		argsFilename = "win_args.txt"
	}

	forgePath := filepath.Join(sm.ServerInfo.Path, "libraries", "net", "minecraftforge", "forge")
	if _, err := os.Stat(forgePath); os.IsNotExist(err) {
		return "legacy", ""
	}

	found := ""
	filepath.WalkDir(forgePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == argsFilename {
			found = path
			return fs.SkipAll
		}
		return nil
	})

	if found != "" {
		if rel, err := filepath.Rel(sm.ServerInfo.Path, found); err == nil {
			return "modern", filepath.ToSlash(rel)
		}
	}
	return "legacy", ""
}

func (sm *ServerManager) findServerJar() string {
	serverPath := sm.ServerInfo.Path

	if _, err := os.Stat(filepath.Join(serverPath, "server.jar")); err == nil {
		return "server.jar"
	}

	if sm.ServerInfo.Type == "Forge" && sm.ServerInfo.Version != "" {
		pattern := regexp.MustCompile(fmt.Sprintf(`(?i)forge-.*%s-.*\.jar$`,
			regexp.QuoteMeta(sm.ServerInfo.Version)))
		entries, _ := os.ReadDir(serverPath)
		for _, e := range entries {
			if pattern.MatchString(e.Name()) {
				return e.Name()
			}
		}
	}

	entries, _ := os.ReadDir(serverPath)
	for _, e := range entries {
		n := e.Name()
		if strings.HasSuffix(n, ".jar") && n != "installer.jar" && n != "forge-installer.jar" {
			return n
		}
	}
	return ""
}

// getAikarFlags — Fuente: https://aikar.co/mcflags.html
func getAikarFlags() []string {
	return []string{
		"-XX:+UseG1GC", "-XX:+ParallelRefProcEnabled", "-XX:MaxGCPauseMillis=200",
		"-XX:+UnlockExperimentalVMOptions", "-XX:+DisableExplicitGC", "-XX:+AlwaysPreTouch",
		"-XX:G1NewSizePercent=30", "-XX:G1MaxNewSizePercent=40", "-XX:G1HeapRegionSize=8M",
		"-XX:G1ReservePercent=20", "-XX:G1HeapWastePercent=5", "-XX:G1MixedGCCountTarget=4",
		"-XX:InitiatingHeapOccupancyPercent=15", "-XX:G1MixedGCLiveThresholdPercent=90",
		"-XX:G1RSetUpdatingPauseTimePercent=5", "-XX:SurvivorRatio=32",
		"-XX:+PerfDisableSharedMem", "-XX:MaxTenuringThreshold=1",
		"-Dusing.aikars.flags=https://mcflags.emc.gs", "-Daikars.new.flags=true",
	}
}
