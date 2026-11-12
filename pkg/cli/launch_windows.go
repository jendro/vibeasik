//go:build windows

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

// launchedByDoubleClick bernilai true bila vibeasik.exe dibuka dari Explorer: Windows membuat
// konsol baru yang hanya berisi proses ini. Dari cmd/PowerShell konsolnya dipakai bersama shell.
func launchedByDoubleClick() bool {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleProcessList")
	if proc.Find() != nil {
		return false
	}
	var ids [2]uint32
	n, _, _ := proc.Call(uintptr(unsafe.Pointer(&ids[0])), uintptr(len(ids)))
	return n == 1
}

// startPanel membuka vibeasik-panel.exe (bila ada di folder yang sama) sebagai ganti konsol, untuk
// pengguna yang membuka vibeasik.exe dari Explorer. Mengembalikan false bila panel tidak tersedia.
func startPanel() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	panel := filepath.Join(filepath.Dir(exe), "vibeasik-panel.exe")
	if _, err := os.Stat(panel); err != nil {
		return false
	}
	return exec.Command(panel).Start() == nil
}
