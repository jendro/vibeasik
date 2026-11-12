//go:build windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
)

// restartTarget biner yang akan dijalankan ulang. Setelah pembaruan, biner lama dipindah ke .old dan
// os.Executable masih menunjuk jalur aslinya, yang kini berisi biner baru.
func restartTarget() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(exe); err != nil || st.IsDir() {
		return "", fmt.Errorf("biner %s tidak ditemukan untuk mulai ulang", exe)
	}
	return exe, nil
}

// restartCheck memastikan mulai ulang akan berhasil, sebelum engine dihentikan.
var restartCheck = func() error { _, err := restartTarget(); return err }

// restartSelf menjalankan salinan baru di konsol yang sama lalu menunggunya selesai. Windows tidak
// punya exec, jadi proses lama tetap ada (diam) supaya terminal tidak kembali ke prompt dan Ctrl+C
// tetap menghentikan engine; kode keluar proses baru diteruskan.
var restartSelf = func() error {
	exe, err := restartTarget()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), restartedEnv+"=1")
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		return err
	}
	return nil
}
