package mcp

import "github.com/vibeasik/vibeasik/pkg/mcp/protocol"

// Alias tipe protokol: definisinya di pkg/mcp/protocol agar bisa dipakai registry & modul tanpa siklus impor.
type (
	JSONRPCRequest  = protocol.JSONRPCRequest
	JSONRPCResponse = protocol.JSONRPCResponse
	RPCError        = protocol.RPCError
	Tool            = protocol.Tool
	ToolSchema      = protocol.ToolSchema
	ToolCallParams  = protocol.ToolCallParams
	ToolCallResult  = protocol.ToolCallResult
	ToolContent     = protocol.ToolContent
)
