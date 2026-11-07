// Command vibeasik adalah CLI vibeasik Community.
//
// Perintah server (serve, scan, ports, mcp) menyusul setelah registry modul selesai
// (lihat NEW_REPOS_PLAN T6); sementara tersedia perintah dasar.
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/vibeasik/vibeasik/internal/guidelines"
)

// version diisi saat rilis lewat -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("vibeasik %s (Community)\n", version)
	case "rules":
		if err := listRules(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "perintah tidak dikenal: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println(`vibeasik — lapisan standar untuk kode buatan AI

Pemakaian:
  vibeasik version   tampilkan versi
  vibeasik rules     daftar rule audit bawaan per kategori
  vibeasik help      bantuan ini`)
}

func listRules() error {
	cats, err := guidelines.LoadAllGuidelines()
	if err != nil {
		return err
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].FilePath < cats[j].FilePath })
	total := 0
	for _, c := range cats {
		fmt.Printf("%-40s %3d rule\n", c.Category, len(c.Guidelines))
		total += len(c.Guidelines)
	}
	fmt.Printf("\n%d kategori, %d rule\n", len(cats), total)
	return nil
}
