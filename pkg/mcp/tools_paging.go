package mcp

import (
	"github.com/vibeasik/vibeasik/pkg/paging"
)

// Patokan paging: ukuran halaman diturunkan dari bobot baris (Aturan 37), bukan angka tetap.

func (s *Server) pagingToolsList() []Tool {
	return []Tool{
		{
			Name: "vibeasik_paging_plan",
			Description: "Plans pagination / how many rows to load per request for ANY list, table, feed or endpoint, from the weight of one row (not a fixed number). " +
				"Give the columns (name, type, size) or a measured row_bytes (serialize ~20 sample rows to JSON and take p90), plus table_rows and UI context. Returns page size, server clamp, OFFSET vs keyset, exact count vs has_more, UI mode (pages / load_more / virtual_list), heavy columns to drop from the list projection, and the row-weight → page-size table. " +
				"Budget: 100 KB raw JSON per page (50 KB mobile), hard cap 512 KB, ≤ 800 DOM nodes on screen, p95 ≤ 300 ms. Call it BEFORE writing a list endpoint, repository query or table/list UI.",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{
				"row_bytes":     prop("integer", "Measured JSON bytes of one list row (p90). Omit if you pass columns"),
				"columns":       map[string]interface{}{"type": "array", "description": "List columns: {name, type (int|bool|uuid|timestamp|varchar|text|json|longtext|blob|enum|float), size? (varchar length), avg? (typical value bytes)}", "items": map[string]interface{}{"type": "object"}},
				"table_rows":    prop("integer", "Approximate row count of the table (0 = unknown)"),
				"mobile":        prop("boolean", "Mobile app / cellular client (50 KB budget)"),
				"nodes_per_row": prop("integer", "DOM nodes (or widgets) one row/card renders; default 20"),
				"deep_paging":   prop("boolean", "Users must jump far into the list (page 50+) → keyset"),
			}},
		},
	}
}

// callPagingTool returns handled=false when name is not a paging tool.
func (s *Server) callPagingTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	if name != "vibeasik_paging_plan" {
		return nil, false, nil
	}
	var in paging.Input
	if err := decodeArgs(args, &in); err != nil {
		return nil, true, err
	}
	plan := paging.Recommend(in)
	md := plan.Markdown() + "\n### Tabel bobot baris → ukuran halaman\n" + paging.Table()
	res, err := jsonResult(map[string]interface{}{"plan": plan, "markdown": md})
	return res, true, err
}
