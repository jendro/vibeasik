package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

// inferCorpus menyusun path uji dari semua kata kunci yang dikenali heuristik rute
// (tunggal, berpasangan, dan kosong) agar tiap cabang terkunci.
func inferCorpus() []string {
	keys := []string{"auth", "login", "register", "token", "logout", "password", "oauth",
		"user", "member", "profile", "role", "permission", "account", "project", "bundle",
		"component", "app", "asset", "product", "item", "inventory", "category", "diag",
		"health", "ping", "metric", "log", "port", "ports", "monitor", "llm", "ai", "scan",
		"fast-scan", "review", "advisor", "config", "model", "security", "censor", "secret",
		"vault", "setting", "settings/llm", "preference", "file", "upload", "download", "media",
		"document", "dir", "kontingen", "official", "daftar", "ikuti", "tagihan", "rekap",
		"atlet", "pertandingan", "kejuaraan", "bagan", "bracket", "kelas", "kategori", "bayar",
		"pembayaran", "rekening", "invoice", "billing", "public", "portal", "gallery", "galeri",
		"blog", "news", "lokasi", "admin", "verify", "verifikasi", "dashboard", "mcp",
		"diagnose", "ai-review", "routes/sync", "routes/enrich", "censored-keywords", ":id",
		"[id]", "zzz", "Login", "AUTH", ""}
	out := []string{"", "/", "/api", "/api/", "/api/zzz"}
	for _, a := range keys {
		out = append(out, "/"+a, "/api/"+a+"/list")
		for _, b := range keys {
			out = append(out, "/api/"+a+"/"+b)
		}
	}
	return out
}

var inferMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "PAGE", "page", "OPTIONS", "", "get"}

func goldenHash(f func(path, method string) string) string {
	h := sha256.New()
	var sb strings.Builder
	for _, p := range inferCorpus() {
		for _, m := range inferMethods {
			sb.Reset()
			fmt.Fprintf(&sb, "%s|%s=>%s\n", m, p, f(p, m))
			h.Write([]byte(sb.String()))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestInferRouteDomainAndFunctionGolden(t *testing.T) {
	got := goldenHash(func(p, m string) string {
		d, f := InferRouteDomainAndFunction(p, m)
		return d + "/" + f
	})
	if got != "b774ba783430a965b917ab8446b45bc07f826b344f2c3f43cf02e1c6891549ec" {
		t.Fatalf("hash domain/function berubah: %s", got)
	}
}

func TestInferRouteDescriptionGolden(t *testing.T) {
	got := goldenHash(func(p, m string) string { return InferRouteDescription(m, p) })
	if got != "e066d1e9281ad4d06c40437048aa1999c3b043ab5893c8fb438253858567b426" {
		t.Fatalf("hash deskripsi berubah: %s", got)
	}
}
