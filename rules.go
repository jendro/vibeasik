// Package vibeasik berisi aset yang disematkan ke binary (rule bawaan).
package vibeasik

import "embed"

// Guidelines adalah salinan bawaan folder guidelines/ (rule audit) yang ikut dikompilasi,
// sehingga binary tetap bisa memuat rule walau dijalankan di luar repo.
//
//go:embed guidelines/*.json
var Guidelines embed.FS
