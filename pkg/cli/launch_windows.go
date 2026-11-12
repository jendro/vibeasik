//go:build windows

package cli

import (
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
