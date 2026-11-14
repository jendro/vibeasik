package routemap

import (
	"go/ast"
	"regexp"
	"strings"
)

// Aturan biaya: perhitungan di memori — termasuk loop bersarang yang memang harus menyentuh semua pasangan —
// dianggap wajar. Yang dilaporkan adalah I/O per item di dalam loop atas data: query database, panggilan HTTP
// keluar, akses berkas, kirim email, antrean/pubsub, dan goroutine baru per item. Semuanya bisa dibatch, diambil
// sekali di luar loop, atau dibatasi.

type ioKind uint8

const (
	ioQuery ioKind = 1 << iota
	ioHTTP
	ioFile
	ioMail
	ioQueue
	ioGoroutine
)

var ioKindNames = []struct {
	k    ioKind
	name string
}{{ioQuery, "query"}, {ioHTTP, "http"}, {ioFile, "file"}, {ioMail, "mail"}, {ioQueue, "queue"}, {ioGoroutine, "goroutine"}}

func (k ioKind) names() []string {
	var out []string
	for _, n := range ioKindNames {
		if k&n.k != 0 {
			out = append(out, n.name)
		}
	}
	return out
}

// IOLabels: nama jenis I/O untuk pengguna (Indonesia), urut seperti IOKinds.
func IOLabels(kinds []string) []string {
	label := map[string]string{"query": "query database", "http": "panggilan HTTP", "file": "akses berkas",
		"mail": "kirim email", "queue": "antrean/pubsub", "goroutine": "goroutine per item"}
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		if l, ok := label[k]; ok {
			out = append(out, l)
		}
	}
	return out
}

// OnlyQueries: I/O per item hanya query database (atau tidak diketahui) — teks lama "N+1 query" tetap tepat.
func OnlyQueries(kinds []string) bool {
	for _, k := range kinds {
		if k != "query" {
			return false
		}
	}
	return true
}

// Fungsi paket standar yang menyentuh berkas/jaringan/SMTP. Kunci "<paket>.<fungsi>". os.Stat/Lstat sengaja tidak
// dihitung, begitu juga os.Mkdir/MkdirAll (pada folder yang sudah ada = cek metadata): hanya metadata yang di-cache
// OS, jauh lebih murah dari membuka/membaca/menulis isi berkas atau query.
var ioPkgFuncs = map[string]ioKind{
	"os.Open": ioFile, "os.OpenFile": ioFile, "os.ReadFile": ioFile, "os.WriteFile": ioFile, "os.Create": ioFile, "os.Remove": ioFile, "os.RemoveAll": ioFile, "os.ReadDir": ioFile,
	"os.Rename": ioFile, "os.Chmod": ioFile, "os.CreateTemp": ioFile,
	"ioutil.ReadFile": ioFile, "ioutil.WriteFile": ioFile, "ioutil.ReadDir": ioFile,
	"filepath.Walk": ioFile, "filepath.WalkDir": ioFile, "filepath.Glob": ioFile,
	"http.Get": ioHTTP, "http.Post": ioHTTP, "http.Head": ioHTTP, "http.PostForm": ioHTTP,
	"smtp.SendMail": ioMail,
}

var (
	httpClientRootRe = regexp.MustCompile(`(?i)client|http|resty`)
	mailRootRe       = regexp.MustCompile(`(?i)mail|smtp|dialer`)
	queueRootRe      = regexp.MustCompile(`(?i)redis|rdb|nats|kafka|amqp|rabbit|producer|publisher|pubsub|channel`)
	queueMethods     = map[string]bool{"Publish": true, "PublishWithContext": true, "RPush": true, "LPush": true, "XAdd": true, "WriteMessages": true, "Produce": true}
)

// ioCallKind: jenis I/O sebuah pemanggilan non-lokal (0 = bukan I/O yang dikenali). Fungsi lokal proyek tidak
// sampai di sini; biayanya dihitung dari badannya sendiri.
func ioCallKind(sel *ast.SelectorExpr) ioKind {
	name := sel.Sel.Name
	if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Obj == nil {
		if k, ok := ioPkgFuncs[pkg.Name+"."+name]; ok {
			return k
		}
	}
	root := strings.ToLower(chainRoot(sel.X))
	switch {
	case name == "SaveFile":
		return ioFile // fiber/echo c.SaveFile(fh, path)
	case name == "Do" && httpClientRootRe.MatchString(root):
		return ioHTTP
	case (name == "Get" || name == "Post" || name == "Put" || name == "Delete" || name == "Head") && strings.Contains(root, "client"):
		return ioHTTP
	case (name == "DialAndSend" || name == "SendMail" || name == "Send") && mailRootRe.MatchString(root):
		return ioMail
	case queueMethods[name] && queueRootRe.MatchString(root):
		return ioQueue
	}
	return 0
}
