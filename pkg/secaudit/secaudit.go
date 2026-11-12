package secaudit

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"
)

// SourceDir = satu folder komponen backend yang dipindai.
type SourceDir struct {
	Name string
	Path string
}

// RouteRef = rute yang dikenal vibeasik (hasil sinkron rute komponen).
type RouteRef struct {
	Method string
	Path   string
}

// Input = sasaran satu uji keamanan.
type Input struct {
	ProjectID   uint
	ComponentID uint
	Target      string
	Root        string      // dasar path relatif di laporan
	Dirs        []SourceDir // folder sumber backend; kosong = uji live saja
	Routes      []RouteRef
	Live        *LiveOptions // nil = tanpa probe live
}

var (
	paramSegRe     = regexp.MustCompile(`(:[A-Za-z0-9_]+\??|\{[^}]+\}|\[[^\]]+\])`)
	authPathRe     = regexp.MustCompile(`(?i)(login|sign-?in|authenticate|/token|/otp)`)
	unsafeLivePath = regexp.MustCompile(`(?i)(logout|delete|remove|destroy|reset|purge|drop|webhook|callback|stream|sse|ws\b|socket|download|export)`)
)

// Run menjalankan audit statis atas sumber dan (bila diminta) probe live, lalu memberi skor.
func Run(ctx context.Context, in Input) *Report {
	start := time.Now()
	rep := &Report{ProjectID: in.ProjectID, ComponentID: in.ComponentID, Target: in.Target, StartedAt: start, RoutesSeen: len(in.Routes)}

	if len(in.Dirs) > 0 {
		var facts staticFacts
		for _, d := range in.Dirs {
			base := in.Root
			if base == "" {
				base = d.Path
			}
			merge(&facts, scanSource(d.Path, base))
		}
		rep.FilesRead = facts.files
		if rep.RoutesSeen == 0 {
			rep.RoutesSeen = facts.routes
		}
		rep.Checks = append(rep.Checks, staticChecks(facts)...)
	} else {
		rep.Notes = append(rep.Notes, "Tidak ada folder backend; hanya probe live yang dijalankan.")
	}

	if in.Live != nil && strings.TrimSpace(in.Live.BaseURL) != "" {
		opts := *in.Live
		if len(opts.Paths) == 0 {
			opts.Paths = LivePaths(in.Routes)
		}
		if len(opts.AuthPaths) == 0 {
			opts.AuthPaths = AuthPaths(in.Routes)
		}
		live, checks := probeLive(ctx, opts)
		rep.Live = live
		rep.Checks = append(rep.Checks, checks...)
	} else {
		rep.Notes = append(rep.Notes, "Probe live tidak dijalankan: kontrol berstatus \"declared\" baru terlihat di kode, belum terbukti di respons HTTP.")
	}

	rep.Score, rep.Grade = score(rep.Checks)
	for _, c := range rep.Checks {
		switch c.Status {
		case StatusPass:
			rep.Summary.Pass++
		case StatusDeclared:
			rep.Summary.Declared++
		case StatusWarn:
			rep.Summary.Warn++
		case StatusFail:
			rep.Summary.Fail++
		default:
			rep.Summary.Skipped++
		}
	}
	sort.SliceStable(rep.Checks, func(i, j int) bool { return rank(rep.Checks[i]) < rank(rep.Checks[j]) })
	rep.DurationMs = time.Since(start).Milliseconds()
	return rep
}

func merge(dst *staticFacts, src staticFacts) {
	dst.files += src.files
	dst.serverFiles += src.serverFiles
	dst.routes += src.routes
	dst.mutations += src.mutations
	dst.sqlParam += src.sqlParam
	cat := func(a, b []Location) []Location {
		for _, l := range b {
			if len(a) >= maxLocations {
				break
			}
			a = append(a, l)
		}
		return a
	}
	dst.authRoutes = cat(dst.authRoutes, src.authRoutes)
	dst.rateLimit = cat(dst.rateLimit, src.rateLimit)
	dst.headers = cat(dst.headers, src.headers)
	dst.hsts = cat(dst.hsts, src.hsts)
	dst.csp = cat(dst.csp, src.csp)
	dst.cors = cat(dst.cors, src.cors)
	dst.corsStar = cat(dst.corsStar, src.corsStar)
	dst.corsCreds = cat(dst.corsCreds, src.corsCreds)
	dst.sqlConcat = cat(dst.sqlConcat, src.sqlConcat)
	dst.cmdInject = cat(dst.cmdInject, src.cmdInject)
	dst.validator = cat(dst.validator, src.validator)
	dst.bodyLimit = cat(dst.bodyLimit, src.bodyLimit)
	dst.authMw = cat(dst.authMw, src.authMw)
	dst.errorLeak = cat(dst.errorLeak, src.errorLeak)
	dst.debugMode = cat(dst.debugMode, src.debugMode)
}

// LivePaths memilih paling banyak 3 rute GET yang aman diuji (tanpa efek samping), health/root lebih dulu.
func LivePaths(routes []RouteRef) []string {
	var pick []string
	seen := map[string]bool{}
	add := func(p string) {
		if !seen[p] && len(pick) < 3 {
			seen[p] = true
			pick = append(pick, p)
		}
	}
	for _, r := range routes {
		m := strings.ToUpper(r.Method)
		if (m == "GET" || m == "PAGE") && (strings.Contains(r.Path, "health") || r.Path == "/") {
			add(concrete(r.Path))
		}
	}
	for _, r := range routes {
		m := strings.ToUpper(r.Method)
		if m != "GET" || unsafeLivePath.MatchString(r.Path) || strings.Contains(r.Path, "*") {
			continue
		}
		add(concrete(r.Path))
	}
	if len(pick) == 0 {
		pick = []string{"/"}
	}
	return pick
}

// AuthPaths memilih rute POST login/token untuk uji brute-force.
func AuthPaths(routes []RouteRef) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range routes {
		if strings.ToUpper(r.Method) == "POST" && authPathRe.MatchString(r.Path) && !seen[r.Path] {
			seen[r.Path] = true
			out = append(out, concrete(r.Path))
		}
	}
	return out
}

func concrete(path string) string {
	if path == "" {
		return "/"
	}
	return paramSegRe.ReplaceAllString(path, "1")
}

// score: 100 dikurangi bobot tiap kontrol yang gagal/lemah; kontrol "declared" tanpa bukti live sedikit mengurangi.
func score(checks []Check) (int, string) {
	s := 100
	for _, c := range checks {
		switch c.Status {
		case StatusFail:
			switch c.Severity {
			case SevCritical:
				s -= 18
			case SevWarning:
				s -= 10
			default:
				s -= 4
			}
		case StatusWarn:
			if c.Severity == SevWarning || c.Severity == SevCritical {
				s -= 6
			} else {
				s -= 3
			}
		case StatusDeclared:
			s -= 1
		}
	}
	if s < 0 {
		s = 0
	}
	switch {
	case s >= 90:
		return s, "A"
	case s >= 75:
		return s, "B"
	case s >= 60:
		return s, "C"
	case s >= 40:
		return s, "D"
	}
	return s, "E"
}

func rank(c Check) int {
	r := map[string]int{StatusFail: 0, StatusWarn: 10, StatusDeclared: 20, StatusPass: 30, StatusSkipped: 40}[c.Status]
	if c.Severity == SevCritical {
		r -= 2
	} else if c.Severity == SevWarning {
		r--
	}
	return r
}
