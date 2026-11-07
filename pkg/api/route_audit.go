package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

// Audit rute mutasi (todo #3252). Autentikasi tidak dipasang per-group: s.localAccess sudah berjalan
// global sebelum route mana pun (lihat local_access.go), sehingga satu middleware menutup seluruh
// route termasuk modul registry/Pro. Audit ini membuktikan tidak ada route mutasi yang lolos tanpa token.

// tokenExemptMutations: satu-satunya rute mutasi yang sengaja dikecualikan dari token lokal
// (berkunci sendiri per sumber; handler-nya menolak permintaan tanpa kunci).
var tokenExemptMutations = map[string]bool{"POST /api/traffic/ingest": true}

var routeParamRe = regexp.MustCompile(`:[A-Za-z0-9_]+\??`)

func isMutatingMethod(m string) bool {
	switch m {
	case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		return true
	}
	return false
}

// MutationRoutes mengembalikan "METHOD /path" unik untuk semua route POST/PUT/PATCH/DELETE terdaftar.
func (s *Server) MutationRoutes() []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range s.app.GetRoutes() {
		if !isMutatingMethod(r.Method) {
			continue
		}
		k := r.Method + " " + r.Path
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// UnguardedMutations mengirim tiap route mutasi tanpa token dan dengan token salah; mengembalikan
// route yang tidak dijawab 401 (dan route kecuali-token yang malah sukses). Kosong = semua terjaga.
func (s *Server) UnguardedMutations() []string {
	var bad []string
	for _, key := range s.MutationRoutes() {
		method, route, _ := strings.Cut(key, " ")
		path := strings.ReplaceAll(routeParamRe.ReplaceAllString(route, "1"), "*", "x")
		for _, tok := range []string{"", "token-salah"} {
			req := httptest.NewRequest(method, path, nil)
			req.Host = "localhost"
			if tok != "" {
				req.Header.Set(localauth.Header, tok)
			}
			resp, err := s.app.Test(req, -1)
			if err != nil {
				bad = append(bad, fmt.Sprintf("%s: %v", key, err))
				break
			}
			code := resp.StatusCode
			_ = resp.Body.Close()
			if tokenExemptMutations[key] {
				if code >= http.StatusOK && code < http.StatusMultipleChoices {
					bad = append(bad, fmt.Sprintf("%s → %d (rute kecuali-token harus menolak tanpa kunci sumber)", key, code))
				}
				break
			}
			if code != http.StatusUnauthorized {
				bad = append(bad, fmt.Sprintf("%s → %d (token=%q), want 401", key, code, tok))
				break
			}
		}
	}
	return bad
}
