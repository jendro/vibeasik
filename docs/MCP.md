# MCP: memakai vibeasik dari agent coding

vibeasik adalah server [Model Context Protocol](https://modelcontextprotocol.io). Agent seperti Claude Code, Cursor, dan
Antigravity memanggil *tool* vibeasik untuk membaca daftar proyek, menjalankan audit, mengelola todo, dan mencari
pengetahuan, semuanya lewat engine yang berjalan di komputer Anda.

## Pasang otomatis

```bash
vibeasik mcp install --dry-run     # rencana saja, tidak mengubah apa pun
vibeasik mcp install               # daftarkan ke semua agent yang terdeteksi
vibeasik mcp install --agent cursor
vibeasik mcp install --uninstall   # lepas pendaftaran
```

Perilaku yang perlu Anda ketahui:

- Hanya agent yang aplikasinya **sudah terpasang** yang disentuh; tidak ada folder baru yang dibuat.
- Berkas konfigurasi dicadangkan ke `<berkas>.vibeasik.bak` sebelum diubah (sekali).
- Aman dijalankan berulang kali.
- Agent yang didukung: `claude-code`, `claude-desktop`, `cursor`, `windsurf`, `antigravity`, `cline`, `roo`.

Setelah pemasangan, mulai ulang agent.

## Dua cara terhubung

### stdio (bawaan, disarankan)

Agent menjalankan `vibeasik mcp` sebagai proses anak. Tidak perlu token: `vibeasik mcp` membaca `~/.vibeasik/token`.
Konfigurasi manual yang setara:

```json
{
  "mcpServers": {
    "vibeasik": { "command": "vibeasik", "args": ["mcp"] }
  }
}
```

Untuk Claude Code lewat CLI:

```bash
claude mcp add vibeasik -- vibeasik mcp
```

### HTTP

Engine juga menyediakan MCP lewat HTTP di `http://localhost:8765/mcp` (aktif saat `vibeasik serve` berjalan).
Setiap permintaan wajib membawa header `X-Vibeasik-Token: <token>`; token dari `vibeasik token`.

### Engine di komputer lain

Bila engine berjalan di dev server jarak jauh, jembatan stdio ke sana:

```bash
vibeasik mcp --remote http://192.168.1.10:8765 --token <token>
```

Engine di sisi jauh harus dijalankan dengan `--host 0.0.0.0` dan sadar akan risikonya; baca
[SECURITY.md](../SECURITY.md). Tool yang menjalankan perintah shell ditolak dari jaringan kecuali Anda mengaktifkannya
(`VIBEASIK_ALLOW_LAN_EXEC=1`, atau `VIBEASIK_EXEC_TOKEN` + `--exec-token`).

## Melihat tool yang tersedia

```bash
vibeasik mcp list
```

Community menyediakan **64 tool** (proyek, komponen, audit dasar, port, todo, peta kode, pengetahuan, dan lainnya).
Build Official menambah sekitar 59 tool Pro, yang menjawab `upgrade_required` bila paket Anda belum membukanya;
lihat [EDITIONS.md](EDITIONS.md). Setiap tool menjelaskan dirinya sendiri kepada agent, jadi Anda cukup meminta dalam
bahasa biasa.

## Pemecahan masalah

| Gejala | Solusi |
|---|---|
| Agent tidak menampilkan vibeasik | Mulai ulang agent; jalankan `vibeasik mcp install` lagi |
| `401` / "token" dari MCP HTTP | Kirim header `X-Vibeasik-Token`; ambil nilai dari `vibeasik token`. Setelah `token --rotate`, perbarui konfigurasi |
| Tool menjawab "upgrade_required" | Fitur itu bagian paket Pro. `vibeasik whoami` menampilkan fitur aktif Anda |
| Studio atau MCP HTTP tidak bisa dibuka | Jalankan `vibeasik serve`; keduanya dilayani oleh engine |
