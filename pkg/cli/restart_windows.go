//go:build windows

package cli

// restartSelf tidak didukung di Windows; Studio meminta pengguna menjalankan ulang engine.
var restartSelf func() error // nil: tidak didukung
