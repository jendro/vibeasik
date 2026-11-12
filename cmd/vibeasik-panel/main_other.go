//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "vibeasik-panel hanya untuk Windows. Di Linux/macOS jalankan: vibeasik serve -o")
	os.Exit(1)
}
