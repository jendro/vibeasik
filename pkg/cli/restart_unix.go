//go:build !windows

package cli

import (
	"os"
	"syscall"
)

// restartSelf mengganti proses ini dengan salinan baru yang membaca ulang konfigurasi.
// Soket Go berstatus close-on-exec sehingga port dilepas sebelum proses baru memakainya.
var restartSelf = func() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}
