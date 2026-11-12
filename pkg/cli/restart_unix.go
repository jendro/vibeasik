//go:build !windows

package cli

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/vibeasik/vibeasik/pkg/config"
)

// restartTarget biner yang akan dijalankan ulang. Setelah pembaruan biner diganti di tempat, sehingga
// Linux melaporkan jalur lama berakhiran " (deleted)": potong agar menunjuk biner baru.
func restartTarget() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe = strings.TrimSuffix(exe, " (deleted)")
	if st, err := os.Stat(exe); err != nil || st.IsDir() {
		return "", fmt.Errorf("biner %s tidak ditemukan untuk mulai ulang", exe)
	}
	return exe, nil
}

// restartCheck memastikan mulai ulang akan berhasil, sebelum engine dihentikan.
var restartCheck = func() error { _, err := restartTarget(); return err }

// restartSelf mengganti proses ini dengan salinan baru yang membaca ulang konfigurasi.
// Soket Go berstatus close-on-exec sehingga port dilepas sebelum proses baru memakainya.
var restartSelf = func() error {
	exe, err := restartTarget()
	if err != nil {
		return err
	}
	return syscall.Exec(exe, os.Args, append(config.ChildEnviron(), restartedEnv+"=1"))
}
