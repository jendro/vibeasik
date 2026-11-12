//go:build windows

// Command vibeasik-panel adalah jendela kendali vibeasik untuk Windows: Jalankan/Hentikan engine dan
// Buka Studio. Dibangun dengan -ldflags "-H windowsgui" agar tidak membuka jendela konsol.
package main

import (
	"os"

	"github.com/vibeasik/vibeasik/pkg/winpanel"
)

func main() { os.Exit(winpanel.Run()) }
