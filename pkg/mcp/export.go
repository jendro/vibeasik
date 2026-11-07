package mcp

// Toolkit untuk modul yang menyediakan tool MCP lewat registry: cara membaca argumen & membentuk hasil
// yang sama dengan tool inti.

// Prop membuat satu properti JSON Schema.
func Prop(typ, desc string) map[string]interface{} { return prop(typ, desc) }

// IDSchema schema objek dengan properti id integer wajib (mis. "project_id").
func IDSchema(required ...string) ToolSchema { return idSchema(required...) }

// ArgUint membaca argumen bilangan bulat positif.
func ArgUint(args map[string]interface{}, key string) uint { return argUint(args, key) }

// ArgUints membaca argumen daftar bilangan bulat positif.
func ArgUints(args map[string]interface{}, key string) []uint { return argUints(args, key) }

// ArgString membaca argumen teks (dipangkas).
func ArgString(args map[string]interface{}, key string) string { return argString(args, key) }

// JSONResult membungkus v sebagai hasil tool berisi JSON berindentasi.
func JSONResult(v interface{}) (*ToolCallResult, error) { return jsonResult(v) }

// DecodeArgs mengurai argumen tool ke struct dst.
func DecodeArgs(src interface{}, dst interface{}) error { return decodeArgs(src, dst) }

// FirstNonEmptyStr nilai teks pertama yang tidak kosong.
func FirstNonEmptyStr(vals ...interface{}) string { return firstNonEmptyStr(vals...) }
