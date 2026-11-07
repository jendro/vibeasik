package api

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Guard untuk endpoint yang mengeksekusi (atau menyimpan) perintah shell: dev runner, agent execute,
// perubahan dev_command, custom agent, dan tool MCP sejenis. Lapisan kedua setelah token akses lokal
// (local_access.go): walau pemanggil punya token, eksekusi tetap hanya dari loopback + origin localhost,
// kecuali LAN diizinkan sadar lewat env.
const (
	envAllowLANExec = "VIBEASIK_ALLOW_LAN_EXEC"
	envExecToken    = "VIBEASIK_EXEC_TOKEN"
	headerExecToken = "X-Vibeasik-Exec-Token"
)

// execDecision is the outcome of evaluating one request against the exec policy.
type execDecision struct {
	Allowed bool
	Code    string
	Reason  string
}

func lanExecAllowed() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envAllowLANExec))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// evaluateExecAccess decides whether a caller may trigger command execution. remote must be the TCP
// peer address (never a forwarded header), origin the browser Origin header ("" for non-browser clients).
func evaluateExecAccess(remote net.IP, origin, token string) execDecision {
	if want := os.Getenv(envExecToken); want != "" && token != "" &&
		subtle.ConstantTimeCompare([]byte(token), []byte(want)) == 1 {
		return execDecision{Allowed: true}
	}

	allowLAN := lanExecAllowed()
	if !allowLAN && (remote == nil || !remote.IsLoopback()) {
		ip := "unknown"
		if remote != nil {
			ip = remote.String()
		}
		return execDecision{
			Code: "exec_lan_forbidden",
			Reason: fmt.Sprintf("Request dari %s ditolak: aksi ini menjalankan perintah shell di komputer vibeasik, "+
				"jadi secara default hanya boleh dari localhost. Untuk mengizinkan akses LAN, jalankan engine dengan "+
				"%s=1, atau set %s=<rahasia> lalu kirim header %s.", ip, envAllowLANExec, envExecToken, headerExecToken),
		}
	}

	if origin != "" && !trustedExecOrigin(origin, allowLAN) {
		return execDecision{
			Code: "exec_origin_forbidden",
			Reason: fmt.Sprintf("Request dari origin %q ditolak: hanya studio vibeasik yang dibuka dari localhost "+
				"yang boleh menjalankan perintah (mencegah website lain memicunya lewat browser). Buka studio via "+
				"http://localhost atau kirim header %s.", origin, headerExecToken),
		}
	}
	return execDecision{Allowed: true}
}

// trustedExecOrigin accepts loopback origins; with LAN opt-in also private-IP and .local origins (the
// studio opened from a phone). A DNS-rebinding page keeps its own domain as Origin, so it never matches.
func trustedExecOrigin(origin string, allowLAN bool) bool {
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	if !allowLAN {
		return false
	}
	if ip != nil {
		return ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}
	return strings.HasSuffix(host, ".local")
}

func execDecisionFor(c *fiber.Ctx) execDecision {
	return evaluateExecAccess(c.Context().RemoteIP(), c.Get(fiber.HeaderOrigin), c.Get(headerExecToken))
}

func denyExec(c *fiber.Ctx, d execDecision) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": d.Reason, "code": d.Code})
}

// requireExecAccess guards a whole route that executes or defines a shell command.
func requireExecAccess(c *fiber.Ctx) error {
	if d := execDecisionFor(c); !d.Allowed {
		return denyExec(c, d)
	}
	return c.Next()
}

// guardDevCommand is for handlers open to everyone that must still not let an untrusted caller store a
// new dev_command (the runner would execute it later). It returns (true, response) when it denied.
func guardDevCommand(c *fiber.Ctx, current string, next ...string) (bool, error) {
	changed := false
	for _, n := range next {
		if n = strings.TrimSpace(n); n != "" && n != strings.TrimSpace(current) {
			changed = true
			break
		}
	}
	if !changed {
		return false, nil
	}
	if d := execDecisionFor(c); !d.Allowed {
		d.Reason = "Menyimpan dev_command baru ditolak. " + d.Reason
		return true, denyExec(c, d)
	}
	return false, nil
}
