package mcp

import (
	"fmt"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/insight"
)

// Temuan → todo lewat agent: tool ini mengembalikan prompt yang sama dengan tombol "Salin prompt agent" di Studio,
// sehingga permintaan "jadikan todo temuan audit <proyek>" dijalankan agent persis seperti prompt yang disalin pengguna.

func (s *Server) findingsToolsList() []Tool {
	scopes := make([]string, 0, len(insight.FindingScopes))
	for _, sc := range insight.FindingScopes {
		scopes = append(scopes, fmt.Sprintf("'%s' (%s)", sc.Key, sc.Label))
	}
	return []Tool{{
		Name: "vibeasik_findings_todo_prompt",
		Description: "Use FIRST when the user asks to turn findings into todos, e.g. 'jadikan todo temuan audit bimtekapsi', 'buat todo dari biaya proses proyek X'. " +
			"Returns the step-by-step instructions (the same prompt Studio's 'Salin prompt agent' button copies) plus how many findings are open; then FOLLOW those steps yourself: " +
			"verify each finding against the code and create todos via vibeasik_audit_triage_brief / vibeasik_audit_triage_submit, without an external LLM.",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"project_id": prop("integer", "Project ID (or give project)"),
				"project":    prop("string", "Project name or slug as the user wrote it, e.g. 'bimtekapsi' or 'Bimtek APSI'"),
				"scope":      prop("string", "Which findings: "+strings.Join(scopes, ", ")+". Default 'audit' = all findings."),
			},
		},
	}}
}

func (s *Server) callFindingsTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	if name != "vibeasik_findings_todo_prompt" {
		return nil, false, nil
	}
	pid := argUint(args, "project_id")
	if pid == 0 {
		ref, _ := args["project"].(string)
		id, err := insight.ResolveProjectRef(s.db, ref)
		if err != nil {
			return nil, true, err
		}
		pid = id
	}
	scope, _ := args["scope"].(string)
	res, err := insight.BuildFindingsTodoPrompt(s.db, pid, scope)
	if err != nil {
		return nil, true, err
	}
	head := fmt.Sprintf("Proyek %s (#%d) — %d temuan %s terbuka", res.ProjectName, res.ProjectID, res.Open, res.Scope.Label)
	if res.Critical > 0 {
		head += fmt.Sprintf(", %d kritis", res.Critical)
	}
	if res.Open == 0 {
		return textResult(head + ". Tidak ada yang perlu dijadikan todo; laporkan ini ke pengguna."), true, nil
	}
	return textResult(head + ".\nIkuti langkah berikut sekarang:\n\n" + res.Prompt), true, nil
}
