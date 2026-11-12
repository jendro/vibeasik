//go:build !windows

package cli

// launchedByDoubleClick hanya berarti di Windows; di OS lain vibeasik dijalankan dari terminal.
func launchedByDoubleClick() bool { return false }

// startPanel hanya ada di Windows.
func startPanel() bool { return false }
