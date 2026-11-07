package mcp

import (
	"github.com/vibeasik/vibeasik/pkg/registry"
)

// envOrchestratorRunID diisi orkestrator (modul Pro) untuk agent yang ia jalankan.
const envOrchestratorRunID = "VIBEASIK_ORCHESTRATOR_RUN_ID"

// moduleTools mengumpulkan tool MCP dari modul terdaftar. Tool modul yang terkunci tetap terlihat
// (agar agent tahu fitur itu ada) dan menjawab upgrade_required saat dipanggil.
func moduleTools() []Tool {
	var out []Tool
	for _, m := range registry.All() {
		if tp, ok := m.(registry.ToolProvider); ok {
			out = append(out, tp.Tools()...)
		}
	}
	return out
}

// callModuleTool meneruskan panggilan ke modul pemilik tool, setelah memeriksa entitlement.
func (s *Server) callModuleTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	for _, m := range registry.All() {
		tp, ok := m.(registry.ToolProvider)
		if !ok || !ownsTool(tp, name) {
			continue
		}
		if !registry.Enabled(m) {
			return nil, true, registry.UpgradeRequired(m.Feature())
		}
		return tp.CallTool(s.db, name, args)
	}
	return nil, false, nil
}

func ownsTool(tp registry.ToolProvider, name string) bool {
	for _, t := range tp.Tools() {
		if t.Name == name {
			return true
		}
	}
	return false
}
