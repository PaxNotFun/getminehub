//go:build windows

package server

import (
	"os/exec"
	"syscall"
	"unsafe"
)

const (
	createNoWindow             = 0x08000000
	jobObjectLimitKillOnClose  = 0x00002000 // JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	jobObjectExtendedLimitInfo = 9          // JobObjectExtendedLimitInformation
	processAllAccess           = 0x1F0FFF
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObject = kernel32.NewProc("CreateJobObjectW")
	procSetInfoJob      = kernel32.NewProc("SetInformationJobObject")
	procAssignProcToJob = kernel32.NewProc("AssignProcessToJobObject")
	procOpenProcess     = kernel32.NewProc("OpenProcess")
	procCloseHandle     = kernel32.NewProc("CloseHandle")
)

// Estructuras Win32 para Job Objects
type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type basicLimitInfo struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type extendedLimitInfo struct {
	BasicLimitInformation basicLimitInfo
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// setProcAttr configura el proceso para que no muestre ventana.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow,
		HideWindow:    true,
	}
}

// killProcess mata el proceso Java.
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}

// initJobObject crea un Win32 Job Object con JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE.
// Esto garantiza que cuando la app Go se cierre (incluso de forma inesperada),
// el sistema operativo matará automáticamente todos los procesos del Job Object,
// es decir el servidor Java no quedará como proceso huérfano.
func initJobObject(sm *ServerManager) {
	handle, _, _ := procCreateJobObject.Call(0, 0)
	if handle == 0 {
		return
	}

	info := extendedLimitInfo{}
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnClose

	ret, _, _ := procSetInfoJob.Call(
		handle,
		jobObjectExtendedLimitInfo,
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if ret == 0 {
		procCloseHandle.Call(handle)
		return
	}

	sm.jobHandle = handle
}

// assignJobObject asigna el proceso creado al Job Object del ServerManager.
func assignJobObject(sm *ServerManager, pid int) {
	if sm.jobHandle == 0 {
		return
	}
	pHandle, _, _ := procOpenProcess.Call(processAllAccess, 0, uintptr(pid))
	if pHandle == 0 {
		return
	}
	procAssignProcToJob.Call(sm.jobHandle, pHandle)
	// No cerramos pHandle — debe mantenerse vivo mientras el proceso exista.
	// Windows lo limpiará cuando el proceso termine.
}
