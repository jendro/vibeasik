package mcp

import (
	"strings"

	"github.com/vibeasik/vibeasik/pkg/knowledge"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// Pusat Pengetahuan: satu pintu patokan & pengetahuan umum pekerja AI vibeasik. Tool membaca/menulis DB langsung
// sehingga jalan juga dari stdio.

func (s *Server) knowledgeToolsList() []Tool {
	return []Tool{
		{
			Name: "vibeasik_knowledge_recall",
			Description: "Searches vibeasik's Knowledge Hub — the single door for office-wide AI knowledge: guideline rubrics, audit checklists, lessons from audit verdicts, employee expertise and knowledge learned about new technologies. " +
				"Results are ranked by relevance (TF-IDF vectors + linked nodes), proven usefulness (helpful/harmful evidence), importance and recency; rules of other ecosystems are filtered out. " +
				"Use it before working in an unfamiliar area. Project-specific facts are not here (see the project's executive board).",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{
				"query":       prop("string", "What you are working on, e.g. todo title + target files"),
				"domains":     prop("string", "Optional comma-separated ecosystems/areas, e.g. 'go,security'"),
				"employee_id": prop("integer", "Optional: include this employee's personal knowledge"),
				"limit":       prop("integer", "Max nodes (default 10)"),
			}, Required: []string{"query"}},
		},
		{
			Name: "vibeasik_knowledge_learn",
			Description: "Adds reusable, project-independent knowledge to the shared office library of the Knowledge Hub (e.g. how a new language/framework builds & tests, a pitfall you just hit). " +
				"A near-duplicate of an existing node reinforces it instead of duplicating. Do NOT store project-specific facts (paths, table names, features of one app).",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{
				"title":   prop("string", "Short topic"),
				"content": prop("string", "1-3 sentences of concrete, reusable knowledge"),
				"kind":    prop("string", "rule | procedure | lesson | concept (default lesson)"),
				"domain":  prop("string", "Ecosystem or area, e.g. go, web, flutter, gleam, security (auto-detected when empty)"),
			}, Required: []string{"title", "content"}},
		},
		{
			Name:        "vibeasik_knowledge_feedback",
			Description: "Reports whether a Knowledge Hub node (label K<id> in your brief) helped or misled you on the current work. Evidence drives ranking; nodes proven harmful are pruned, guideline rules flagged for review.",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{
				"node_id": prop("integer", "Node id (the number after K)"),
				"helpful": prop("boolean", "true = helped, false = wrong/misleading"),
			}, Required: []string{"node_id", "helpful"}},
		},
		{
			Name: "vibeasik_knowledge_gap",
			Description: "Reports a knowledge gap: a technology, language or method the office does not know yet. The daemon turns open gaps into a learning agenda for the best-fit employee, whose results become shared knowledge nodes. " +
				"Ecosystems with zero nodes are detected automatically; use this for specific new methods inside known ecosystems too.",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{
				"topic":  prop("string", "What needs to be learned, e.g. 'Go 1.26 range-over-func iterators'"),
				"domain": prop("string", "Ecosystem/area (auto-detected when empty)"),
				"sample": prop("string", "Optional context where the gap appeared"),
			}, Required: []string{"topic"}},
		},
	}
}

// callKnowledgeTool returns handled=false when name is not a knowledge tool.
func (s *Server) callKnowledgeTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	switch name {
	case "vibeasik_knowledge_recall":
		q := knowledge.Query{Text: argString(args, "query"), EmployeeID: argUint(args, "employee_id"), Limit: int(argUint(args, "limit"))}
		for _, d := range strings.Split(argString(args, "domains"), ",") {
			if d = strings.TrimSpace(d); d != "" {
				q.Domains = append(q.Domains, d)
			}
		}
		res, err := jsonResult(knowledge.Recall(s.db, q))
		return res, true, err
	case "vibeasik_knowledge_learn":
		node, merged, err := knowledge.Learn(s.db, knowledge.Note{
			Title: argString(args, "title"), Content: argString(args, "content"), Kind: argString(args, "kind"),
			Domain: argString(args, "domain"), Source: models.KnowledgeSrcAgent,
		})
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(map[string]interface{}{"node": node, "merged_into_existing": merged})
		return res, true, err
	case "vibeasik_knowledge_feedback":
		helpful, _ := args["helpful"].(bool)
		node, err := knowledge.Feedback(s.db, argUint(args, "node_id"), helpful)
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(map[string]interface{}{"id": node.ID, "helpful": node.Helpful, "harmful": node.Harmful, "utility": node.Utility()})
		return res, true, err
	case "vibeasik_knowledge_gap":
		g, err := knowledge.ReportGap(s.db, argString(args, "domain"), argString(args, "topic"), argString(args, "sample"))
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(g)
		return res, true, err
	}
	return nil, false, nil
}
