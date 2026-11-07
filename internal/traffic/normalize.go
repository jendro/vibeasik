package traffic

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

// Event = satu request dari SDK (bentuk sama dengan models.TrafficIngestEvent di cloud-server).
type Event struct {
	TS       int64   `json:"ts"`
	Method   string  `json:"method"`
	Route    string  `json:"route"`
	Path     string  `json:"path"`
	Status   int     `json:"status"`
	DurMS    float64 `json:"dur_ms"`
	BytesIn  int64   `json:"bytes_in"`
	BytesOut int64   `json:"bytes_out"`
	IP       string  `json:"ip"`
	Country  string  `json:"country"`
	UA       string  `json:"ua"`
	Referer  string  `json:"referer"`
	Instance string  `json:"instance,omitempty"`
}

// RouteNotFound = rute gabungan untuk 404 tanpa pola rute framework (pemindai acak tidak meledakkan jumlah rute).
const RouteNotFound = "(404)"

var (
	reUUID    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	reNum     = regexp.MustCompile(`^\d+$`)
	reHex     = regexp.MustCompile(`^[0-9a-fA-F]{16,}$`)
	reToken   = regexp.MustCompile(`^[A-Za-z0-9_\-]{20,}$`)
	reHasNum  = regexp.MustCompile(`\d`)
	reVersion = regexp.MustCompile(`^v\d+$`)
	reAuth    = regexp.MustCompile(`(?i)(login|signin|sign-in|auth|otp|register|signup|sign-up|password|reset|token|verify|forgot)`)
	reScanner = regexp.MustCompile(`(?i)(\.env|wp-|wordpress|\.git|phpmyadmin|\.php|xmlrpc|\.aws|\.ssh|/admin\.|config\.|backup|\.sql|cgi-bin|/vendor/|actuator|\.well-known/security)`)
)

var staticExt = map[string]bool{
	".js": true, ".mjs": true, ".css": true, ".map": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".avif": true, ".svg": true, ".ico": true, ".woff": true, ".woff2": true, ".ttf": true, ".otf": true,
	".mp4": true, ".webm": true, ".mp3": true, ".pdf": true, ".zip": true, ".txt": true, ".xml": true, ".json": false,
}

// CleanPath membuang query string/fragment & membatasi panjang.
func CleanPath(p string) string {
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if p == "" {
		p = "/"
	}
	return clip(p, 255)
}

// NormalizeRoute memberi pola rute: pola framework dari SDK diutamakan; tanpa itu segmen ber-id diganti placeholder.
func NormalizeRoute(route, p string, status int) string {
	if r := strings.TrimSpace(route); r != "" && r != "*" && r != "/*" {
		return clip(r, 255)
	}
	if status == 404 {
		return RouteNotFound
	}
	if IsStatic(p) {
		return "static:*" + strings.ToLower(path.Ext(p))
	}
	segs := strings.Split(strings.Trim(p, "/"), "/")
	for i, s := range segs {
		switch {
		case s == "":
		case reNum.MatchString(s):
			segs[i] = ":id"
		case reUUID.MatchString(s):
			segs[i] = ":uuid"
		case reHex.MatchString(s):
			segs[i] = ":hash"
		case reToken.MatchString(s) && reHasNum.MatchString(s):
			segs[i] = ":token"
		case len(s) > 40:
			segs[i] = ":slug"
		}
	}
	if len(segs) > 8 {
		segs = append(segs[:8], "*")
	}
	return clip("/"+strings.Join(segs, "/"), 255)
}

func IsStatic(p string) bool {
	return staticExt[strings.ToLower(path.Ext(p))]
}

func IsAuthRoute(route string) bool { return reAuth.MatchString(route) }

func IsScannerPath(p string) bool { return reScanner.MatchString(p) }

// RouteGroup = segmen bermakna pertama (setelah api / vN), dipakai untuk menimbang pemecahan service.
func RouteGroup(route string) string {
	if route == RouteNotFound || strings.HasPrefix(route, "static:") {
		return route
	}
	for _, s := range strings.Split(strings.Trim(route, "/"), "/") {
		if s == "" || strings.EqualFold(s, "api") || reVersion.MatchString(s) {
			continue
		}
		if strings.HasPrefix(s, ":") || strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
			return "/"
		}
		return "/" + s
	}
	return "/"
}

// ClassifyUA → (kelas, keluarga): kelas bot | api | mobile | desktop | unknown.
func ClassifyUA(ua string) (string, string) {
	l := strings.ToLower(ua)
	switch {
	case l == "":
		return "unknown", "-"
	case strings.Contains(l, "bot") || strings.Contains(l, "crawl") || strings.Contains(l, "spider") || strings.Contains(l, "slurp") ||
		strings.Contains(l, "facebookexternalhit") || strings.Contains(l, "headless") || strings.Contains(l, "lighthouse") || strings.Contains(l, "uptime"):
		return "bot", botFamily(l)
	}
	for _, c := range []struct{ key, name string }{
		{"curl/", "curl"}, {"wget", "wget"}, {"python-requests", "python-requests"}, {"python-urllib", "python"}, {"aiohttp", "python"},
		{"go-http-client", "go"}, {"okhttp", "okhttp (android)"}, {"axios", "axios"}, {"node-fetch", "node"}, {"undici", "node"},
		{"dart", "dart/flutter"}, {"postman", "postman"}, {"insomnia", "insomnia"}, {"java/", "java"}, {"guzzle", "php"},
		{"cfnetwork", "ios app"}, {"alamofire", "ios app"}, {"k6", "k6"}, {"apachebench", "ab"},
	} {
		if strings.Contains(l, c.key) {
			return "api", c.name
		}
	}
	family := "lainnya"
	switch {
	case strings.Contains(l, "edg/"):
		family = "Edge"
	case strings.Contains(l, "opr/") || strings.Contains(l, "opera"):
		family = "Opera"
	case strings.Contains(l, "samsungbrowser"):
		family = "Samsung"
	case strings.Contains(l, "firefox/"):
		family = "Firefox"
	case strings.Contains(l, "chrome/") || strings.Contains(l, "crios/"):
		family = "Chrome"
	case strings.Contains(l, "safari/"):
		family = "Safari"
	}
	if strings.Contains(l, "mobile") || strings.Contains(l, "android") || strings.Contains(l, "iphone") || strings.Contains(l, "ipad") {
		return "mobile", family
	}
	if strings.Contains(l, "mozilla/") {
		return "desktop", family
	}
	return "unknown", clip(strings.Fields(ua + " -")[0], 32)
}

func botFamily(l string) string {
	for _, b := range []string{"googlebot", "bingbot", "yandex", "baiduspider", "duckduckbot", "ahrefs", "semrush", "mj12bot", "gptbot", "claudebot", "bytespider", "facebookexternalhit", "petalbot", "applebot", "headless", "lighthouse", "uptime"} {
		if strings.Contains(l, b) {
			return b
		}
	}
	return "bot lain"
}

// RefererHost menyimpan host perujuk saja (path/query perujuk bisa berisi data pribadi).
func RefererHost(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if !strings.Contains(ref, "://") {
		return clip(strings.ToLower(strings.SplitN(ref, "/", 2)[0]), 255)
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return clip(strings.ToLower(u.Host), 255)
}

func clip(v string, n int) string {
	if len(v) > n {
		return v[:n]
	}
	return v
}
