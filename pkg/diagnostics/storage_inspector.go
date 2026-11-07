package diagnostics

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Lokasi penyimpanan berkas lokal (tanpa object storage seperti S3). Aplikasi hasil build (Go, Rust, Java,
// Node hasil bundle) dijalankan service manager dari working directory apa pun dan tanpa source tree, jadi
// folder simpan/muat berkas wajib tetap: dari env (mis. STORAGE_DIR) dengan default absolut di luar folder
// rilis, dibuat saat start. Path relatif ikut CWD, folder temp dihapus saat reboot, path dari lokasi source
// (runtime.Caller) atau path laptop developer tidak ada di server.
const (
	storageQuote = "[\"'`]"
	// literal path relatif ke folder penyimpanan atau berkas SQLite: "uploads/...", "./storage", "app.db"
	storageRelLiteral = storageQuote + `(?:\./)?(?:(?:uploads?|storage|media|attachments?|files|documents?|avatars?|userdata|data)(?:[/\\][^"'` + "`" + `]*)?|[^"'` + "`" + `/\\:]+\.(?:db|sqlite3?))` + storageQuote
)

var (
	storageRelativePathRegex = regexp.MustCompile(`(?i)(?:` +
		// argumen pertama pemanggilan I/O berkas / static server / koneksi SQLite
		`\b(?:MkdirAll|Mkdir|Create|WriteFile|OpenFile|Open|Join|writeFileSync|writeFile|appendFile|createWriteStream|mkdirSync|static|makedirs|Dir|connect)\s*\(\s*(?:process\.cwd\(\)\s*,\s*|os\.getcwd\(\)\s*,\s*)?` +
		// argumen kedua: SaveFile(file, "uploads/..."), Static("/uploads", "./uploads"), move_uploaded_file($tmp, "uploads/...")
		`|\b(?:SaveFile|SaveUploadedFile|Static|move_uploaded_file|save|rename)\s*\(\s*[^,()]+,\s*` +
		// nilai konfigurasi: uploadDir := "./uploads", UPLOAD_FOLDER = 'uploads', dest: 'uploads/'
		`|\b(?:dest|destination|[a-z_]*(?:upload|storage|media|attachment|file)s?_?(?:dir|path|folder|root|location))\s*(?::=|=|:)\s*` +
		`)` + storageRelLiteral)
	storageTempDirRegex     = regexp.MustCompile(`(?i)(?:\bos\.TempDir\(\)|\bos\.tmpdir\(\)|\btempfile\.gettempdir\(\)|\bsys_get_temp_dir\(\)|` + storageQuote + `/tmp/)`)
	storageKeywordRegex     = regexp.MustCompile(`(?i)(upload|storage|media|attachment|avatar|document)`)
	storageMachinePathRegex = regexp.MustCompile(storageQuote + `(?:/home/|/Users/|[A-Za-z]:[\\/]{1,2}Users[\\/]{1,2})([^/\\"'` + "`" + `]+)[/\\]`)
	// nama user placeholder di contoh/dokumentasi, bukan path mesin sungguhan
	storagePlaceholderUsers = map[string]bool{"user": true, "username": true, "you": true, "me": true, "example": true, "nama": true, "xxx": true, "<user>": true, "$user": true, "${user}": true}
	storageSourceDirRegex   = regexp.MustCompile(`runtime\.Caller\(\s*0\s*\)[^\n]*\n(?:[^\n]*\n){0,3}?[^\n]*filepath\.Dir\(`)
	// Pemakaian object storage di kode (bukan sekadar dependensi): bila ada, aturan folder lokal dilewati
	objectStorageUsageRegex = regexp.MustCompile(`(?i)(PutObject|putObject|put_object|upload_fileobj|s3\.New|NewFromConfig|S3Client|minio\.New|storage\.NewClient|\.storage\.from\(|cloudinary|Storage::disk\(\s*['"]s3|@vercel/blob|BlobServiceClient|azblob\.)`)
)

const storageFix = "// Folder penyimpanan tetap: dari env, default absolut di sebelah binary (bukan CWD), dibuat saat start.\n" +
	"func storageDir() string {\n    if d := os.Getenv(\"STORAGE_DIR\"); d != \"\" {\n        return d // server: mis. /var/lib/nama-app/storage, di luar folder rilis\n    }\n" +
	"    exe, _ := os.Executable()\n    return filepath.Join(filepath.Dir(exe), \"storage\")\n}\n\n" +
	"// main(): if err := os.MkdirAll(storageDir(), 0o755); err != nil { log.Fatal(err) }\n" +
	"// simpan/muat: filepath.Join(storageDir(), \"uploads\", namaAman)"

// inspectStoragePaths menjalankan aturan STORAGE-* pada kode server (komentar dilewati).
func inspectStoragePaths(content, relPath, lang string, lines []string) []models.Finding {
	var findings []models.Finding
	add := func(rule, title, desc, suggestion string, lineNum int, snippet string) {
		findings = append(findings, models.Finding{
			Category:     "deployment",
			Severity:     "WARNING",
			RuleID:       rule,
			Title:        title,
			Description:  desc,
			Suggestion:   suggestion,
			TargetFile:   relPath,
			Line:         lineNum,
			Snippet:      snippet,
			TheoryBasis:  "The Twelve-Factor App (Factor III: Config, Factor VI: Processes — backing storage di luar folder rilis) & Filesystem Hierarchy Standard (/var/lib untuk data aplikasi).",
			SuggestedFix: storageFix,
		})
	}
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		lineNum := idx + 1
		switch {
		case storageRelativePathRegex.MatchString(trimmed):
			add("STORAGE-RELATIVE-PATH", "Lokasi Berkas Relatif terhadap Working Directory",
				fmt.Sprintf("Path penyimpanan relatif pada baris %d diselesaikan terhadap working directory proses, bukan folder aplikasi. Binary yang dijalankan systemd/supervisor/docker dari folder lain akan menyimpan ke tempat berbeda atau gagal memuat berkas yang sudah ada.", lineNum),
				"Ambil folder dari env (mis. STORAGE_DIR) dengan default absolut di sebelah binary (os.Executable), buat dengan MkdirAll saat start, dan daftarkan kuncinya di .env.example.",
				lineNum, trimmed)
		case storageTempDirRegex.MatchString(trimmed) && storageKeywordRegex.MatchString(trimmed):
			add("STORAGE-TEMP-DIR", "Berkas Persisten Disimpan di Folder Temp",
				fmt.Sprintf("Baris %d menyimpan berkas unggahan/penyimpanan di folder temp sistem. Isi /tmp dibersihkan saat reboot (dan terisolasi per service bila systemd PrivateTmp aktif), sehingga berkas hilang.", lineNum),
				"Simpan berkas permanen di folder penyimpanan tetap (STORAGE_DIR); pakai temp hanya untuk berkas sementara yang langsung dipindah atau dihapus.",
				lineNum, trimmed)
		}
		if m := storageMachinePathRegex.FindStringSubmatch(trimmed); m != nil && !storagePlaceholderUsers[strings.ToLower(m[1])] {
			add("STORAGE-MACHINE-PATH", "Path Absolut Mesin Developer Tertanam di Kode",
				fmt.Sprintf("Baris %d memuat path home/user mesin developer. Path ini tidak ada di server, jadi simpan/muat berkas gagal setelah aplikasi diunggah.", lineNum),
				"Ganti dengan nilai dari env atau path yang diturunkan dari lokasi binary.",
				lineNum, trimmed)
		}
	}
	if lang == "go" {
		if loc := storageSourceDirRegex.FindStringIndex(content); loc != nil {
			lineNum := countLinesUntil(content, loc[0])
			add("STORAGE-SOURCE-RELATIVE", "Folder Berkas Diturunkan dari Lokasi Source (runtime.Caller)",
				fmt.Sprintf("Baris %d menurunkan folder dari path berkas source saat kompilasi. Binary di server tidak membawa source tree, jadi path menunjuk folder mesin build.", lineNum),
				"Turunkan folder dari os.Executable() atau env, atau tanam berkas statis dengan //go:embed.",
				lineNum, strings.TrimSpace(lines[lineNum-1]))
		}
	}
	return findings
}

// dropStorageWhenObjectStore membuang temuan folder lokal (relatif/temp) bila salah satu berkas server
// memakai object storage (S3/MinIO/GCS/...), karena berkas tidak disimpan di disk. Berkas hanya dibaca
// ulang bila memang ada temuan tersebut.
func dropStorageWhenObjectStore(findings []models.Finding, files []string) []models.Finding {
	isLocal := func(f models.Finding) bool {
		return f.RuleID == "STORAGE-RELATIVE-PATH" || f.RuleID == "STORAGE-TEMP-DIR"
	}
	has := false
	for _, f := range findings {
		if isLocal(f) {
			has = true
			break
		}
	}
	if !has {
		return findings
	}
	usesObjectStore := false
	for _, p := range files {
		if data, err := os.ReadFile(p); err == nil && objectStorageUsageRegex.Match(data) {
			usesObjectStore = true
			break
		}
	}
	if !usesObjectStore {
		return findings
	}
	out := findings[:0]
	for _, f := range findings {
		if !isLocal(f) {
			out = append(out, f)
		}
	}
	return out
}
