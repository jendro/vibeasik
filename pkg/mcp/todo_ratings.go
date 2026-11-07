package mcp

import (
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// complexityArgSchema & impactArgSchema memungkinkan agent menilai todo sendiri saat create/update_todo;
// bila tidak diisi, engine menaksirnya secara heuristik.
var complexityArgSchema = map[string]interface{}{
	"type":        "object",
	"description": "Optional complexity rating 1-5 per dimension: {logic, coding, scope, risk, note}. Omit to let vibeasik estimate it.",
	"properties": map[string]interface{}{
		"logic": map[string]interface{}{"type": "integer"}, "coding": map[string]interface{}{"type": "integer"},
		"scope": map[string]interface{}{"type": "integer"}, "risk": map[string]interface{}{"type": "integer"},
		"note": map[string]interface{}{"type": "string"},
	},
}

var impactArgSchema = map[string]interface{}{
	"type": "object",
	"description": "Optional impact rating 0-5 per dimension (0 = no effect) → final score 0-100 weighted by category. " +
		"Code quality is judged by its benefit to AI agents (ai_context: easier for an agent to find/understand; maintainability: safer future agent edits); " +
		"humans only feel ux (look & feel) and speed (loading). summary = what improves once done. Omit to let vibeasik estimate it.",
	"properties": map[string]interface{}{
		"ai_context": map[string]interface{}{"type": "integer"}, "maintainability": map[string]interface{}{"type": "integer"},
		"reliability": map[string]interface{}{"type": "integer"}, "security": map[string]interface{}{"type": "integer"},
		"speed": map[string]interface{}{"type": "integer"}, "ux": map[string]interface{}{"type": "integer"},
		"feature": map[string]interface{}{"type": "integer"}, "summary": map[string]interface{}{"type": "string"},
	},
}

func complexityArg(args map[string]interface{}) *models.LLMComplexity {
	var c models.LLMComplexity
	if args["complexity"] == nil || decodeArgs(args["complexity"], &c) != nil {
		return nil
	}
	return &c
}

func impactArg(args map[string]interface{}) *models.LLMImpact {
	var i models.LLMImpact
	if args["impact"] == nil || decodeArgs(args["impact"], &i) != nil {
		return nil
	}
	return &i
}

// firstNonEmptyStr mengembalikan string tak kosong pertama (argumen MCP boleh bukan string).
func firstNonEmptyStr(vals ...interface{}) string {
	for _, v := range vals {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
