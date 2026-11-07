// Katalog patokan audit standar & Deep Check yang ditampilkan di AuditCriteriaDrawer.
// ID aturan mengikuti RuleID yang dihasilkan local-engine (internal/diagnostics).

export interface AuditCriterion {
  id: string
  category: 'security' | 'critical_logic' | 'backend' | 'frontend' | 'mobile' | 'smelly_design' | 'code_smell' | 'deployment'
  title: string
  scope: 'Backend (BE)' | 'Frontend (FE)' | 'Mobile' | 'Database' | 'Universal' | 'Backend, Mobile & Skrip (bukan UI Frontend)'
  severity: 'critical' | 'high' | 'warning' | 'info'
  rule: string
  rationale: string
  inspection_check: string
  suggested_action: string
  is_deep?: boolean
  example_code?: {
    bad: string
    good: string
  }
}

// Master Audit Criteria Catalog covering BE, FE, Mobile, Security, Critical Queries/Processes, Smelly Code & Design
export const AUDIT_CRITERIA: AuditCriterion[] = [
  // ================= 1. KEAMANAN & VULNERABILITY =================
  {
    id: 'SEC-SQLI',
    category: 'security',
    title: 'SQL Injection via Raw String Concatenation',
    scope: 'Database',
    severity: 'critical',
    rule: 'Dilarang keras menyambung parameter input pengguna langsung ke query SQL menggunakan formatting string (fmt.Sprintf, template literal, atau operator +). Seluruh query wajib memakai prepared statement & parameterized placeholder (?, $1, :id).',
    rationale: 'Penyerang dapat menyuntikkan perintah SQL jahat untuk membocorkan seluruh database, merusak tabel, atau melewati proteksi autentikasi (OWASP Top 10 A03).',
    inspection_check: 'Pindai pemanggilan db.Raw, db.Exec, db.Where, atau .query() yang memuat string concatenation dengan variabel pengguna.',
    suggested_action: 'Gunakan parameter placeholder: db.Where("email = ?", email) atau db.Raw("SELECT id, name FROM users WHERE id = ?", id).',
    example_code: {
      bad: 'Contoh tidak aman: membentuk query SQL dengan fmt.Sprintf atau penyambungan string memakai input pengguna.',
      good: 'db.Raw("SELECT id, name, email FROM users WHERE email = ?", inputEmail)'
    }
  },
  {
    id: 'SEC-COMMAND-INJECTION',
    category: 'security',
    title: 'Command Injection & Remote Code Execution (RCE)',
    scope: 'Backend (BE)',
    severity: 'critical',
    rule: 'Dilarang melewatkan input pengguna langsung ke pemanggilan subproses sistem (exec.Command, child_process.exec, os.system, eval, system, passthru). Jangan gunakan shell invocation (sh -c / bash -c) dengan string gabungan.',
    rationale: 'Penyerang dapat menyisipkan separator perintah (;, &&, |) untuk mengeksekusi script arbitrary, menghapus file sistem host, atau memasang backdoor server.',
    inspection_check: 'Deteksi pemanggilan exec.Command, child_process, eval(), atau system() yang menggabungkan parameter dinamis.',
    suggested_action: 'Gunakan whitelist argumen tetap, validasi input secara ketat, dan lewatkan argumen sebagai slice/array terpisah tanpa shell interpreter.',
    example_code: {
      bad: 'DILARANG: shell invocation dengan input pengguna yang digabungkan menjadi satu perintah.',
      good: 'DISARANKAN: API proses native dengan executable tetap dan argumen terpisah yang sudah divalidasi.'
    }
  },
  {
    id: 'SEC-PATH-TRAVERSAL',
    category: 'security',
    title: 'Path Traversal & Arbitrary File Access',
    scope: 'Backend (BE)',
    severity: 'critical',
    rule: 'Semua operasi pembacaan, penulisan, atau pengunduhan file (os.ReadFile, fs.readFile) yang menerima nama file dari klien wajib dibersihkan dengan filepath.Clean() dan diverifikasi berada di dalam batas direktori dasar (safe root boundary check).',
    rationale: 'Input seperti "../../etc/passwd" atau "....//config.json" memungkinkan penyerang membaca file sensitif di luar folder penyimpanan publik.',
    inspection_check: 'Periksa operasi I/O file yang menerima parameter query/param langsung tanpa filepath.Clean dan path boundary verification.',
    suggested_action: 'Gunakan filepath.Clean(filepath.Join(baseDir, filename)), lalu validasi filepath.Rel(baseDir, cleanPath). Tolak hasil yang sama dengan ".." atau yang diawali ".." + string(os.PathSeparator). Jangan mengandalkan strings.HasPrefix karena /safe/root-evil dapat lolos pemeriksaan prefix.',
    example_code: {
      bad: 'data, err := os.ReadFile("./uploads/" + c.Params("filename"))',
      good: 'clean := filepath.Clean(filepath.Join(baseDir, c.Params("filename")))\nrelative, err := filepath.Rel(baseDir, clean)\nif err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) { return c.Status(403).SendString("Akses ditolak") }\ndata, err := os.ReadFile(clean)'
    }
  },
  {
    id: 'SEC-XSS-UNSAFE-HTML',
    category: 'security',
    title: 'Cross-Site Scripting (XSS) via Raw HTML Injection',
    scope: 'Frontend (FE)',
    severity: 'critical',
    rule: 'Dilarang merender teks dari input pengguna atau endpoint eksternal menggunakan v-html (Vue), dangerouslySetInnerHTML (React), atau innerHTML tanpa melewati library sanitizer HTML tepercaya (seperti DOMPurify).',
    rationale: 'Script jahat (<script> atau <img onerror=...>) dapat dieksekusi di browser korban untuk mencuri token session, cookie autentikasi, atau melakukan defacing.',
    inspection_check: 'Pindai template komponen terhadap keberadaan v-html atau innerHTML yang tidak dibungkus fungsi sanitasi.',
    suggested_action: 'Gunakan text interpolation biasa ({{ text }}), atau bersihkan dengan DOMPurify.sanitize(rawHtml) sebelum dirender.',
    example_code: {
      bad: '<div v-html="userBio" />',
      good: '<div v-html="DOMPurify.sanitize(userBio)" />'
    }
  },
  {
    id: 'SEC-HARDCODED-SECRETS',
    category: 'security',
    title: 'Hardcoded Secrets, Tokens & Private Keys',
    scope: 'Universal',
    severity: 'critical',
    rule: 'Dilarang keras menuliskan password database, JWT secret key, API key pihak ketiga, atau sertifikat privat langsung di dalam kode sumber (.go, .ts, .vue, .dart, .py). Semua kredensial wajib dipanggil melalui environment variables.',
    rationale: 'Kredensial yang tertulis di kode mudah bocor saat repository di-commit ke Git atau dipublikasikan ke bundle client-side frontend.',
    inspection_check: 'Pindai string literal yang memuat pola jwt_secret, api_key, db_password, private_key dengan string token >12 karakter.',
    suggested_action: 'Pindahkan seluruh secret ke file .env dan baca via os.Getenv() atau process.env.',
    example_code: {
      bad: 'const jwtSecret = "<EXAMPLE_ONLY_NOT_A_SECRET>"\n// Go: jwtSecret := "<EXAMPLE_ONLY_NOT_A_SECRET>"',
      good: '// TypeScript/Node (backend saja; jangan masuk bundle client)\nconst jwtSecret = process.env.JWT_SECRET\n\n// Go\njwtSecret := os.Getenv("JWT_SECRET")'
    }
  },
  {
    id: 'SEC-BOLA-IDOR',
    category: 'security',
    title: 'Broken Object Level Authorization (BOLA / IDOR)',
    scope: 'Backend (BE)',
    severity: 'critical',
    rule: 'Setiap endpoint mutasi (PUT/PATCH/DELETE) atau pengambilan data sensitif berbasis ID wajib memvalidasi bahwa entitas tersebut dimiliki oleh user atau tenant yang sedang login. Jangan hanya mengandalkan ID dari parameter URL.',
    rationale: 'BOLA/IDOR adalah celah nomor satu di OWASP API Top 10: penyerang cukup mengganti angka ID di URL untuk memanipulasi atau menghapus data pengguna lain.',
    inspection_check: 'Periksa handler update/delete apakah menyertakan klausul kepemilikan WHERE user_id = current_user.id.',
    suggested_action: 'Sertakan user_id dari JWT token ke dalam filter kueri: db.Where("id = ? AND user_id = ?", id, currentUserID).Delete(&record).',
    example_code: {
      bad: 'db.Delete(&Order{}, "id = ?", req.ID)',
      good: 'db.Where("id = ? AND user_id = ?", req.ID, user.ID).Delete(&Order{})'
    }
  },
  {
    id: 'SEC-SSRF-GUARD',
    category: 'security',
    title: 'Server-Side Request Forgery (SSRF) Mitigations',
    scope: 'Backend (BE)',
    severity: 'critical',
    rule: 'Endpoint yang melakukan request HTTP keluar berdasarkan URL masukan pengguna wajib memvalidasi domain terhadap allowlist dan menolak seluruh IP privat internal (127.0.0.1, 10.0.0.0/8, 192.168.0.0/16, 169.254.169.254).',
    rationale: 'Penyerang dapat menggunakan server sebagai proksi untuk membaca metadata AWS/GCP, memindai port lokal internal, atau menyerang servis mikro di dalam jaringan VPC.',
    inspection_check: 'Pindai fungsi HTTP client yang menerima URL dinamis dari payload request tanpa IP validation.',
    suggested_action: 'Resolve DNS dan tolak alamat IP loopback serta IP subnet privat RFC 1918 sebelum membuka koneksi HTTP.',
    example_code: {
      bad: 'resp, err := http.Get(userProvidedURL)',
      good: 'if !isSafePublicURL(userProvidedURL) { return errors.New("URL tidak diizinkan") }\nresp, err := safeClient.Get(userProvidedURL)'
    }
  },

  // ================= 2. LOGIKA KRITIS: KUERI & PROSES BERAT =================
  {
    id: 'CRIT-NPLUS1-QUERY',
    category: 'critical_logic',
    title: 'N+1 Query Anti-Pattern di Dalam Perulangan',
    scope: 'Database',
    severity: 'critical',
    rule: 'Dilarang mengeksekusi kueri database (db.Find, db.Query, repo.Get) di dalam perulangan (for, while, map, foreach). Data relasional wajib diambil secara batch di luar loop menggunakan Eager Loading (GORM Preload, SQL JOIN, atau WHERE IN).',
    rationale: 'Jika ada 1.000 data, perulangan mengeksekusi 1.001 kali round-trip kueri ke database. Ini memicu latensi kumulatif puluhan detik dan menghabiskan connection pool.',
    inspection_check: 'Pindai pemanggilan method database di dalam blok for/while loop atau array map.',
    suggested_action: 'Gunakan GORM Preload atau kumpulkan ID terlebih dahulu lalu panggil db.Where("id IN ?", ids).',
    example_code: {
      bad: 'for _, user := range users {\n    db.Find(&orders, "user_id = ?", user.ID)\n}',
      good: 'db.Preload("Orders").Find(&users)'
    }
  },
  {
    id: 'CRIT-SELECT-STAR',
    category: 'critical_logic',
    title: 'SELECT * Projection Anti-Pattern (Unbounded Columns)',
    scope: 'Database',
    severity: 'warning',
    rule: 'Hindari penggunaan SELECT * pada kueri produksi dan servis API bervolume tinggi. Sebutkan kolom yang dibutuhkan secara spesifik dan sertakan LIMIT/paginasi.',
    rationale: 'SELECT * mematikan optimasi covering index, membebani bandwidth transfer jaringan antar-server, dan menghabiskan RAM untuk menyimpan data kolom biner/teks panjang yang tidak dipakai.',
    inspection_check: 'Pindai kueri SQL mentah atau ORM yang memuat SELECT * tanpa batasan kolom.',
    suggested_action: 'Gunakan db.Select("id, title, status, created_at").Limit(50).',
    example_code: {
      bad: 'db.Raw("SELECT * FROM transactions WHERE status = ?", status)',
      good: 'db.Select("id, reference_no, amount, status").Where("status = ?", status).Limit(50)'
    }
  },
  {
    id: 'CRIT-WILDCARD-LIKE',
    category: 'critical_logic',
    title: 'Kueri Berat: Wildcard Prefix LIKE \'%...\' (Full Table Scan)',
    scope: 'Database',
    severity: 'warning',
    rule: 'Dilarang menggunakan kueri dengan karakter wildcard di awal (LIKE \'%keyword%\' atau ILIKE \'%keyword%\') pada tabel bervolume besar tanpa Full-Text Search (FTS) index.',
    rationale: 'Wildcard di awal mematikan struktur B-Tree index dan memaksa mesin database membaca setiap baris data dari disk ke memori (Full Table Scan / O(N) disk I/O).',
    inspection_check: 'Pindai klausul LIKE atau ILIKE yang diawali dengan tanda persen %.',
    suggested_action: 'Gunakan pencarian prefix LIKE \'keyword%\' atau implementasikan FTS5 di SQLite / tsvector di PostgreSQL.',
    example_code: {
      bad: 'db.Where("description LIKE ?", "%" + query + "%")',
      good: '// Gunakan prefix matching atau Full-Text Search:\ndb.Where("description LIKE ?", query + "%")'
    }
  },
  {
    id: 'CRIT-CURSOR-LEAK',
    category: 'critical_logic',
    title: 'Database Cursor Leak: sql.Rows Tidak Ditutup (Missing defer Close)',
    scope: 'Database',
    severity: 'critical',
    rule: 'Setiap pemanggilan kueri database yang menghasilkan kursor baris (rows, err := db.Query(...)) WAJIB segera diikuti oleh statement \'defer rows.Close()\'.',
    rationale: 'Jika rows tidak ditutup, koneksi database tetap ditahan dan tidak dikembalikan ke pool. Ketika traffic meningkat, connection pool akan habis (exhausted) dan server mengalami deadlock total.',
    inspection_check: 'Deteksi pemanggilan db.Query() pada bahasa Go yang tidak memiliki defer rows.Close() di fungsinya.',
    suggested_action: 'Tambahkan defer rows.Close() tepat setelah pengecekan if err != nil.',
    example_code: {
      bad: 'rows, err := db.Query("SELECT id FROM users")\nfor rows.Next() { ... }',
      good: 'rows, err := db.Query("SELECT id FROM users")\nif err != nil { return err }\ndefer rows.Close()\nfor rows.Next() { ... }'
    }
  },
  {
    id: 'CRIT-TX-ROLLBACK',
    category: 'critical_logic',
    title: 'Dangling Database Transaction: Transaksi Tanpa Guard Rollback',
    scope: 'Database',
    severity: 'warning',
    rule: 'Setiap transaksi database manual (tx := db.Begin()) WAJIB memiliki pengaman \'defer tx.Rollback()\' tepat setelah pembukaan transaksi.',
    rationale: 'Jika terjadi panic atau error return di tengah fungsi sebelum tx.Commit(), transaksi akan menggantung di status idle in transaction dan mengunci baris data (table/row lock).',
    inspection_check: 'Deteksi pembukaan transaksi database tanpa keberadaan defer tx.Rollback().',
    suggested_action: 'Pasang defer tx.Rollback() langsung setelah db.Begin(). Jika tx.Commit() berhasil dieksekusi, Rollback() akan diabaikan secara aman.',
    example_code: {
      bad: 'tx := db.Begin()\nif err := tx.Create(&item).Error; err != nil { return err }\ntx.Commit()',
      good: 'tx := db.Begin()\ndefer tx.Rollback()\nif err := tx.Create(&item).Error; err != nil { return err }\ntx.Commit()'
    }
  },
  {
    id: 'CRIT-SYNC-IO-BLOCKING',
    category: 'critical_logic',
    title: 'Proses Berat: Operasi File Sinkron Memblokir Event Loop',
    scope: 'Backend (BE)',
    severity: 'critical',
    rule: 'Dilarang menggunakan fungsi I/O file sinkron (seperti fs.readFileSync, fs.writeFileSync di Node.js, atau eksekusi CPU intensif tanpa goroutine di Go) di dalam handler request HTTP.',
    rationale: 'Node.js bekerja dengan single-threaded event loop. Pemanggilan file sinkron membekukan seluruh proses server sehingga ribuan request lain mengalami antrean dan timeout.',
    inspection_check: 'Pindai penggunaan method fs.*Sync di dalam controller atau route handler.',
    suggested_action: 'Ganti pemanggilan sinkron dengan asynchronous fs.promises.readFile atau streaming read stream.',
    example_code: {
      bad: 'const fileData = fs.readFileSync(filePath, "utf-8")',
      good: 'const fileData = await fs.promises.readFile(filePath, "utf-8")'
    }
  },
  {
    id: 'CRIT-UNBOUNDED-BODY-READ',
    category: 'critical_logic',
    title: 'Proses Berat: Pembacaan Request Body Tanpa Batas Ukuran (OOM Risk)',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Dilarang membaca seluruh request body menggunakan io.ReadAll(r.Body) tanpa membatasi ukuran pembacaan (misal dengan http.MaxBytesReader atau io.LimitReader).',
    rationale: 'Penyerang dapat mengirimkan request multipart berukuran gigabyte untuk menguras memori RAM server seketika hingga proses crash (Out-Of-Memory kill).',
    inspection_check: 'Pindai pembacaan io.ReadAll pada HTTP body tanpa pembatas ukuran.',
    suggested_action: 'Batasi pembacaan stream: r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // Maks 10MB.',
    example_code: {
      bad: 'bodyBytes, err := io.ReadAll(c.Request().Body)',
      good: 'limited := io.LimitReader(c.Request().Body, 5*1024*1024)\nbodyBytes, err := io.ReadAll(limited)'
    }
  },

  // ================= 3. BACKEND (BE) & API SMELLS =================
  {
    id: 'BE-GOD-CONTROLLER',
    category: 'backend',
    title: 'God Controller & Large Handler Smell (>400 LOC)',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'File controller atau handler HTTP tidak boleh melebihi 400 baris kode atau menangani terlalu banyak entitas sekaligus. Handler hanya boleh memvalidasi request, memanggil service layer, dan memetakan status response.',
    rationale: 'File controller yang membengkak melanggar Single Responsibility Principle (SRP), sulit dibuatkan unit test, dan rentan memicu merge conflict di tim pengembang.',
    inspection_check: 'Hitung jumlah baris pada berkas-berkas di folder handler, controller, atau api.',
    suggested_action: 'Pecah controller menjadi sub-controller berbasis domain (misal: user_handler, order_handler) dan pindahkan logika bisnis ke Service layer.',
    example_code: {
      bad: '// Satu file api_handler.go memuat 1.200 baris mengurus user, order, produk, dan audit',
      good: '// Dipecah menjadi modul rapi:\n// - user_handler.go\n// - order_handler.go\n// - product_handler.go'
    }
  },
  {
    id: 'BE-UNGUARDED-MUTATION',
    category: 'backend',
    title: 'Rute Mutasi Data (POST/PUT/DELETE) Tanpa Middleware Autentikasi',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Rute yang mengubah kondisi data sistem (POST, PUT, PATCH, DELETE) wajib dilindungi oleh middleware autentikasi (JWT / Session Guard), kecuali endpoint publik yang sengaja dikecualikan (login, register, webhook).',
    rationale: 'Rute mutasi publik memungkinkan pengguna tanpa kredensial memasukkan data sampah, memodifikasi resource, atau menghapus entitas sistem.',
    inspection_check: 'Pindai definisi rute POST/PUT/PATCH/DELETE pada kode aplikasi yang tidak memuat handler middleware Auth. Kecualikan string di dalam properti example_code (blok contoh adalah dokumentasi, bukan rute yang terdaftar).',
    suggested_action: 'Pasang middleware pelindung sebelum handler: api.Post("/items", authMiddleware, createItem).',
    example_code: {
      bad: 'Rute mutasi pengubah role didaftarkan tanpa middleware autentikasi sebelum handler.',
      good: 'app.Post("/api/users/update-role", authMiddleware, adminOnlyGuard, updateRoleHandler)'
    }
  },
  {
    id: 'BE-WILDCARD-CORS',
    category: 'backend',
    title: 'Kebijakan CORS Terlalu Permisif (AllowOrigins: \'*\')',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Dilarang mengizinkan seluruh domain (wildcard \'*\') pada konfigurasi CORS API jika endpoint mendukung kredensial pengguna atau menangani data rahasia.',
    rationale: 'CORS wildcard membuka celah bagi website jahat di browser pengguna untuk memanggil API dan membaca data respon rahasia.',
    inspection_check: 'Deteksi AllowOrigins: "*" atau origin: "*" pada middleware CORS.',
    suggested_action: 'Tentukan origin domain frontend secara spesifik (misal: "http://localhost:3000, https://app.example.com").',
    example_code: {
      bad: 'cors.New(cors.Config{ AllowOrigins: "*" })',
      good: 'cors.New(cors.Config{\n    AllowOrigins: []string{"http://localhost:3000", "https://app.mycompany.com"},\n    AllowCredentials: true,\n})'
    }
  },
  {
    id: 'BE-SWALLOWED-ERROR',
    category: 'backend',
    title: 'Swallowed Error Smell: Error Diabaikan Tanpa Penanganan',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Dilarang mengabaikan error return (misal: \'_ = err\' di Go, blok \'catch() {}\' kosong di JS, atau \'except: pass\' di Python). Setiap error wajib dicatat dalam log terstruktur atau diteruskan ke error handler layer.',
    rationale: 'Mengabaikan error menyebabkan silent failure: sistem tetap berjalan seolah berhasil padahal terjadi kegagalan database atau I/O, menghasilkan inkonsistensi data.',
    inspection_check: 'Pindai pola _ = err, catch () {}, atau except: pass pada kode sumber.',
    suggested_action: 'Tangani error secara eksplisit dan catat konteks error ke logger sistem.',
    example_code: {
      bad: '_ = db.Save(&user).Error // Error ditelan mentah-mentah',
      good: 'if err := db.Save(&user).Error; err != nil {\n    log.Printf("[ERROR] gagal menyimpan user: %v", err)\n    return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan data"})\n}'
    }
  },
  {
    id: 'BE-MISSING-TIMEOUT',
    category: 'backend',
    title: 'HTTP Client Tanpa Batas Waktu (Infinite Timeout Hazard)',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Inisialisasi HTTP client (seperti &http.Client{} di Go atau axios tanpa timeout) wajib menyertakan nilai Timeout eksplisit (misal: 10 * time.Second).',
    rationale: 'Client tanpa timeout akan menggantung selamanya jika server tujuan mengalami network congestion, menghabiskan thread pool dan menumpuk goroutine hingga server kehabisan memori.',
    inspection_check: 'Deteksi inisialisasi http.Client default atau http.Get langsung tanpa konfigurasi timeout.',
    suggested_action: 'Tetapkan nilai timeout yang masuk akal pada struct client atau gunakan context.WithTimeout.',
    example_code: {
      bad: 'client := &http.Client{} // Default timeout adalah 0 (tidak pernah timeout!)',
      good: 'client := &http.Client{\n    Timeout: 10 * time.Second,\n}'
    }
  },
  {
    id: 'BE-REST-VERB-IN-PATH',
    category: 'backend',
    title: 'Pelanggaran RESTful URI: Penggunaan Kata Kerja di Path Rute',
    scope: 'Backend (BE)',
    severity: 'info',
    rule: 'URL endpoint REST API harus menggunakan kata benda jamak (nouns) untuk merepresentasikan resource, bukan kata kerja (actions). Aksi direpresentasikan oleh HTTP Method (GET, POST, PUT, DELETE).',
    rationale: 'Mencampur kata kerja di URI (misal /getUser, /deleteItem) merusak standar RESTful Richardson Maturity Model dan membingungkan integrasi klien otomatis.',
    inspection_check: 'Pindai pola rute HTTP yang mengandung get, fetch, delete, remove, create, add di dalam URL path.',
    suggested_action: 'Ubah pola URL: ganti GET /getUser/:id menjadi GET /users/:id, dan POST /deleteUser menjadi DELETE /users/:id.',
    example_code: {
      bad: 'Rute dengan kata kerja di path: GET /getUser/:id dan POST /deleteItem/:id.',
      good: 'Rute berbasis resource: GET /users/:id dan DELETE /items/:id.'
    }
  },
  {
    id: 'ROUTER-DUPLICATE-MIDDLEWARE',
    category: 'backend',
    title: 'Middleware Terpasang Ganda di Rantai Rute',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Middleware yang sudah aktif di global/group (auth, rate limit, CORS) tidak boleh dipasang ulang di group/rute anaknya (patokan ARCH-004: satu pemeriksaan di satu lapisan).',
    rationale: 'Middleware ganda berjalan dua kali per request (latensi, kuota rate limit terpotong ganda) dan membuat aturan mudah berbeda di dua tempat saat salah satunya diubah.',
    inspection_check: 'Peta rute (routemap) menelusuri rantai middleware efektif tiap rute Go dari main → group → rute; nama middleware yang sama muncul dua kali di satu rantai dilaporkan di lokasi pemasangan keduanya.',
    suggested_action: 'Hapus pemasangan ulang dan biarkan middleware di lapisan yang paling luas. Bila konfigurasinya memang berbeda (mis. limit login lebih ketat), beri nama/konfigurasi yang jelas berbeda.',
    example_code: {
      bad: 'api := app.Group("/api", middleware.RequireAuth())\napi.Post("/users", middleware.RequireAuth(), handlers.CreateUser)',
      good: 'api := app.Group("/api", middleware.RequireAuth())\napi.Post("/users", handlers.CreateUser)'
    }
  },
  {
    id: 'ROUTER-DUPLICATE-VALIDATION',
    category: 'backend',
    title: 'Validasi Ganda di Jalur Rute (Handler → Service)',
    scope: 'Backend (BE)',
    severity: 'info',
    rule: 'Kondisi validasi yang sama tidak diperiksa di dua fungsi pada jalur satu rute (handler dan service yang dipanggilnya). Pilih satu lapisan: handler/validator untuk bentuk input, service untuk aturan domain (patokan ARCH-004).',
    rationale: 'Validasi ganda membuat aturan bisa berbeda di dua tempat (salah satu lupa diperbarui) dan menambah kode tanpa menambah keamanan. Cek nil defensif dan pemeriksaan error tidak dihitung.',
    inspection_check: 'Untuk tiap rute Go, if-validasi yang langsung return di handler dan fungsi yang dijangkaunya dinormalkan (req.Email vs in.Email → Email); kondisi sama di fungsi berbeda dilaporkan di lokasi kedua.',
    suggested_action: 'Hapus salinan di salah satu lapisan. Pertahankan salinan service bila service itu juga dipanggil dari jalur lain (CLI, job, MCP) tanpa validasi handler.',
    example_code: {
      bad: '// handler\nif strings.TrimSpace(req.Email) == "" { return c.SendStatus(400) }\n// service\nif strings.TrimSpace(in.Email) == "" { return ErrEmailRequired }',
      good: '// handler: cukup teruskan error domain\nif err := svc.CreateUser(req.Email); errors.Is(err, ErrEmailRequired) { return c.SendStatus(400) }'
    }
  },

  // ================= 4. FRONTEND (FE) & SMELLY DESIGN =================
  {
    id: 'FE-ZERO-ALERT',
    category: 'smelly_design',
    title: 'Penggunaan Dialog Bawaan Browser (Zero Alert Policy)',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Dilarang keras memanggil dialog synchronous bawaan browser (alert(), confirm(), prompt()). Notifikasi wajib menggunakan Toast (useToast()) dan konfirmasi wajib memakai Modal Dialog kustom yang elegan.',
    rationale: 'Dialog browser bersifat synchronous dan membekukan thread antarmuka, tampilannya jelek dan berbeda-beda di setiap OS, serta merusak pengalaman pengguna modern.',
    inspection_check: 'Pindai kode Vue/TS/JS terhadap pemanggilan alert(, confirm(, atau prompt(.',
    suggested_action: 'Ganti alert() dengan useToast() dari Nuxt UI dan konfirmasi dengan komponen Modal kustom.',
    example_code: {
      bad: 'if (confirm("Yakin hapus?")) { deleteItem() }',
      good: '// Gunakan modal dialog konfirmasi reaktif atau toast notification'
    }
  },
  {
    id: 'FE-LISTENER-LEAK',
    category: 'frontend',
    title: 'SPA Memory Leak: Listener & Timer Tidak Dibersihkan',
    scope: 'Frontend (FE)',
    severity: 'high',
    rule: 'Setiap event listener (window.addEventListener, document.on) atau setInterval di dalam komponen Vue/React wajib dibersihkan di dalam lifecycle hook onUnmounted / onBeforeUnmount.',
    rationale: 'Pada aplikasi Single Page Application, event listener yang tidak dilepas akan terus hidup di background, menahan memori DOM komponen yang sudah ditutup, dan memicu kebocoran memori.',
    inspection_check: 'Deteksi addEventListener atau setInterval tanpa pasangan removeEventListener atau clearInterval di onUnmounted.',
    suggested_action: 'Gunakan useEventListener dari VueUse atau daftarkan pembersihan di onUnmounted(() => window.removeEventListener(...)).',
    example_code: {
      bad: 'onMounted(() => {\n    window.addEventListener("resize", handleResize)\n}) // Lupa unmounted!',
      good: 'onMounted(() => {\n    window.addEventListener("resize", handleResize)\n})\nonUnmounted(() => {\n    window.removeEventListener("resize", handleResize)\n})'
    }
  },
  {
    id: 'FE-LAYOUT-SHIFTS',
    category: 'smelly_design',
    title: 'Cumulative Layout Shift (CLS) & Ketiadaan Skeleton Loader',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Halaman yang memuat data asinkron wajib menyertakan skeleton placeholder yang proporsional dengan layout konten akhir, serta menentukan rasio aspek atau lebar/tinggi eksplisit untuk gambar dan media.',
    rationale: 'Konten yang tiba-tiba muncul dan menggeser elemen di bawahnya (layout shift) menyebabkan pengguna salah klik tombol dan menurunkan skor Core Web Vitals (Google SEO).',
    inspection_check: 'Periksa komponen yang memuat data async: apakah memiliki skeleton loader atau hanya teks loading polos yang melompat.',
    suggested_action: 'Pasang komponen USkeleton dengan dimensi serupa elemen asli saat status pending/loading aktif.',
    example_code: {
      bad: '<div v-if="loading">Loading...</div>\n<div v-else class="h-64 grid grid-cols-3">...</div>',
      good: '<div v-if="loading" class="h-64 grid grid-cols-3 gap-4">\n    <USkeleton class="h-32 rounded-xl" v-for="i in 3" :key="i" />\n</div>\n<div v-else class="h-64 grid grid-cols-3 gap-4">...</div>'
    }
  },
  {
    id: 'FE-CONTRAST-A11Y',
    category: 'smelly_design',
    title: 'Rasio Kontras Rendah & Aksesibilitas Buruk (WCAG AA Violation)',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Rasio kontras warna teks terhadap latar belakang wajib memenuhi standar minimal 4.5:1 untuk teks normal dan 3:1 untuk teks besar. Hindari penggunaan teks abu-abu pudar (text-neutral-400 pada background terang).',
    rationale: 'Kontras rendah membuat teks tidak terbaca oleh pengguna dengan gangguan penglihatan atau saat layar terkena pantulan sinar matahari.',
    inspection_check: 'Pindai kombinasi warna teks dan latar belakang menggunakan contrast checker antislop-human.',
    suggested_action: 'Gunakan kelas teks dengan kontras tegas: text-neutral-800 dark:text-neutral-100.',
    example_code: {
      bad: '<p class="text-neutral-300 bg-white">Teks penting tapi tidak terbaca</p>',
      good: '<p class="text-neutral-800 dark:text-neutral-200">Teks kontras tinggi dan mudah dibaca</p>'
    }
  },
  {
    id: 'FE-API-RELOAD-UNCACHED',
    category: 'frontend',
    title: 'Pemanggilan API Berulang Tanpa Caching Variabel (Tab Switch Reload Smell)',
    scope: 'Frontend (FE)',
    severity: 'high',
    rule: 'Pemanggilan API backend pada sub-tab atau halaman dinamis wajib disimpan di variabel/state lokal atau cache agar tidak di-reload berulang kali setiap kali pengguna berpindah tab antarmuka.',
    rationale: 'Tanpa caching variabel di level halaman, perpindahan sub-tab akan terus-menerus memicu request HTTP ke backend, menyebabkan skeleton flickering dan keborosan bandwidth serta komputasi server.',
    inspection_check: 'Deteksi pemanggilan API backend di onMounted() sub-tab tanpa pengujian cache (contoh: if (!data.value) fetchData()).',
    suggested_action: 'Simpan data di ref/useState level halaman atau composable induk, lalu lekatkan props/cache ke sub-komponen tab.',
    example_code: {
      bad: '// Di sub-komponen tab:\nonMounted(() => {\n  loadHeavyDataFromBackend() // Re-fetch setiap ganti tab!\n})',
      good: '// Di halaman induk:\nconst cachedData = ref(null)\nconst loadData = async () => {\n  if (!cachedData.value) cachedData.value = await fetchApi()\n}\n// Di sub-tab: gunakan props cachedData'
    }
  },
  {
    id: 'FE-OVERFETCHING-UNUSED-DATA',
    category: 'frontend',
    title: 'Overfetching API: Pemanggilan Data Berat yang Tidak Digunakan di Halaman',
    scope: 'Frontend (FE)',
    severity: 'high',
    rule: 'Komponen dilarang memanggil endpoint API berat jika data responnya tidak digunakan atau dirender secara esensial ke antarmuka pengguna.',
    rationale: 'Memanggil API agregasi besar hanya untuk membaca 1-2 metrik kecil membuang waktu jaringan, memicu proses kompilasi/render di backend yang sia-sia, dan memperlambat waktu respon UI.',
    inspection_check: 'Pindai pemanggilan API agregasi (seperti base-knowledge atau visual audit) pada tab yang tidak merender data tersebut.',
    suggested_action: 'Hapus pemanggilan API yang tidak dibutuhkan, atau gunakan data yang sudah tersedia di props / endpoint yang lebih ringan.',
    example_code: {
      bad: '// Tab ide/chat memanggil full base-knowledge hanya untuk ambil apps_count:\nconst bk = await getProjectBaseKnowledge(id)\nconst count = bk.apps_count',
      good: '// Gunakan props yang sudah ada:\nconst count = computed(() => props.project?.components?.length || 0)'
    }
  },

  // ================= 5. MOBILE (FLUTTER / REACT NATIVE / NATIVE) =================
  {
    id: 'MOB-UI-THREAD-FREEZE',
    category: 'mobile',
    title: 'Pencegahan UI Thread Freezing & ANR pada Mobile',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Semua komputasi berat, parsing JSON berukuran besar (>50KB), enkripsi/hashing, dan pengolahan gambar pada aplikasi mobile dilarang dijalankan di UI/Main Thread. Wajib gunakan compute(), Isolate.run, atau Dispatchers.Default. Pada Flutter, build() tidak boleh berisi jsonDecode / IO sinkron sama sekali.',
    rationale: 'Menjalankan proses berat di main thread membuat frame rate anjlok di bawah 60 FPS (jank) dan memicu dialog ANR (Application Not Responding) yang mematikan aplikasi paksa.',
    inspection_check: 'Flutter (statis): jsonDecode, json.decode, readAs*Sync, writeAsStringSync, atau sleep yang dieksekusi langsung di build() (di luar closure seperti onPressed) → WARNING. Android dicek oleh aturan AND-* & review LLM.',
    suggested_action: 'Bungkus pemrosesan ke dalam isolate: final data = await compute(parseLargeJson, responseBody).',
    example_code: {
      bad: '// Dijalankan langsung di UI Thread:\nfinal data = jsonDecode(hugeStringPayload);',
      good: '// Dijalankan di background Isolate:\nfinal data = await compute(parseLargeJson, hugeStringPayload);'
    }
  },
  {
    id: 'MOB-CONTROLLER-LEAK',
    category: 'mobile',
    title: 'Resource Flutter Tidak Dilepas (Controller / FocusNode / Stream / Timer)',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Setiap TextEditingController, ScrollController, PageController, TabController, AnimationController, FocusNode, ValueNotifier, StreamController, StreamSubscription, dan Timer yang DIBUAT oleh kelas itu sendiri wajib dilepas di dispose(): .dispose() / .close() / .cancel().',
    rationale: 'Controller yang tidak di-dispose akan terus ditahan oleh framework di memori RAM bersama seluruh konteks widget, memicu kebocoran memori kumulatif hingga crash Out-Of-Memory.',
    inspection_check: 'Per field kelas: field yang diinisialisasi atau di-assign di kelas tersebut (initState, .listen(), Timer.periodic) harus muncul pada pemanggilan dispose/close/cancel atau di dalam method dispose(). Field yang diterima dari konstruktor (milik widget induk) tidak dihitung. Hanya untuk aplikasi Flutter.',
    suggested_action: 'Tambahkan override method dispose() dan panggil controller.dispose() serta super.dispose().',
    example_code: {
      bad: 'class _MyScreenState extends State<MyScreen> {\n  final _controller = TextEditingController();\n  // Lupa menulis dispose()!\n}',
      good: 'class _MyScreenState extends State<MyScreen> {\n  final _controller = TextEditingController();\n\n  @override\n  void dispose() {\n    _controller.dispose();\n    super.dispose();\n  }\n}'
    }
  },
  {
    id: 'MOB-SETSTATE-IN-BUILD',
    category: 'mobile',
    title: 'Larangan Pemanggilan setState() di Dalam Method Build',
    scope: 'Mobile',
    severity: 'critical',
    rule: 'Dilarang keras memanggil setState() atau mutasi state apa pun langsung di dalam tubuh method Widget build(BuildContext context). Mutasi state hanya boleh terjadi di event listener atau lifecycle hook.',
    rationale: 'Memanggil setState di dalam build memicu siklus perulangan render tak berujung (infinite rebuild loop), mengunci thread rendering, dan membuat baterai perangkat terkuras cepat.',
    inspection_check: 'Flutter: setState() yang dieksekusi langsung di tubuh Widget build(BuildContext) — closure (onPressed: () => setState(...), addPostFrameCallback) tidak dihitung.',
    suggested_action: 'Pindahkan pembaruan state ke event listener tombol atau gunakan WidgetsBinding.instance.addPostFrameCallback.',
    example_code: {
      bad: 'Widget build(BuildContext context) {\n  setState(() { counter++; }); // Bahaya fatal!\n  return Text("$counter");\n}',
      good: 'Widget build(BuildContext context) {\n  return ElevatedButton(\n    onPressed: () => setState(() { counter++; }),\n    child: Text("$counter"),\n  );\n}'
    }
  },
  {
    id: 'MOB-INSECURE-LOCAL-STORAGE',
    category: 'mobile',
    title: 'Penyimpanan Kredensial Tanpa Enkripsi pada Mobile',
    scope: 'Mobile',
    severity: 'critical',
    rule: 'Dilarang menyimpan token JWT, kredensial autentikasi, atau data sensitif pengguna ke dalam SharedPreferences mentah, NSUserDefaults, atau AsyncStorage tanpa enkripsi. Gunakan FlutterSecureStorage, Android Keystore, atau iOS Keychain.',
    rationale: 'SharedPreferences disimpan dalam berkas XML teks polos di storage perangkat, yang dapat diekstraksi dengan mudah melalui backup ADB atau perangkat yang di-root/jailbreak.',
    inspection_check: 'Flutter: SharedPreferences.setString/setStringList dengan kunci (string literal atau nama konstanta) bermakna token, jwt, password, secret, credential, api_key, session_id, pin, atau otp.',
    suggested_action: 'Beralih ke package flutter_secure_storage atau EncryptedSharedPreferences bawaan Android Jetpack Security.',
    example_code: {
      bad: 'final prefs = await SharedPreferences.getInstance();\nprefs.setString("jwt_token", token);',
      good: 'final storage = FlutterSecureStorage();\nawait storage.write(key: "jwt_token", value: token);'
    }
  },
  {
    id: 'MOB-OFFLINE-HANDLING',
    category: 'mobile',
    title: 'Penanganan Koneksi Offline & Network Resiliency',
    scope: 'Mobile',
    severity: 'high',
    rule: 'Aplikasi mobile harus menangani status jaringan terputus (SocketException / No Internet) dengan baik melalui caching lokal dan tampilan peringatan offline yang ramah, bukan layar putih atau error crash.',
    rationale: 'Koneksi jaringan seluler sering terputus atau beralih BTS. Ketiadaan penanganan offline merusak persepsi keandalan aplikasi di mata pengguna.',
    inspection_check: 'Periksa service API mobile apakah menangani SocketException dan menyediakan fallback data cache.',
    suggested_action: 'Simpan salinan data lokal di SQLite/Isar dan tampilkan komponen status offline dengan tombol Coba Lagi.',
    example_code: {
      bad: 'final res = await http.get(url); // Crash jika tidak ada koneksi internet',
      good: 'try {\n  return await fetchOnline();\n} on SocketException {\n  return getLocalCachedData();\n}'
    }
  },

  // ================= 6. SMELLY CODE & ARSITEKTUR =================
  {
    id: 'CODE-TIGHT-COUPLING',
    category: 'code_smell',
    title: 'Tight Coupling & Ketiadaan Abstraksi Repository',
    scope: 'Backend (BE)',
    severity: 'high',
    rule: 'Lapisan Controller dan Service dilarang mengimpor atau memanggil langsung koneksi database mentah (*gorm.DB). Seluruh kueri wajib dienkapsulasi di dalam interface Repository yang independen.',
    rationale: 'Mengikat langsung controller dengan ORM membuat kode mustahil di-mock saat unit test dan menyulitkan pergantian database di masa mendatang.',
    inspection_check: 'Pindai handler/service apakah menerima parameter struct *gorm.DB langsung alih-alih interface repository.',
    suggested_action: 'Bungkus kueri database ke dalam interface repository di internal/repository.',
    example_code: {
      bad: 'type UserService struct { DB *gorm.DB }',
      good: 'type UserService struct { repo UserRepository }'
    }
  },
  {
    id: 'CODE-STUB-HANDLER',
    category: 'code_smell',
    title: 'Dangling / Stub Endpoint Belum Terimplementasi',
    scope: 'Backend (BE)',
    severity: 'info',
    rule: 'Handler rute tidak boleh dibiarkan dengan komentar placeholder // TODO: implement atau mengembalikan string kosong tanpa status HTTP 501 (Not Implemented).',
    rationale: 'Endpoint dangling yang dipanggil klien dapat menimbulkan bug tak terduga jika dianggap sudah berfungsi.',
    inspection_check: 'Pindai handler yang memuat // TODO implement atau panic("not implemented").',
    suggested_action: 'Selesaikan implementasi handler atau kembalikan response HTTP 501 dengan format JSON standar.',
    example_code: {
      bad: 'func UpdateUser(c *fiber.Ctx) error { return nil }',
      good: 'func UpdateUser(c *fiber.Ctx) error {\n    return c.Status(501).JSON(fiber.Map{"error": "Endpoint dalam tahap pengembangan"})\n}'
    }
  },
  // ================= 7. DEPLOY: MULTI ENV & LOKASI BERKAS =================
  // Engine: local-engine/internal/diagnostics/env_hygiene.go & storage_inspector.go
  {
    id: 'ENV-FILE-TRACKED',
    category: 'security',
    title: 'Berkas Env Ter-commit ke Git',
    scope: 'Universal',
    severity: 'critical',
    rule: 'Hanya template .env.example (juga .env.sample/.env.template/.env.dist) yang boleh tercatat di git. .env, .env.local, dan setiap .env.server* per server tujuan tidak boleh ter-commit.',
    rationale: 'Berkas yang sudah ter-commit tetap terunggah ke remote & semua server walau polanya kemudian ditambahkan ke .gitignore, dan secret-nya tersimpan di riwayat git.',
    inspection_check: 'git ls-files di folder proyek/komponen, cari berkas bernama .env atau .env.* selain template.',
    suggested_action: 'git rm --cached <berkas>, tutup polanya di .gitignore, lalu rotate semua secret di dalamnya.',
    example_code: {
      bad: '$ git ls-files\n.env.example\n.env.server.jakarta   # ikut terunggah',
      good: '$ git ls-files\n.env.example'
    }
  },
  {
    id: 'ENV-NOT-IGNORED',
    category: 'security',
    title: 'Berkas Env di Disk Tidak Ter-ignore',
    scope: 'Universal',
    severity: 'critical',
    rule: 'Setiap berkas env yang ada di disk (.env, .env.local, .env.server, .env.server.<nama>, .env.production, ...) wajib cocok dengan pola .gitignore.',
    rationale: 'Berkas env yang tidak ter-ignore akan ikut pada commit berikutnya (git add .) dan membocorkan kredensial.',
    inspection_check: 'git check-ignore --no-index untuk setiap berkas env di folder (menghitung wildcard, negasi ! dan .gitignore bertingkat persis seperti git).',
    suggested_action: 'Pasang blok wildcard env di .gitignore dan kecualikan hanya .env.example.',
    example_code: {
      bad: '# .gitignore\n.env\n.env.local',
      good: '# .gitignore\n.env\n.env.*\n!.env.example'
    }
  },
  {
    id: 'ENV-IGNORE-INCOMPLETE',
    category: 'security',
    title: 'Pola .gitignore Belum Menutup Env Server Berikutnya',
    scope: 'Universal',
    severity: 'warning',
    rule: 'Aplikasi yang diunggah ke banyak server memakai satu berkas env per server (.env.server.<nama>). Pola .gitignore wajib wildcard sehingga server baru otomatis tertutup, bukan daftar nama server yang sudah ada.',
    rationale: 'Pola per nama (.env.server.jakarta) membuat env server baru (.env.server.surabaya) lolos ke git tanpa disadari.',
    inspection_check: 'Uji nama yang belum ada (.env, .env.local, .env.server, .env.server.server-baru) dengan git check-ignore --no-index pada folder yang memakai env.',
    suggested_action: 'Ganti pola per nama dengan .env + .env.* + !.env.example.',
    example_code: {
      bad: '.env\n.env.server.jakarta',
      good: '.env\n.env.*\n!.env.example'
    }
  },
  {
    id: 'ENV-EXAMPLE-IGNORED',
    category: 'deployment',
    title: '.env.example Ikut Ter-ignore',
    scope: 'Universal',
    severity: 'warning',
    rule: 'Template .env.example wajib bisa di-commit: pola .env* / .env.* harus disertai pengecualian !.env.example.',
    rationale: 'Tanpa template, server dan developer baru tidak tahu variabel apa yang harus diisi.',
    inspection_check: 'git check-ignore --no-index pada template yang belum tercatat di git.',
    suggested_action: 'Tambahkan !.env.example setelah pola env lalu commit template-nya.',
    example_code: {
      bad: '.env*',
      good: '.env\n.env.*\n!.env.example'
    }
  },
  {
    id: 'ENV-EXAMPLE-MISSING',
    category: 'deployment',
    title: 'Template .env.example Belum Ada',
    scope: 'Universal',
    severity: 'warning',
    rule: 'Folder yang punya berkas env (.env, .env.local, .env.server*) wajib menyediakan .env.example dengan semua kunci bernilai dummy/kosong.',
    rationale: 'Setiap server baru disiapkan dengan menyalin template menjadi .env.server.<nama>; tanpa template, konfigurasi ditebak dan mudah terlewat.',
    inspection_check: 'Cek keberadaan template di setiap folder (root proyek & komponen) yang memiliki berkas env.',
    suggested_action: 'Buat .env.example berisi seluruh kunci tanpa nilai asli dan commit.',
    example_code: {
      bad: '.env.local\n.env.server.jakarta\n# tidak ada .env.example',
      good: '# .env.example\nPORT=8080\nDATABASE_URL=\nSTORAGE_DIR=/var/lib/nama-app/storage'
    }
  },
  {
    id: 'ENV-EXAMPLE-DRIFT',
    category: 'deployment',
    title: 'Kunci Env Tidak Terdokumentasi di Template',
    scope: 'Universal',
    severity: 'warning',
    rule: 'Setiap kunci yang dipakai di .env, .env.local, atau .env.server* wajib tercantum di .env.example.',
    rationale: 'Kunci baru yang hanya ditambahkan ke env lokal membuat server berikutnya (disiapkan dari template) kekurangan konfigurasi dan gagal saat runtime.',
    inspection_check: 'Bandingkan nama kunci (tanpa nilai) di setiap berkas env dengan kunci di template; yang dilaporkan hanya nama kunci.',
    suggested_action: 'Tambahkan kunci yang kurang ke .env.example dengan nilai dummy/kosong.',
    example_code: {
      bad: '# .env.local\nJWT_SECRET=...\n# .env.example\nPORT=',
      good: '# .env.example\nPORT=\nJWT_SECRET='
    }
  },
  {
    id: 'STORAGE-RELATIVE-PATH',
    category: 'deployment',
    title: 'Folder Simpan/Muat Berkas Relatif terhadap Working Directory',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Bila tidak memakai object storage (S3/MinIO/GCS/Azure Blob/Cloudinary), folder penyimpanan berkas & database SQLite wajib tetap: dibaca dari env (mis. STORAGE_DIR) dengan default absolut di sebelah binary (os.Executable), dibuat dengan MkdirAll saat start, dan di server ditempatkan di luar folder rilis (mis. /var/lib/<app>).',
    rationale: 'Binary hasil build (Go/Rust/Java/Node bundle) dijalankan systemd/supervisor/docker dari working directory lain, sehingga path relatif ("./uploads", "app.db") menyimpan ke tempat berbeda atau gagal memuat berkas lama; folder di dalam rilis ikut terhapus saat deploy ulang.',
    inspection_check: 'Pindai kode server: literal relatif ke folder uploads/storage/media/files/data atau berkas .db/.sqlite sebagai argumen I/O berkas, static server, SaveFile, atau nilai uploadDir/UPLOAD_FOLDER/dest. Dilewati bila kode memakai object storage.',
    suggested_action: 'Pusatkan di satu helper storageDir() dari env + lokasi binary, daftarkan STORAGE_DIR di .env.example.',
    example_code: {
      bad: 'uploadDir := "./uploads"\nc.SaveFile(file, "uploads/"+name)',
      good: 'func storageDir() string {\n    if d := os.Getenv("STORAGE_DIR"); d != "" { return d }\n    exe, _ := os.Executable()\n    return filepath.Join(filepath.Dir(exe), "storage")\n}\nc.SaveFile(file, filepath.Join(storageDir(), "uploads", name))'
    }
  },
  {
    id: 'STORAGE-TEMP-DIR',
    category: 'deployment',
    title: 'Berkas Persisten Disimpan di Folder Temp',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Berkas unggahan/penyimpanan permanen dilarang disimpan di os.TempDir(), os.tmpdir(), tempfile.gettempdir(), sys_get_temp_dir(), atau /tmp.',
    rationale: 'Folder temp dibersihkan saat reboot dan terisolasi per service bila systemd PrivateTmp aktif, sehingga berkas hilang.',
    inspection_check: 'Baris kode server yang memakai folder temp sistem bersama kata upload/storage/media/attachment/avatar/document. Dilewati bila kode memakai object storage.',
    suggested_action: 'Simpan di folder penyimpanan tetap (STORAGE_DIR); temp hanya untuk berkas sementara.',
    example_code: {
      bad: 'filepath.Join(os.TempDir(), "uploads", name)',
      good: 'filepath.Join(storageDir(), "uploads", name)'
    }
  },
  {
    id: 'STORAGE-SOURCE-RELATIVE',
    category: 'deployment',
    title: 'Folder Berkas Diturunkan dari Lokasi Source (runtime.Caller)',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Program Go dilarang menurunkan folder data/template dari path berkas source (runtime.Caller(0) + filepath.Dir).',
    rationale: 'Path source dibekukan saat kompilasi; binary di server tidak membawa source tree sehingga path menunjuk folder mesin build.',
    inspection_check: 'runtime.Caller(0) yang diikuti filepath.Dir dalam 4 baris.',
    suggested_action: 'Pakai os.Executable() atau env, atau tanam berkas statis dengan //go:embed.',
    example_code: {
      bad: '_, b, _, _ := runtime.Caller(0)\nbase := filepath.Dir(b)',
      good: '//go:embed templates\nvar templates embed.FS'
    }
  },
  {
    id: 'STORAGE-MACHINE-PATH',
    category: 'deployment',
    title: 'Path Absolut Mesin Developer Tertanam di Kode',
    scope: 'Backend (BE)',
    severity: 'warning',
    rule: 'Kode server dilarang memuat path home mesin developer (/home/<user>/, /Users/<user>/, C:\\Users\\<user>\\).',
    rationale: 'Path tersebut tidak ada di server, jadi simpan/muat berkas gagal setelah aplikasi diunggah.',
    inspection_check: 'Literal string berawalan /home/<nama>/, /Users/<nama>/, atau <drive>:\\Users\\<nama>\\ di kode server; nama placeholder (user, username, example, ...) diabaikan.',
    suggested_action: 'Ganti dengan nilai env atau path yang diturunkan dari lokasi binary / os.UserHomeDir().',
    example_code: {
      bad: 'root := "/home/<user>/app/data" // path laptop developer',
      good: 'root := os.Getenv("STORAGE_DIR")'
    }
  },
  // ================= 7b. DEFAULT PROYEK: dev.sh, dev-mobile.sh, deploy.sh, PORT & KONEKSI FE→BE (Aturan 45) =================
  // Engine: local-engine/internal/diagnostics/dev_tooling.go
  {
    id: 'DEV-SCRIPT-MISSING',
    category: 'deployment',
    title: 'dev.sh Wajib: Semua Komponen Jalan dengan Satu Perintah',
    scope: 'Universal',
    severity: 'warning',
    rule: 'Root proyek wajib punya dev.sh executable yang menyalakan semua komponen (BE, FE, mobile) paralel, PORT dari blok PORTS.md, dan URL API BE diekspor ke FE/mobile.',
    rationale: 'Proyek langsung jalan tanpa mengingat perintah tiap komponen, dan port selalu sinkron antara BE dan FE.',
    inspection_check: 'DEV-SCRIPT-MISSING (tidak ada), DEV-SCRIPT-NOT-EXECUTABLE (tanpa chmod +x), DEV-SCRIPT-INCOMPLETE (folder komponen tidak disebut).',
    suggested_action: 'Buat dev.sh: PORT per komponen, ekspor URL API, jalankan dengan & lalu wait, trap \'kill 0\' EXIT.',
    example_code: {
      bad: '# README: buka 3 terminal, jalankan go run . lalu bun dev',
      good: 'trap \'kill 0\' EXIT\n(cd backend && PORT=4105 go run .) &\n(cd frontend && PORT=4100 NUXT_PUBLIC_API_BASE=http://localhost:4105 bun run dev) &\nwait'
    }
  },
  {
    id: 'MOBILE-DEV-SCRIPT-MISSING',
    category: 'deployment',
    title: 'dev-mobile.sh Wajib: Aplikasi Flutter Langsung ke HP',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Proyek dengan komponen Flutter wajib punya dev-mobile.sh executable di root (Aturan 49, patokan nemudijalan): default menjalankan flutter run di PC Windows lewat flutter-win (sync kode via SSH, HP tercolok ke PC), atau ./dev-mobile.sh laptop; API_BASE_URL diarahkan ke IP LAN laptop & port BE dari PORTS.md.',
    rationale: 'Aplikasi Android bisa dicoba langsung di HP dari laptop dev dengan satu perintah, dan tetap memuat data dari BE yang jalan di laptop.',
    inspection_check: 'MOBILE-DEV-SCRIPT-MISSING (tidak ada), MOBILE-DEV-SCRIPT-NOT-EXECUTABLE (tanpa chmod +x), MOBILE-DEV-SCRIPT-INCOMPLETE (tidak menyebut folder mobile, flutter-win, atau API_BASE_URL).',
    suggested_action: 'Salin template dev-mobile.sh dari todo standar vibeasik (folder mobile, port BE & folder unik di PC Windows sudah terisi), chmod +x, commit.',
    example_code: {
      bad: '# README: cd mobile && flutter run (API ke 10.0.2.2, tidak sampai ke BE dari HP)',
      good: './dev-mobile.sh              # PC Windows via flutter-win (default)\n./dev-mobile.sh laptop       # Flutter di laptop\n./dev-mobile.sh devices      # daftar HP/emulator'
    }
  },
  {
    id: 'DEPLOY-SCRIPT-MISSING',
    category: 'deployment',
    title: 'deploy.sh Wajib: Pakai .env.<tujuan>, Lalu Kembali ke .env.local',
    scope: 'Universal',
    severity: 'warning',
    rule: 'Root proyek wajib punya deploy.sh. Pengguna cukup mengisi .env.server (atau .env.<nama>, ./deploy.sh <nama>); skrip memasangnya sebagai env aktif, build & upload, lalu selalu mengembalikan .env.local (trap EXIT). Hanya .env.example yang masuk git.',
    rationale: 'Deploy tanpa edit konfigurasi manual, kredensial server tidak bocor ke git, dan laptop dev tidak tertinggal memakai env server.',
    inspection_check: 'DEPLOY-SCRIPT-MISSING, DEPLOY-ENV-NOT-RESTORED (tanpa .env.local + trap), DEPLOY-ENV-NOT-SELECTABLE (tidak memakai .env.server/.env.$NAMA).',
    suggested_action: 'TARGET=${1:-server}; trap restore EXIT; cp .env.$TARGET .env; build; rsync ke DEPLOY_HOST:DEPLOY_PATH.',
    example_code: {
      bad: 'cp .env.server .env && rsync ... # .env.local tidak dipulihkan',
      good: 'trap \'cp .env.local .env\' EXIT\ncp ".env.${1:-server}" .env\nbun run build && rsync -az .output "$DEPLOY_HOST:$DEPLOY_PATH/"'
    }
  },
  {
    id: 'PORT-SHARED',
    category: 'deployment',
    title: 'Port Bentrok Antar Proyek',
    scope: 'Universal',
    severity: 'warning',
    rule: 'Port efektif tiap komponen (run_port, atau port deklarasi) tidak boleh sama dengan komponen proyek lain: setiap proyek memakai blok port PORTS.md sendiri.',
    rationale: 'Semua proyek harus bisa dijalankan serentak saat dev tanpa address already in use.',
    inspection_check: 'RuleID PORT-SHARED-<port> bila port sama dipakai komponen proyek lain.',
    suggested_action: 'vibeasik_assign_port ke port blok proyek, perbarui env PORT, dev.sh, dan URL API FE/mobile dalam satu commit.'
  },
  {
    id: 'FE-API-PORT-MISMATCH',
    category: 'deployment',
    title: 'FE & Mobile Gagal Memuat Data BE saat Dev',
    scope: 'Universal',
    severity: 'warning',
    rule: 'URL API FE/mobile di env menunjuk port BE proyek yang sama; mobile tidak memakai localhost (pakai IP LAN atau 10.0.2.2 emulator Android).',
    rationale: 'Port BE yang berpindah atau localhost di emulator membuat halaman kosong walau BE berjalan.',
    inspection_check: 'FE-API-PORT-MISMATCH (URL API di .env.local/.env/.env.development ≠ port BE) dan MOBILE-API-LOCALHOST (localhost di env atau kode mobile).',
    suggested_action: 'Ekspor URL API dari dev.sh berdasarkan PORT BE; mobile lewat --dart-define API_BASE_URL=http://$LAN_IP:$BE_PORT.',
    example_code: {
      bad: 'const base = \'http://localhost:8080\' // di aplikasi Flutter',
      good: 'const base = String.fromEnvironment(\'API_BASE_URL\')'
    }
  },
  {
    id: 'FE-DUPLICATE-IMPORT',
    category: 'frontend',
    title: 'Duplicate Import & WARN Duplicated imports Nuxt',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Satu modul diimpor sekali per berkas; satu nama auto-import Nuxt hanya diekspor dari satu berkas di composables/ atau utils/.',
    rationale: 'Unimport diam-diam memakai salah satu definisi kembar, jadi kode bisa memanggil fungsi yang salah.',
    inspection_check: 'Import modul sama berulang dalam satu berkas JS/TS/Vue; nama ekspor kembar antar berkas composables/ & utils/.',
    suggested_action: 'Gabungkan import; sisakan satu definisi atau ganti nama salah satunya.',
    example_code: {
      bad: 'import { ref } from \'vue\'\nimport { computed } from \'vue\'',
      good: 'import { ref, computed } from \'vue\''
    }
  },
  // ================= 8. DEEP CHECK SPESIFIK (SIMULASI REAL-TIME & SINTETIK) =================
  {
    id: 'DEEP-ROUTE-LATENCY',
    category: 'critical_logic',
    title: 'Simulasi Waktu Eksekusi & Latensi Rute API',
    scope: 'Backend (BE)',
    severity: 'critical',
    is_deep: true,
    rule: 'Seluruh endpoint API publik maupun terproteksi harus merespons di bawah ambang batas interaktif (Target: <200ms). Rute dengan estimasi eksekusi >500ms dikategorikan sebagai Bottleneck Kritis.',
    rationale: 'Latensi tinggi (>500ms) memicu penumpukan connection pool, timeout pada gateway/frontend, dan merusak kenyamanan responsivitas pengguna.',
    inspection_check: 'Hanya diuji saat Deep Check: Mengukur latency real-time probe (jika service aktif) atau menghitung simulasi sintetik berbasis beban I/O kueri database, operasi file, dan network calls di dalam handler.',
    suggested_action: 'Terapkan Redis/In-Memory response caching, indexing database, atau delegasikan kalkulasi berat ke background job queue (202 Accepted).',
    example_code: {
      bad: '// Kueri beruntun serial tanpa cache di dalam handler:\nfor _, id := range ids { db.Find(&item, id) }',
      good: '// Gunakan batch query dan in-memory cache:\ndb.Where("id IN ?", ids).Find(&items)\ncache.Set(cacheKey, items, 5*time.Minute)'
    }
  },
  {
    id: 'DEEP-FORM-VALIDATION',
    category: 'backend',
    title: 'Validasi Skema Form & Request Payload Eksternal',
    scope: 'Backend (BE)',
    severity: 'warning',
    is_deep: true,
    rule: 'Setiap rute mutasi state (POST, PUT, PATCH) yang menerima payload dari klien wajib memiliki skema validasi deklaratif (Go validator struct tags, Zod schema di TS/JS, atau FormRequest).',
    rationale: 'Membiarkan input klien langsung diproses tanpa validasi skema memicu bug fatal, data korup, bypass tipe data, hingga kerentanan injeksi.',
    inspection_check: 'Hanya diuji saat Deep Check: Memindai keberadaan deklarasi validator (validate:"required", zod.object, joi, atau manual validation guard) pada handler mutasi.',
    suggested_action: 'Bungkus payload dengan schema validation library dan kembalikan HTTP 400 Bad Request jika data tidak valid.',
    example_code: {
      bad: 'var req UserPayload\nc.BodyParser(&req)\ndb.Create(&req)',
      good: 'var req UserPayload\nif err := c.BodyParser(&req); err != nil { return err }\nif err := validator.New().Struct(req); err != nil { return c.Status(400).JSON(fiber.Map{"error": err.Error()}) }'
    }
  },
  {
    id: 'DEEP-QUERY-UNPAGINATED',
    category: 'critical_logic',
    title: 'Potensi Unpaginated Bulk Query pada Koleksi Data',
    scope: 'Database',
    severity: 'warning',
    is_deep: true,
    rule: 'Pengambilan data tabel koleksi (db.Find, SELECT) pada endpoint API wajib menyertakan klausul batas LIMIT dan OFFSET atau cursor-based pagination.',
    rationale: 'Kueri tanpa limit akan memuat seluruh data ke memori RAM aplikasi. Saat volume data mencapai puluhan ribu baris, server dapat mengalami OOM (Out Of Memory) crash.',
    inspection_check: 'Hanya diuji saat Deep Check: Mendeteksi kueri db.Find() atau SELECT tanpa LIMIT/OFFSET atau pagination helper pada handler koleksi.',
    suggested_action: 'Selalu terapkan batas default (misal limit: 20 atau 50) dengan pagination query parameter.',
    example_code: {
      bad: 'db.Find(&allOrders)',
      good: 'db.Limit(pageSize).Offset((page - 1) * pageSize).Find(&orders)'
    }
  },
  {
    id: 'DEEP-ROUTE-RATE-LIMIT',
    category: 'security',
    title: 'Perlindungan Rate Limiting pada Endpoint API Publik',
    scope: 'Backend (BE)',
    severity: 'warning',
    is_deep: true,
    rule: 'API wajib menerapkan middleware rate limiting untuk membatasi frekuensi request per alamat IP atau token sesi klien.',
    rationale: 'Mencegah serangan brute-force credential stuffing, API scraping berlebihan, dan Denial-of-Service (DoS).',
    inspection_check: 'Hanya diuji saat Deep Check: Memeriksa keberadaan middleware limiter.New / express-rate-limit pada konfigurasi router aplikasi.',
    suggested_action: 'Pasang middleware rate limiter pada entry-point router publik (misal maksimal 120 request per menit per IP).',
    example_code: {
      bad: 'app.Post("/api/auth/login", loginHandler)',
      good: 'app.Post("/api/auth/login", limiter.New(limiter.Config{ Max: 10, Expiration: 1 * time.Minute }), loginHandler)'
    }
  },
  {
    id: 'DEEP-FUNC-HIGH-COMPLEXITY',
    category: 'code_smell',
    title: 'Kompleksitas Siklomatis Fungsi Berlebihan (CC > 15)',
    scope: 'Backend, Mobile & Skrip (bukan UI Frontend)',
    severity: 'info',
    is_deep: true,
    rule: 'Kompleksitas siklomatis McCabe (1 + jumlah if/for/while/case/catch/&&/||/ternary) satu fungsi maksimal 15. CC 16–30 = INFO, CC > 30 = WARNING (pita risiko SEI: 21–50 tinggi, > 50 tidak dapat diuji). Operator ?? tidak dihitung karena hanya nilai default (mis. copyWith).',
    rationale: 'CC adalah jumlah minimum jalur independen yang harus diuji agar semua cabang tercakup; fungsi ber-CC tinggi berkorelasi dengan densitas bug dan biaya review. Metrik ini berguna sebagai penunjuk prioritas refactor, bukan bukti bug, sehingga diberi bobot rendah dan hanya jalan di Deep Check. Kode UI frontend tidak diukur: berjalan di browser satu pengguna sehingga cabang rumit tidak berdampak ke pengguna lain — untuk FE yang diaudit adalah ukuran komponen (FE-LARGE-COMPONENT / FE-LARGE-MODULE).',
    inspection_check: 'Hanya diuji saat Deep Check: setiap fungsi bernama (Go, TS/JS non-UI, Dart, Kotlin, Java, PHP, Python) diukur dari deklarasi hingga akhir badannya (kurung kurawal / indentasi Python). Berkas UI frontend (.vue, .svelte, .tsx/.jsx di luar server/, TS/JS di components/, pages/, composables/, stores/, hooks/, layouts/, konfigurasi nuxt/vite/next) dilewati. String, komentar, berkas test & hasil generator (*.g.dart, *.freezed.dart, *.pb.go, "DO NOT EDIT") diabaikan. Untuk Go hasilnya identik dengan perhitungan AST gocyclo.',
    suggested_action: 'Pecah menjadi sub-fungsi bernama, gunakan guard clause / early return untuk meratakan if bersarang, dan ganti if/switch berantai berbasis tipe dengan map lookup atau polymorphism.',
    example_code: {
      bad: 'const applyPreset = (role) => {\n  if (role === \'backend\') { /* 8 baris */ }\n  else if (role === \'web\') { /* 8 baris */ }\n  else if (role === \'mobile\') { /* ... */ }\n}',
      good: 'const PRESETS = { backend: {...}, web: {...}, mobile: {...} }\nconst applyPreset = (role) => Object.assign(form, PRESETS[role] ?? {})'
    }
  },
  // ================= 9. TAMPILAN & PERFORMA FRONTEND (FE) =================
  {
    id: 'FE-LARGE-COMPONENT',
    category: 'frontend',
    title: 'Halaman / Komponen UI Terlalu Besar (> 600 Baris)',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Satu berkas halaman/komponen (.vue, .tsx, .jsx, .svelte) maksimal 600 baris; di atas 1.000 baris dikategorikan kritis. Data statis besar (katalog, konstanta, mock) wajib dipisah ke modul tersendiri.',
    rationale: 'Komponen raksasa memperlambat hot-reload & kompilasi, membuat chunk halaman membengkak, memicu re-render luas saat satu state berubah, dan sangat sulit direview.',
    inspection_check: 'Menghitung jumlah baris setiap berkas UI. > 600 baris = WARNING, > 1.000 baris = CRITICAL.',
    suggested_action: 'Pecah per bagian layar (header, filter, tabel, modal) menjadi sub-komponen, pindahkan state & API ke composable/hook, dan data statis ke utils/.',
    example_code: {
      bad: '// pages/report.vue — 1.400 baris: template, 30 ref, 15 fungsi API, katalog data statis',
      good: '// pages/report.vue (< 300 baris) -> orkestrasi\n// components/report/ReportFilter.vue\n// components/report/ReportTable.vue\n// composables/useReport.ts'
    }
  },
  {
    id: 'FE-LARGE-MODULE',
    category: 'frontend',
    title: 'Modul Frontend (composable/store/hook) Terlalu Besar (> 800 Baris)',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Berkas TS/JS di components/, pages/, composables/, stores/, hooks/, layouts/ maksimal 800 baris dan dipecah per domain.',
    rationale: 'Modul gemuk ikut dimuat utuh di setiap halaman yang mengimpornya sehingga code splitting dan tree-shaking tidak efektif.',
    inspection_check: 'Menghitung baris berkas .ts/.js/.tsx/.jsx pada folder sumber frontend.',
    suggested_action: 'Pisahkan per domain (useProjects.ts, useDiagnostics.ts) dan re-export dari indeks bila perlu.',
    example_code: {
      bad: '// composables/useApi.ts — 1.500 baris berisi semua endpoint',
      good: '// composables/api/projects.ts\n// composables/api/diagnostics.ts\n// composables/useApi.ts -> re-export'
    }
  },
  {
    id: 'FE-HEAVY-LIB-IMPORT',
    category: 'frontend',
    title: 'Import Library Berat Secara Utuh (Bundle Bloat)',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Dilarang mengimpor seluruh library besar (import _ from "lodash", moment, import * as dari icon set / UI kit). Impor hanya fungsi/komponen yang dipakai.',
    rationale: 'Import utuh mematikan tree-shaking sehingga ratusan KB JavaScript ikut terkirim ke browser dan memperlambat waktu interaktif.',
    inspection_check: 'Pindai pola import default lodash/moment dan import * as dari lodash, lucide-react, @mui, date-fns, echarts, rxjs, antd.',
    suggested_action: 'Gunakan named import ES module (lodash-es), ganti moment dengan dayjs/date-fns, dan impor ikon satu per satu.',
    example_code: {
      bad: 'import _ from \'lodash\'\nimport moment from \'moment\'',
      good: 'import { debounce } from \'lodash-es\'\nimport dayjs from \'dayjs\''
    }
  },
  {
    id: 'FE-VFOR-MISSING-KEY',
    category: 'frontend',
    title: 'Rendering List v-for Tanpa :key',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Setiap elemen v-for wajib memiliki :key unik dan stabil (ID entitas, bukan index array).',
    rationale: 'Tanpa key, Vue tidak bisa melacak identitas item: DOM dirender ulang penuh, state input/animasi bisa tertukar antar baris.',
    inspection_check: 'Membaca tag pembuka yang memuat v-for (termasuk atribut multi-baris) dan memastikan ada :key / v-bind:key.',
    suggested_action: 'Tambahkan :key="item.id" pada elemen yang di-loop.',
    example_code: {
      bad: '<li v-for="user in users">{{ user.name }}</li>',
      good: '<li v-for="user in users" :key="user.id">{{ user.name }}</li>'
    }
  },
  {
    id: 'FE-VIF-WITH-VFOR',
    category: 'frontend',
    title: 'v-if dan v-for pada Elemen yang Sama',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Jangan menggabungkan v-if dan v-for pada elemen yang sama. Saring data melalui computed atau pindahkan v-if ke <template> pembungkus.',
    rationale: 'Kondisi dievaluasi untuk setiap item di setiap render sehingga boros, dan prioritas direktif (Vue 2 vs 3) membingungkan.',
    inspection_check: 'Mendeteksi tag yang memuat v-for sekaligus v-if.',
    suggested_action: 'Buat computed hasil filter lalu loop computed tersebut.',
    example_code: {
      bad: '<li v-for="u in users" v-if="u.active" :key="u.id">',
      good: 'const activeUsers = computed(() => users.value.filter(u => u.active))\n<li v-for="u in activeUsers" :key="u.id">'
    }
  },
  {
    id: 'FE-INLINE-BASE64-ASSET',
    category: 'frontend',
    title: 'Aset Base64 Besar Tertanam di Kode',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Data URI base64 di atas ±3KB dilarang ditanam di kode/CSS; simpan sebagai berkas terpisah.',
    rationale: 'Base64 menambah ±33% ukuran, tidak bisa di-cache terpisah, dan memperbesar bundle JS/CSS yang render-blocking.',
    inspection_check: 'Mendeteksi string data:image|font;base64 dengan panjang > 4.000 karakter.',
    suggested_action: 'Pindahkan ke public/ atau assets/ dan rujuk lewat URL.',
    example_code: {
      bad: '<img src="data:image/png;base64,iVBORw0KGgo...(50KB)">',
      good: '<img src="/images/hero.webp" width="1200" height="600" alt="Hero">'
    }
  },
  {
    id: 'FE-CONSOLE-LOG-LEFTOVER',
    category: 'frontend',
    title: 'Sisa console.log Debugging (≥ 3 per Berkas)',
    scope: 'Frontend (FE)',
    severity: 'info',
    rule: 'Kode aplikasi frontend tidak boleh membawa console.log/console.debug debugging ke produksi.',
    rationale: 'Log berlebih memperlambat render loop (serialisasi objek besar), membocorkan data internal ke DevTools, dan mengotori konsol.',
    inspection_check: 'Menghitung console.log/debug pada berkas UI & folder sumber frontend (skrip CLI diabaikan).',
    suggested_action: 'Hapus log atau bungkus dengan if (import.meta.dev).',
    example_code: {
      bad: 'console.log(\'data\', hugePayload)',
      good: 'if (import.meta.dev) console.debug(\'[report] loaded\', hugePayload.length)'
    }
  },
  {
    id: 'FE-IMG-LAYOUT-SHIFT',
    category: 'smelly_design',
    title: 'Gambar Tanpa Dimensi Eksplisit (Layout Shift)',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Setiap <img> wajib memiliki width & height, atau kelas ukuran/aspect-ratio (w-*, h-*, size-*, aspect-*) agar ruangnya dipesan sebelum gambar dimuat.',
    rationale: 'Gambar tanpa dimensi mendorong konten di bawahnya saat selesai dimuat (CLS buruk) sehingga pengguna salah klik.',
    inspection_check: 'Membaca setiap tag <img> di <template>/JSX dan mencari atribut width/height atau kelas ukuran.',
    suggested_action: 'Tambahkan width/height atau kelas aspect-video / aspect-square.',
    example_code: {
      bad: '<img :src="cover" class="w-full rounded-xl">',
      good: '<img :src="cover" class="w-full aspect-video object-cover rounded-xl" alt="Sampul">'
    }
  },
  {
    id: 'FE-IMG-MISSING-ALT',
    category: 'smelly_design',
    title: 'Gambar Tanpa Teks Alternatif (alt)',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Semua <img> wajib memiliki atribut alt; gunakan alt="" untuk gambar dekoratif.',
    rationale: 'Pembaca layar tidak dapat mendeskripsikan gambar dan SEO gambar hilang (WCAG 1.1.1 Level A).',
    inspection_check: 'Mendeteksi tag <img> tanpa alt / :alt.',
    suggested_action: 'Tambahkan alt deskriptif.',
    example_code: {
      bad: '<img src="/logo.svg">',
      good: '<img src="/logo.svg" alt="Logo Vibeasik" width="120" height="32">'
    }
  },
  {
    id: 'FE-IMG-NO-LAZY',
    category: 'frontend',
    title: 'Gambar Tanpa loading="lazy"',
    scope: 'Frontend (FE)',
    severity: 'info',
    rule: 'Gambar di bawah lipatan layar (galeri, kartu, daftar) wajib memakai loading="lazy" dan decoding="async".',
    rationale: 'Tanpa lazy loading, semua gambar diunduh saat halaman dibuka sehingga LCP dan kuota data pengguna terbuang.',
    inspection_check: 'Mendeteksi tag <img> tanpa atribut loading.',
    suggested_action: 'Tambahkan loading="lazy"; untuk gambar hero di atas lipatan gunakan fetchpriority="high".',
    example_code: {
      bad: '<img :src="shot.url" alt="Screenshot">',
      good: '<img :src="shot.url" alt="Screenshot" loading="lazy" decoding="async" width="320" height="200">'
    }
  },
  {
    id: 'FE-CLICKABLE-NON-INTERACTIVE',
    category: 'smelly_design',
    title: 'div/span Diberi @click Tanpa role & tabindex',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Elemen yang dapat diklik wajib berupa <button>/<a>, atau minimal memiliki role="button", tabindex="0", dan handler keyboard.',
    rationale: 'div/span yang diklik tidak bisa difokus dengan Tab dan tidak dikenali pembaca layar (WCAG 2.1.1 & 4.1.2).',
    inspection_check: 'Mendeteksi <div>/<span>/<li>/<td> yang memiliki @click/onClick tanpa role dan tabindex.',
    suggested_action: 'Ganti dengan <button type="button"> atau tambahkan role, tabindex, dan @keydown.enter.',
    example_code: {
      bad: '<div class="card cursor-pointer" @click="open(item)">',
      good: '<button type="button" class="card text-left" @click="open(item)">'
    }
  },
  {
    id: 'FE-SMALL-FONT',
    category: 'smelly_design',
    title: 'Ukuran Teks Terlalu Kecil (< 11px)',
    scope: 'Frontend (FE)',
    severity: 'info',
    rule: 'Ukuran teks minimal 11–12px untuk label/caption dan 14–16px untuk isi utama. Hindari text-[9px]/text-[10px].',
    rationale: 'Teks terlalu kecil sulit dibaca di layar HP dan oleh pengguna dengan gangguan penglihatan; terlihat tidak profesional.',
    inspection_check: 'Mendeteksi kelas text-[Npx] atau font-size: Npx dengan N < 11.',
    suggested_action: 'Gunakan skala tipografi tema (text-xs, text-sm, text-base).',
    example_code: {
      bad: '<span class="text-[9px]">Status</span>',
      good: '<span class="text-xs">Status</span>'
    }
  },
  {
    id: 'FE-FIXED-WIDTH-LAYOUT',
    category: 'smelly_design',
    title: 'Lebar Layout Tetap ≥ 768px (Tidak Responsif)',
    scope: 'Frontend (FE)',
    severity: 'warning',
    rule: 'Dilarang menetapkan width/min-width tetap ≥ 768px tanpa prefix breakpoint. Gunakan w-full + max-w-*, grid/flex fluid.',
    rationale: 'Lebar tetap memunculkan scroll horizontal dan konten terpotong di tablet/HP (WCAG 1.4.10 Reflow).',
    inspection_check: 'Mendeteksi width/min-width: ≥768px di CSS dan w-[≥768px] Tailwind tanpa prefix md:/lg:.',
    suggested_action: 'Ganti dengan w-full max-w-5xl mx-auto, atau terapkan hanya di lg:w-[...].',
    example_code: {
      bad: '<div class="w-[1200px]">',
      good: '<div class="w-full max-w-7xl mx-auto px-4">'
    }
  },
  {
    id: 'FE-INLINE-STYLE-OVERUSE',
    category: 'smelly_design',
    title: 'Inline Style Berlebihan (≥ 8 per Berkas)',
    scope: 'Frontend (FE)',
    severity: 'info',
    rule: 'Gaya visual wajib melalui kelas utility/design system, bukan atribut style="..." statis.',
    rationale: 'Inline style mem-bypass tema, tidak mendukung dark mode/breakpoint, dan membuat tampilan antar halaman tidak konsisten.',
    inspection_check: 'Menghitung atribut style="..." statis (bukan :style dinamis) di template.',
    suggested_action: 'Pindahkan ke kelas Tailwind atau <style scoped> dengan token.',
    example_code: {
      bad: '<div style="margin-top: 12px; color: #333">',
      good: '<div class="mt-3 text-neutral-700 dark:text-neutral-300">'
    }
  },
  {
    id: 'FE-HARDCODED-COLORS',
    category: 'smelly_design',
    title: 'Warna Hex Hardcoded Berulang (≥ 12 per Berkas)',
    scope: 'Frontend (FE)',
    severity: 'info',
    rule: 'Warna wajib berasal dari design token/tema (Tailwind config, app.config, CSS variables), bukan kode hex tersebar di template/style.',
    rationale: 'Hex tersebar membuat palet tidak konsisten, rebranding sulit, dan dark mode rusak.',
    inspection_check: 'Menghitung kode warna hex di <template>/<style> (fallback var(--x, #hex) diabaikan).',
    suggested_action: 'Definisikan token warna lalu gunakan kelas semantik.',
    example_code: {
      bad: '.title { color: #7c3aed } .badge { background: #ede9fe }',
      good: ':root { --color-brand: #7c3aed }\n.title { color: var(--color-brand) }'
    }
  },
  {
    id: 'FE-ZINDEX-ESCALATION',
    category: 'smelly_design',
    title: 'Eskalasi z-index Ekstrem (≥ 1000)',
    scope: 'Frontend (FE)',
    severity: 'info',
    rule: 'Gunakan skala z-index terpusat (dropdown 10, sticky 20, modal 50, toast 60). Nilai ≥ 1000 dianggap smell.',
    rationale: 'Perang z-index membuat modal, dropdown, dan toast saling menutupi secara acak.',
    inspection_check: 'Mendeteksi z-index: ≥1000 atau z-[≥1000].',
    suggested_action: 'Definisikan token layer dan gunakan konsisten.',
    example_code: {
      bad: '.popup { z-index: 99999 }',
      good: '.popup { z-index: var(--z-modal) } /* 50 */'
    }
  },
  {
    id: 'FE-IMPORTANT-OVERUSE',
    category: 'smelly_design',
    title: 'Penggunaan !important Berlebihan (≥ 5 per Berkas)',
    scope: 'Frontend (FE)',
    severity: 'info',
    rule: '!important hanya untuk utility override yang disengaja; perbaiki spesifisitas selector.',
    rationale: 'Cascade CSS menjadi kacau sehingga tema/override komponen sulit dilakukan.',
    inspection_check: 'Menghitung kemunculan !important per berkas.',
    suggested_action: 'Gunakan selector yang lebih tepat atau utility class.',
    example_code: {
      bad: '.btn { color: red !important; padding: 0 !important }',
      good: '.card .btn-danger { color: var(--color-danger) }'
    }
  },

  // ================= 10. ANDROID NATIVE (KOTLIN / JAVA / XML LAYOUT) =================
  {
    id: 'AND-GOD-ACTIVITY',
    category: 'mobile',
    title: 'Activity / Fragment / Screen Terlalu Besar (> 600 Baris)',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Activity, Fragment, Adapter, ViewModel, atau Screen Compose maksimal 600 baris (kritis > 1.000). UI hanya merender state; logika di ViewModel/UseCase.',
    rationale: 'God Activity sulit diuji, memperlambat build incremental, dan sering menahan referensi berlebihan yang memicu memory leak.',
    inspection_check: 'Menghitung baris berkas .kt/.java bernama *Activity, *Fragment, *Screen, *Adapter, *ViewModel.',
    suggested_action: 'Ekstrak logika ke ViewModel + Repository, pecah UI ke Composable/Fragment kecil.',
    example_code: {
      bad: '// MainActivity.kt — 1.800 baris: networking, DB, adapter, dialog',
      good: '// MainActivity.kt -> setContent { MainScreen(vm) }\n// MainViewModel.kt, OrderRepository.kt, OrderList.kt'
    }
  },
  {
    id: 'AND-MAIN-THREAD-BLOCKING',
    category: 'mobile',
    title: 'Operasi Blocking di Main Thread (ANR)',
    scope: 'Mobile',
    severity: 'critical',
    rule: 'Dilarang runBlocking / Thread.sleep di Activity/Fragment/Composable, dan dilarang allowMainThreadQueries() pada Room.',
    rationale: 'Main thread yang tertahan > 5 detik memunculkan dialog ANR; > 16ms per frame sudah menyebabkan jank.',
    inspection_check: 'Mendeteksi runBlocking{, Thread.sleep(, .allowMainThreadQueries() pada layer UI.',
    suggested_action: 'Gunakan viewModelScope/lifecycleScope + withContext(Dispatchers.IO).',
    example_code: {
      bad: 'val user = runBlocking { api.getUser() }',
      good: 'viewModelScope.launch { val user = withContext(Dispatchers.IO) { api.getUser() } }'
    }
  },
  {
    id: 'AND-GLOBALSCOPE-LEAK',
    category: 'mobile',
    title: 'Coroutine di GlobalScope',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Dilarang GlobalScope.launch/async; gunakan scope yang terikat lifecycle.',
    rationale: 'Coroutine terus berjalan setelah layar ditutup, menahan referensi Activity, dan menguras baterai.',
    inspection_check: 'Mendeteksi GlobalScope.launch / GlobalScope.async.',
    suggested_action: 'Ganti dengan viewModelScope / lifecycleScope.',
    example_code: {
      bad: 'GlobalScope.launch { sync() }',
      good: 'lifecycleScope.launch { sync() }'
    }
  },
  {
    id: 'AND-STATIC-CONTEXT-LEAK',
    category: 'mobile',
    title: 'Context / Activity / View di Field Statis',
    scope: 'Mobile',
    severity: 'critical',
    rule: 'Dilarang menyimpan Context, Activity, Fragment, atau View di field static (Java) atau companion object/object (Kotlin).',
    rationale: 'Activity tidak pernah di-garbage-collect setelah rotasi/navigasi sehingga memori terus naik hingga OOM.',
    inspection_check: 'Mendeteksi static Context/Activity/View (Java) dan var bertipe Context/Activity/View di dalam companion object.',
    suggested_action: 'Gunakan applicationContext via DI (Hilt) atau WeakReference.',
    example_code: {
      bad: 'companion object { lateinit var activity: Activity }',
      good: 'class Repo @Inject constructor(@ApplicationContext private val ctx: Context)'
    }
  },
  {
    id: 'AND-NOTIFY-DATASET-CHANGED',
    category: 'mobile',
    title: 'RecyclerView notifyDataSetChanged()',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Pembaruan list wajib memakai ListAdapter + DiffUtil atau notifyItem* spesifik.',
    rationale: 'notifyDataSetChanged mem-bind ulang seluruh item tanpa animasi sehingga scroll patah pada list panjang.',
    inspection_check: 'Mendeteksi pemanggilan .notifyDataSetChanged().',
    suggested_action: 'Migrasi ke ListAdapter dan submitList().',
    example_code: {
      bad: 'items = newItems; adapter.notifyDataSetChanged()',
      good: 'adapter.submitList(newItems) // ListAdapter + DiffUtil'
    }
  },
  {
    id: 'AND-COMPOSE-STATE-NO-REMEMBER',
    category: 'mobile',
    title: 'Compose: mutableStateOf Tanpa remember',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'State di dalam @Composable wajib dibungkus remember/rememberSaveable atau diangkat ke ViewModel.',
    rationale: 'State dibuat ulang setiap recomposition sehingga input hilang dan UI berkedip.',
    inspection_check: 'Mendeteksi mutableStateOf/ListOf/MapOf tanpa remember pada berkas yang berisi @Composable (ViewModel dikecualikan).',
    suggested_action: 'var text by rememberSaveable { mutableStateOf("") }',
    example_code: {
      bad: 'var query = mutableStateOf("")',
      good: 'var query by rememberSaveable { mutableStateOf("") }'
    }
  },
  {
    id: 'AND-COMPOSE-LAZY-NO-KEY',
    category: 'mobile',
    title: 'Compose: LazyColumn items() Tanpa key',
    scope: 'Mobile',
    severity: 'info',
    rule: 'items() pada LazyColumn/LazyRow/LazyGrid wajib menyertakan key stabil.',
    rationale: 'Tanpa key, perubahan urutan membuat Compose me-recompose item yang salah dan state per item tertukar.',
    inspection_check: 'Mendeteksi items( tanpa key = pada berkas yang memakai Lazy list.',
    suggested_action: 'items(users, key = { it.id }) { ... }',
    example_code: {
      bad: 'LazyColumn { items(users) { UserRow(it) } }',
      good: 'LazyColumn { items(users, key = { it.id }) { UserRow(it) } }'
    }
  },
  {
    id: 'AND-BITMAP-DECODE-UI',
    category: 'mobile',
    title: 'Decode Bitmap Manual di Layer UI',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Gambar dimuat lewat Coil/Glide (downsampling + cache), bukan BitmapFactory.decode* di Activity/Fragment/Composable.',
    rationale: 'Decode resolusi penuh di main thread memicu OutOfMemoryError dan frame drop.',
    inspection_check: 'Mendeteksi BitmapFactory.decodeFile/Resource/Stream/ByteArray pada layer UI.',
    suggested_action: 'Gunakan AsyncImage (Coil) atau Glide.with(view).load(url).',
    example_code: {
      bad: 'imageView.setImageBitmap(BitmapFactory.decodeFile(path))',
      good: 'imageView.load(file) // Coil, otomatis downsample & cache'
    }
  },
  {
    id: 'AND-DEEP-LAYOUT',
    category: 'mobile',
    title: 'Hierarki Layout XML Terlalu Dalam (> 10 Level)',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Kedalaman view pada layout XML maksimal 10 level (Android Lint TooDeepLayout).',
    rationale: 'Setiap level menambah pass measure/layout; nested LinearLayout + weight dapat mengukur ulang anak berkali-kali sehingga frame > 16ms.',
    inspection_check: 'Menghitung kedalaman elemen terdalam pada berkas res/layout*/*.xml.',
    suggested_action: 'Ratakan dengan ConstraintLayout, gunakan <merge>/<include>/ViewStub.',
    example_code: {
      bad: '<LinearLayout><LinearLayout><LinearLayout>... (12 level)',
      good: '<ConstraintLayout> <!-- view sejajar dengan constraint --> </ConstraintLayout>'
    }
  },
  {
    id: 'AND-LARGE-LAYOUT',
    category: 'mobile',
    title: 'Layout XML Terlalu Besar (> 300 Baris)',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Satu berkas layout maksimal 300 baris; bagian berulang/opsional dipecah ke <include> atau ViewStub.',
    rationale: 'Inflate layout besar memperlambat pembukaan layar dan menyulitkan perawatan.',
    inspection_check: 'Menghitung baris berkas res/layout*/*.xml.',
    suggested_action: 'Pecah per section dengan <include layout="@layout/..."/>.',
    example_code: {
      bad: '<!-- activity_checkout.xml 900 baris -->',
      good: '<include layout="@layout/section_address" />\n<ViewStub android:layout="@layout/section_promo" ... />'
    }
  },
  {
    id: 'AND-TEXTSIZE-NOT-SP',
    category: 'mobile',
    title: 'textSize Memakai dp/px, Bukan sp',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Ukuran teks wajib dalam sp (atau TextAppearance tema) agar mengikuti pengaturan ukuran font sistem.',
    rationale: 'Teks dp/px tidak membesar untuk pengguna low-vision — pelanggaran aksesibilitas.',
    inspection_check: 'Mendeteksi android:textSize="Ndp|px".',
    suggested_action: 'Gunakan android:textSize="14sp" atau textAppearance.',
    example_code: {
      bad: 'android:textSize="14dp"',
      good: 'android:textAppearance="?attr/textAppearanceBodyMedium"'
    }
  },
  {
    id: 'AND-SMALL-TOUCH-TARGET',
    category: 'mobile',
    title: 'Area Sentuh Tombol < 48dp',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Button, ImageButton, CheckBox, Switch, dan view clickable minimal 48×48dp (gunakan minWidth/minHeight atau padding).',
    rationale: 'Tombol kecil sulit ditekan dan sering salah sentuh (Material & WCAG 2.5.8).',
    inspection_check: 'Membaca tag kontrol interaktif dan mendeteksi layout_width/height numerik < 48dp tanpa minWidth/minHeight.',
    suggested_action: 'Tambahkan android:minWidth="48dp" android:minHeight="48dp".',
    example_code: {
      bad: '<ImageButton android:layout_width="24dp" android:layout_height="24dp" />',
      good: '<ImageButton android:layout_width="48dp" android:layout_height="48dp" android:padding="12dp" />'
    }
  },
  {
    id: 'AND-IMAGE-NO-CONTENTDESC',
    category: 'mobile',
    title: 'ImageView / ImageButton Tanpa contentDescription',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Gambar/ikon bermakna wajib memiliki contentDescription; gambar dekoratif diberi importantForAccessibility="no".',
    rationale: 'TalkBack tidak dapat membacakan fungsi tombol ikon.',
    inspection_check: 'Mendeteksi ImageView/ImageButton tanpa contentDescription.',
    suggested_action: 'android:contentDescription="@string/cd_close".',
    example_code: {
      bad: '<ImageButton android:src="@drawable/ic_close" />',
      good: '<ImageButton android:src="@drawable/ic_close" android:contentDescription="@string/cd_close" />'
    }
  },
  {
    id: 'AND-HARDCODED-TEXT',
    category: 'mobile',
    title: 'Teks Hardcoded di Layout',
    scope: 'Mobile',
    severity: 'info',
    rule: 'android:text / android:hint wajib merujuk @string agar dapat dilokalisasi.',
    rationale: 'Teks literal tidak bisa diterjemahkan dan sulit dikelola konsisten.',
    inspection_check: 'Mendeteksi android:text/hint berisi literal (bukan @string / ?attr).',
    suggested_action: 'Pindahkan ke res/values/strings.xml.',
    example_code: {
      bad: 'android:text="Masuk"',
      good: 'android:text="@string/login"'
    }
  },
  {
    id: 'AND-HARDCODED-COLOR-XML',
    category: 'mobile',
    title: 'Warna Hex Hardcoded di Layout (≥ 5)',
    scope: 'Mobile',
    severity: 'info',
    rule: 'Warna pada layout wajib merujuk @color atau atribut tema (?attr/colorPrimary).',
    rationale: 'Hex langsung merusak dark theme dan konsistensi brand.',
    inspection_check: 'Menghitung textColor/background/tint/src berisi #hex.',
    suggested_action: 'Definisikan di colors.xml / tema Material.',
    example_code: {
      bad: 'android:textColor="#212121"',
      good: 'android:textColor="?attr/colorOnSurface"'
    }
  },
  {
    id: 'AND-LEGACY-LISTVIEW',
    category: 'mobile',
    title: 'ListView / GridView Lama',
    scope: 'Mobile',
    severity: 'info',
    rule: 'Daftar data memakai RecyclerView (atau LazyColumn di Compose), bukan ListView/GridView.',
    rationale: 'ListView tidak mendaur ulang view seefisien RecyclerView dan tidak mendukung DiffUtil.',
    inspection_check: 'Mendeteksi tag <ListView> / <GridView> di layout.',
    suggested_action: 'Migrasi ke RecyclerView + ListAdapter.',
    example_code: {
      bad: '<ListView android:id="@+id/list" />',
      good: '<androidx.recyclerview.widget.RecyclerView android:id="@+id/list" />'
    }
  },

  // ================= 11. FLUTTER (hanya aplikasi Flutter: import package:flutter, BuildContext, atau pubspec sdk: flutter) =================
  {
    id: 'MOB-LARGE-WIDGET-FILE',
    category: 'mobile',
    title: 'Screen / Widget Flutter Terlalu Besar (> 600 Baris)',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Berkas screen/page/widget Flutter maksimal 600 baris; sub-tree diekstrak menjadi StatelessWidget terpisah (bukan method _buildX).',
    rationale: 'Widget besar me-rebuild seluruh sub-tree saat setState dan sulit dirawat.',
    inspection_check: 'Menghitung baris berkas .dart di screens/, pages/, widgets/, views/ atau *_screen.dart/*_page.dart/*_widget.dart.',
    suggested_action: 'Ekstrak widget const kecil dan pindahkan state ke Bloc/Riverpod.',
    example_code: {
      bad: 'Widget _buildHeader() { ... } // method helper di dalam State 1.200 baris',
      good: 'class OrderHeader extends StatelessWidget { const OrderHeader({super.key}); ... }'
    }
  },
  {
    id: 'MOB-LISTVIEW-NON-BUILDER',
    category: 'mobile',
    title: 'Flutter ListView/GridView Tanpa .builder',
    scope: 'Mobile',
    severity: 'info',
    rule: 'Daftar dinamis/panjang wajib memakai ListView.builder / GridView.builder / ListView.separated.',
    rationale: 'Konstruktor children membangun semua item sekaligus meski tidak terlihat.',
    inspection_check: 'Mendeteksi ListView( / GridView( tanpa named constructor.',
    suggested_action: 'ListView.builder(itemCount: n, itemBuilder: ...)',
    example_code: {
      bad: 'ListView(children: items.map((e) => Tile(e)).toList())',
      good: 'ListView.builder(itemCount: items.length, itemBuilder: (_, i) => Tile(items[i]))'
    }
  },
  {
    id: 'MOB-SHRINKWRAP-LIST',
    category: 'mobile',
    title: 'Flutter shrinkWrap: true pada Scrollable',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Hindari shrinkWrap: true; gunakan CustomScrollView + Sliver untuk menggabungkan beberapa daftar.',
    rationale: 'shrinkWrap mengukur semua anak di awal sehingga lazy loading hilang dan scroll berat.',
    inspection_check: 'Mendeteksi shrinkWrap: true.',
    suggested_action: 'CustomScrollView(slivers: [...SliverList.builder...]).',
    example_code: {
      bad: 'Column(children: [Header(), ListView(shrinkWrap: true, children: ...)])',
      good: 'CustomScrollView(slivers: [SliverToBoxAdapter(child: Header()), SliverList.builder(...)])'
    }
  },
  {
    id: 'MOB-IMAGE-NETWORK-NO-CACHE',
    category: 'mobile',
    title: 'Flutter Image.network Tanpa Cache/Resize',
    scope: 'Mobile',
    severity: 'info',
    rule: 'Gambar jaringan memakai CachedNetworkImage atau cacheWidth/cacheHeight sesuai ukuran tampil.',
    rationale: 'Gambar resolusi penuh di-decode ke memori dan diunduh ulang setiap kali layar dibuka.',
    inspection_check: 'Mendeteksi Image.network pada berkas tanpa CachedNetworkImage/cacheWidth.',
    suggested_action: 'CachedNetworkImage(imageUrl: url, memCacheWidth: 300).',
    example_code: {
      bad: 'Image.network(url)',
      good: 'CachedNetworkImage(imageUrl: url, memCacheWidth: 300)'
    }
  },
  {
    id: 'MOB-OPACITY-WIDGET',
    category: 'mobile',
    title: 'Flutter Widget Opacity (saveLayer Mahal)',
    scope: 'Mobile',
    severity: 'info',
    rule: 'Gunakan AnimatedOpacity/FadeTransition untuk animasi dan warna ber-alpha untuk elemen statis, bukan widget Opacity.',
    rationale: 'Opacity memicu saveLayer offscreen yang mahal, terutama bila dianimasikan.',
    inspection_check: 'Mendeteksi pemanggilan widget Opacity(.',
    suggested_action: 'FadeTransition(opacity: animation, child: child).',
    example_code: {
      bad: 'Opacity(opacity: _value, child: card)',
      good: 'FadeTransition(opacity: _animation, child: card)'
    }
  },

  {
    id: 'MOB-FUTURE-IN-BUILD',
    category: 'mobile',
    title: 'Flutter FutureBuilder/StreamBuilder Membuat Future di build()',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Future/Stream untuk FutureBuilder/StreamBuilder wajib diperoleh sebelumnya (initState, late final, provider), bukan dipanggil langsung di argumen future:/stream:.',
    rationale: 'build() dapat dipanggil setiap frame; future inline memicu request ulang pada setiap rebuild sehingga loading berkedip dan beban jaringan berlipat.',
    inspection_check: 'Mendeteksi future: / stream: berisi pemanggilan fungsi (selain Future.value/error) di dalam build() pada berkas yang memakai FutureBuilder/StreamBuilder.',
    suggested_action: 'Simpan Future di field State lalu oper field tersebut.',
    example_code: {
      bad: 'FutureBuilder(future: repo.fetchItems(), builder: ...)',
      good: 'late final Future<List<Item>> _items = repo.fetchItems();\n...\nFutureBuilder(future: _items, builder: ...)'
    }
  },
  {
    id: 'MOB-CONTROLLER-IN-BUILD',
    category: 'mobile',
    title: 'Flutter Controller/FocusNode Dibuat di build()',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Controller, FocusNode, ValueNotifier, dan StreamController tidak boleh dibuat di tubuh build(); jadikan field State yang dibuat sekali dan di-dispose.',
    rationale: 'Objek dibuat ulang setiap rebuild: teks input/posisi scroll hilang dan instance lama tidak pernah di-dispose.',
    inspection_check: 'Mendeteksi konstruktor TextEditingController/ScrollController/.../FocusNode yang dieksekusi langsung di build() (di luar closure).',
    suggested_action: 'Pindahkan ke field State (late final) dan lepaskan di dispose().',
    example_code: {
      bad: 'Widget build(BuildContext context) {\n  final ctrl = TextEditingController();\n  return TextField(controller: ctrl);\n}',
      good: 'final _ctrl = TextEditingController();\n@override\nvoid dispose() { _ctrl.dispose(); super.dispose(); }'
    }
  },
  {
    id: 'MOB-ASYNC-GAP-CONTEXT',
    category: 'mobile',
    title: 'Flutter BuildContext/setState Setelah await Tanpa Cek mounted',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Setelah await di fungsi/closure async, periksa if (!mounted) return; (State) atau if (!context.mounted) return; sebelum memakai context atau memanggil setState.',
    rationale: 'Layar bisa sudah ditutup saat future selesai: setState melempar "setState() called after dispose()" dan context yang tidak valid membuat Navigator/SnackBar gagal (lint use_build_context_synchronously).',
    inspection_check: 'Per badan async (termasuk closure onPressed: () async {...}): penggunaan context / setState setelah ekspresi await selesai tanpa kemunculan mounted di antaranya. Argumen dari pemanggilan yang di-await sendiri (await Navigator.push(context, ...)) tidak dihitung.',
    suggested_action: 'Tambahkan cek mounted, atau ambil Navigator/ScaffoldMessenger sebelum await.',
    example_code: {
      bad: 'await repo.save(item);\nNavigator.of(context).pop();',
      good: 'await repo.save(item);\nif (!context.mounted) return;\nNavigator.of(context).pop();'
    }
  },
  {
    id: 'MOB-PRINT-STATEMENT',
    category: 'code_smell',
    title: 'Flutter print() di Kode Aplikasi',
    scope: 'Mobile',
    severity: 'info',
    rule: 'Kode aplikasi (lib/) tidak memakai print(); gunakan debugPrint, dart:developer log, atau package logger, dan bungkus log diagnostik dengan kDebugMode.',
    rationale: 'print ikut ke build rilis dan dapat membocorkan data ke log perangkat (lint avoid_print dari flutter_lints).',
    inspection_check: 'Mendeteksi print( di luar test/, tool/, integration_test/ (debugPrint dan baris ber-kDebugMode diabaikan).',
    suggested_action: 'dev.log(\'pesan\', name: \'Modul\') atau debugPrint().',
    example_code: {
      bad: 'print(\'token: $token\');',
      good: 'if (kDebugMode) debugPrint(\'login sukses\');'
    }
  },
  {
    id: 'MOB-NO-FLUTTER-LINTS',
    category: 'mobile',
    title: 'Flutter Tanpa Set Lint Resmi (flutter_lints)',
    scope: 'Mobile',
    severity: 'warning',
    rule: 'Setiap paket Flutter memiliki analysis_options.yaml yang meng-include set lint resmi (flutter_lints, lints, very_good_analysis, dsb.) dan flutter analyze dijalankan di CI.',
    rationale: 'Tanpa lint, analyzer tidak memperingatkan use_build_context_synchronously, avoid_print, prefer_const_constructors, dan aturan Effective Dart lain yang mencegah bug lifecycle & rebuild berlebih.',
    inspection_check: 'Untuk setiap pubspec.yaml yang bergantung pada SDK Flutter: analysis_options.yaml di folder yang sama harus ada dan berisi include: package:<set lint>/....',
    suggested_action: 'flutter pub add --dev flutter_lints, lalu include: package:flutter_lints/flutter.yaml.',
    example_code: {
      bad: '# analysis_options.yaml tidak ada / kosong',
      good: 'include: package:flutter_lints/flutter.yaml'
    }
  },

  // ================= 12. ASET MEDIA =================
  {
    id: 'ASSET-OVERSIZED',
    category: 'frontend',
    title: 'Aset Gambar / Video Terlalu Berat',
    scope: 'Universal',
    severity: 'warning',
    rule: 'Batas ukuran aset: gambar web ≤ 500KB (kritis ≥ 1,5MB), drawable Android ≤ 200KB, SVG ≤ 200KB, video ≤ 5MB.',
    rationale: 'Aset berat memperlambat LCP, memboroskan kuota pengguna mobile, dan memperbesar ukuran build/APK.',
    inspection_check: 'Mengukur ukuran berkas .png/.jpg/.gif/.webp/.avif/.svg/.mp4/.webm/.mov di folder sumber (tanpa membaca isi).',
    suggested_action: 'Kompres & konversi ke WebP/AVIF, ganti GIF dengan MP4/WebM, optimasi SVG dengan SVGO, gunakan VectorDrawable untuk ikon Android.',
    example_code: {
      bad: 'public/images/hero.png (3,2 MB)',
      good: 'public/images/hero.webp (180 KB) + srcset 640/1280/1920'
    }
  },

  // ================= 13. DEEP CHECK: TAMPILAN, BUNDLE & BUILD =================
  {
    id: 'DEEP-FE-LARGE-CHUNK',
    category: 'frontend',
    title: 'Chunk JavaScript Hasil Build > 500KB',
    scope: 'Frontend (FE)',
    severity: 'warning',
    is_deep: true,
    rule: 'Setiap chunk JS hasil build maksimal 500KB (belum dikompresi); ≥ 1,5MB kritis.',
    rationale: 'Browser harus mengunduh, mem-parse, dan mengeksekusi seluruh chunk sebelum halaman interaktif (INP/TTI buruk di HP menengah).',
    inspection_check: 'Hanya diuji saat Deep Check: mengukur berkas JS di .output/public/_nuxt, dist/assets, build/static/js, .next/static/chunks (5 terbesar dilaporkan).',
    suggested_action: 'Dynamic import / defineAsyncComponent untuk komponen berat, analisis dengan vite-bundle-visualizer.',
    example_code: {
      bad: 'import MonacoEditor from \'~/components/MonacoEditor.vue\'',
      good: 'const MonacoEditor = defineAsyncComponent(() => import(\'~/components/MonacoEditor.vue\'))'
    }
  },
  {
    id: 'DEEP-FE-BUNDLE-BUDGET',
    category: 'frontend',
    title: 'Total Bundle JavaScript > 3MB',
    scope: 'Frontend (FE)',
    severity: 'warning',
    is_deep: true,
    rule: 'Total JS hasil build maksimal 3MB; tetapkan performance budget di CI.',
    rationale: 'Bundle besar memperlambat pemuatan di jaringan seluler dan perangkat low-end.',
    inspection_check: 'Hanya diuji saat Deep Check: menjumlahkan ukuran seluruh JS pada folder hasil build.',
    suggested_action: 'Code-splitting per rute, audit dependensi, hapus polyfill/locale tak perlu.',
    example_code: {
      bad: '// 1 vendor.js berisi semua library (4,8MB)',
      good: 'build: { chunkSizeWarningLimit: 500, rollupOptions: { output: { manualChunks: { charts: [\'echarts\'] } } } }'
    }
  },
  {
    id: 'DEEP-FE-CSS-BLOAT',
    category: 'frontend',
    title: 'Total CSS Hasil Build > 300KB',
    scope: 'Frontend (FE)',
    severity: 'info',
    is_deep: true,
    rule: 'Total CSS hasil build maksimal 300KB.',
    rationale: 'CSS bersifat render-blocking sehingga First Contentful Paint tertunda.',
    inspection_check: 'Hanya diuji saat Deep Check: menjumlahkan ukuran CSS pada folder hasil build.',
    suggested_action: 'Pastikan purge Tailwind aktif dan hapus library CSS yang tidak dipakai.',
    example_code: {
      bad: '@import \'bootstrap/dist/css/bootstrap.css\'; /* hanya pakai grid */',
      good: '/* gunakan utility Tailwind yang di-purge otomatis */'
    }
  },
  {
    id: 'DEEP-FE-ROUTE-NOT-LAZY',
    category: 'frontend',
    title: 'Halaman Diimpor Statis di Router (Tanpa Lazy Loading)',
    scope: 'Frontend (FE)',
    severity: 'warning',
    is_deep: true,
    rule: 'Router SPA (Vue Router / React Router) wajib memuat halaman secara lazy (() => import(...) / React.lazy).',
    rationale: 'Import statis memasukkan semua halaman ke bundle awal sehingga halaman pertama lambat terbuka.',
    inspection_check: 'Hanya diuji saat Deep Check: berkas router/routes/App dengan ≥ 6 import statis dari pages/views/screens tanpa dynamic import.',
    suggested_action: '{ path: \'/reports\', component: () => import(\'@/pages/Reports.vue\') }',
    example_code: {
      bad: 'import Reports from \'@/pages/Reports.vue\'\n{ path: \'/reports\', component: Reports }',
      good: '{ path: \'/reports\', component: () => import(\'@/pages/Reports.vue\') }'
    }
  },
  {
    id: 'DEEP-UI-COMPONENT-HOTSPOT',
    category: 'frontend',
    title: 'Sebaran Komponen Gemuk (≥ 30% Berkas UI > 400 Baris)',
    scope: 'Universal',
    severity: 'warning',
    is_deep: true,
    rule: 'Maksimal 30% berkas UI (FE, Activity/Fragment/Screen, Flutter screen) boleh melebihi 400 baris.',
    rationale: 'Jika banyak komponen gemuk, masalahnya struktural: komponen tidak dipecah, re-render luas, dan code splitting tidak efektif.',
    inspection_check: 'Hanya diuji saat Deep Check: menghitung rasio berkas UI > 400 baris dan menampilkan 5 terbesar.',
    suggested_action: 'Pasang lint max-lines dan refactor mulai dari berkas terbesar.',
    example_code: {
      bad: '// 18 dari 40 komponen > 400 baris',
      good: '"max-lines": ["warn", { "max": 400, "skipBlankLines": true }]'
    }
  },
  {
    id: 'DEEP-ASSET-WEIGHT',
    category: 'frontend',
    title: 'Total Aset Media Sumber > 10MB',
    scope: 'Universal',
    severity: 'warning',
    is_deep: true,
    rule: 'Total gambar/video di kode sumber satu komponen maksimal 10MB; media besar dipindah ke CDN/object storage.',
    rationale: 'Aset ikut dibundel/didistribusikan, memperbesar waktu muat dan ukuran build/APK.',
    inspection_check: 'Hanya diuji saat Deep Check: menjumlahkan ukuran semua aset media di folder sumber.',
    suggested_action: 'Kompres massal, pindahkan video/galeri ke CDN, muat secara lazy.',
    example_code: {
      bad: 'public/videos/*.mp4 (45MB) ikut di repo & build',
      good: '<video src="https://cdn.example.com/intro-720p.mp4" preload="none" poster="/intro.webp">'
    }
  },
  {
    id: 'DEEP-ASSET-NO-MODERN-FORMAT',
    category: 'frontend',
    title: 'Belum Ada Gambar Format Modern (WebP/AVIF)',
    scope: 'Universal',
    severity: 'info',
    is_deep: true,
    rule: 'Proyek dengan ≥ 10 gambar PNG/JPG > 50KB wajib mulai menyajikan WebP/AVIF.',
    rationale: 'WebP/AVIF 25–50% lebih kecil dengan kualitas setara.',
    inspection_check: 'Hanya diuji saat Deep Check: menghitung PNG/JPG > 50KB dan keberadaan .webp/.avif.',
    suggested_action: 'Konversi aset dan gunakan <picture> / NuxtImg.',
    example_code: {
      bad: '<img src="/hero.png">',
      good: '<picture><source srcset="/hero.avif" type="image/avif"><img src="/hero.webp" alt="..."></picture>'
    }
  },
  {
    id: 'DEEP-AND-APK-SIZE',
    category: 'mobile',
    title: 'Ukuran APK/AAB > 50MB',
    scope: 'Mobile',
    severity: 'warning',
    is_deep: true,
    rule: 'Artefak rilis Android maksimal 50MB (kritis ≥ 100MB).',
    rationale: 'Ukuran besar menurunkan conversion instalasi dan aplikasi cenderung dihapus di perangkat berpenyimpanan kecil.',
    inspection_check: 'Hanya diuji saat Deep Check: mengukur .apk/.aab di build/outputs (termasuk Flutter build/app/outputs).',
    suggested_action: 'Rilis AAB, aktifkan R8 + shrinkResources, konversi gambar ke WebP/vector, split ABI.',
    example_code: {
      bad: 'app-release.apk (128MB) — semua ABI & PNG resolusi penuh',
      good: 'app-release.aab + isMinifyEnabled + isShrinkResources + WebP'
    }
  },
  {
    id: 'DEEP-AND-RELEASE-NOT-OPTIMIZED',
    category: 'mobile',
    title: 'Build Release Android Tanpa R8 / Shrink Resources',
    scope: 'Mobile',
    severity: 'warning',
    is_deep: true,
    rule: 'buildTypes.release wajib mengaktifkan isMinifyEnabled dan isShrinkResources.',
    rationale: 'Tanpa R8, APK membawa kode & resource tak terpakai sehingga lebih besar dan start-up lebih lambat.',
    inspection_check: 'Hanya diuji saat Deep Check: membaca blok release pada build.gradle(.kts) modul aplikasi.',
    suggested_action: 'Aktifkan minify + shrinkResources dan uji aturan ProGuard.',
    example_code: {
      bad: 'release { isMinifyEnabled = false }',
      good: 'release {\n    isMinifyEnabled = true\n    isShrinkResources = true\n}'
    }
  },
  {
    id: 'DEEP-AND-DEBUGGABLE',
    category: 'security',
    title: 'AndroidManifest android:debuggable="true"',
    scope: 'Mobile',
    severity: 'critical',
    is_deep: true,
    rule: 'Manifest tidak boleh menetapkan debuggable secara eksplisit; biarkan Gradle mengatur per buildType.',
    rationale: 'Aplikasi rilis dapat di-attach debugger, data privat terbaca via run-as, dan performa runtime turun.',
    inspection_check: 'Hanya diuji saat Deep Check: membaca AndroidManifest.xml (src/main).',
    suggested_action: 'Hapus atribut android:debuggable.',
    example_code: {
      bad: '<application android:debuggable="true">',
      good: '<application>'
    }
  },
  {
    id: 'DEEP-AND-CLEARTEXT',
    category: 'security',
    title: 'AndroidManifest usesCleartextTraffic="true"',
    scope: 'Mobile',
    severity: 'warning',
    is_deep: true,
    rule: 'HTTP tanpa enkripsi tidak boleh diizinkan global; pengecualian domain dev lewat network_security_config.',
    rationale: 'Data pengguna dapat disadap/dimodifikasi di jaringan publik.',
    inspection_check: 'Hanya diuji saat Deep Check: membaca atribut usesCleartextTraffic di manifest.',
    suggested_action: 'Gunakan HTTPS dan networkSecurityConfig khusus dev.',
    example_code: {
      bad: '<application android:usesCleartextTraffic="true">',
      good: '<application android:networkSecurityConfig="@xml/network_security_config">'
    }
  },
  {
    id: 'DEEP-AND-LARGE-HEAP',
    category: 'mobile',
    title: 'android:largeHeap / hardwareAccelerated="false"',
    scope: 'Mobile',
    severity: 'info',
    is_deep: true,
    rule: 'Hindari largeHeap="true" (menutupi memory leak) dan jangan mematikan hardware acceleration di level aplikasi. (Dilaporkan sebagai DEEP-AND-LARGE-HEAP / DEEP-AND-HW-ACCEL-OFF.)',
    rationale: 'largeHeap memperlama GC dan menyembunyikan leak; rendering CPU membuat animasi & scroll patah.',
    inspection_check: 'Hanya diuji saat Deep Check: membaca atribut largeHeap dan hardwareAccelerated di manifest.',
    suggested_action: 'Profil memori dengan LeakCanary, perbaiki leak, hapus largeHeap; matikan HW accel hanya per-View bila perlu.',
    example_code: {
      bad: '<application android:largeHeap="true" android:hardwareAccelerated="false">',
      good: '<application> <!-- default: heap normal, HW accel aktif -->'
    }
  },

  // ================= 14. REVIEW LLM (VIA ANTREAN) =================
  {
    id: 'LLM-REVIEW-*',
    category: 'code_smell',
    title: 'Review Kode oleh LLM Sesuai Bahasa (Opsional, via Antrean)',
    scope: 'Universal',
    severity: 'info',
    rule: 'Tombol "Review LLM" memasukkan job llm_audit ke antrean engine. Maksimal 8 berkas prioritas (paling banyak temuan statis / paling besar, maks 3 per bahasa) direview dengan checklist ekosistemnya: Go, Node.js, Python, PHP, Java/Kotlin server, Frontend web (Vue/React/Svelte), Android (Kotlin/Java & XML layout), Flutter, dan Dart non-Flutter (server/CLI, tanpa checklist widget).',
    rationale: 'Regex statis tidak memahami alur logika, re-render, lifecycle, atau otorisasi. LLM melengkapi temuan yang sulit dideteksi pola, tetapi lambat sehingga wajib lewat antrean agar UI tidak terkunci dan pengguna melihat progresnya.',
    inspection_check: 'Berjalan di antrean (1 slot LLM): progres per berkas tampil di bar Antrean. Setiap prompt hanya memuat aturan bahasa berkas itu; kategori temuan dibatasi per ekosistem (UI web → frontend, Android/Flutter → mobile, backend → query/router). Temuan statis yang sudah ada tidak diulang.',
    suggested_action: 'Temuan bertanda badge LLM dan rule_id LLM-REVIEW-<BAHASA>; gunakan filter "Review LLM" dan verifikasi sebelum menerapkan perbaikan.',
    example_code: {
      bad: '// Menunggu LLM secara sinkron di request HTTP → UI membeku puluhan detik',
      good: '// Antrean non-blocking:\nPOST /api/queue/enqueue { type: \'llm_audit\', project_id: 1 }  // → 202 + progres di bar Antrean'
    }
  }
]
