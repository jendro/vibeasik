package models

// CodingPrinciples adalah prioritas koding yang sama untuk semua agent (run orkestrator, sub agent mode divisi,
// agent MCP) dan resolver audit (patokan ARCH-004): aman → benar → sesederhana mungkin, tanpa pemeriksaan ganda.
const CodingPrinciples = "Prioritas kode: aman → benar → sesederhana mungkin. Pilih perubahan terkecil yang menuntaskan masalah; " +
	"pakai ulang helper/pola yang sudah ada sebelum menulis yang baru, dan jangan menambah abstraksi, flag, atau lapisan tanpa kebutuhan nyata. " +
	"Satu pemeriksaan cukup di satu lapisan: sebelum menambah middleware (auth, rate limit, CORS) atau validasi, cek rantai rute (MCP `vibeasik_code_map_routes`) " +
	"dan fungsi pemanggil/yang dipanggil — bila sudah dijaga di global/group/handler/service, jangan dipasang ulang. Bila menemukan salinan ganda di area yang kamu ubah, " +
	"sederhanakan ke satu lapisan, tetapi hanya bila salinan lain terbukti selalu berjalan di jalur yang sama (keamanan tidak boleh melemah). " +
	"Bentuk fungsi: tiap fungsi yang kamu tulis/ubah targetkan kompleksitas siklomatis ≤ 10 (batas keras 15; hitung 1 + tiap if/for/while/case/catch/&&/||/ternary) " +
	"dan if bersarang maksimal 2 tingkat — pakai guard clause/early return, ekstrak tiap langkah ke helper bernama dengan satu tanggung jawab, " +
	"ganti if/switch berantai dengan map/tabel lookup; jangan menambah cabang ke fungsi yang sudah panjang, buat helper baru."
