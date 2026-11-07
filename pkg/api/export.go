package api

import "github.com/gofiber/fiber/v2"

// Ekspor penjaga eksekusi untuk route modul (Pro) yang menjalankan perintah atau agent:
// kebijakan yang sama dengan route inti (hanya loopback, kecuali LAN diizinkan sadar + token).

// ExecDecision hasil penilaian apakah request boleh menjalankan perintah.
type ExecDecision = execDecision

// ExecDecisionFor menilai request c.
func ExecDecisionFor(c *fiber.Ctx) ExecDecision { return execDecisionFor(c) }

// DenyExec menulis jawaban 403 untuk keputusan d.
func DenyExec(c *fiber.Ctx, d ExecDecision) error { return denyExec(c, d) }

// RequireExecAccess middleware Fiber yang menolak eksekusi dari luar laptop ini.
func RequireExecAccess(c *fiber.Ctx) error { return requireExecAccess(c) }
