package knowledge

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	invalidate()
	db, err := gorm.Open(sqlite.Open("file:"+filepath.Join(t.TempDir(), "k.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.KnowledgeNode{}, &models.KnowledgeLink{}, &models.KnowledgeCluster{}, &models.KnowledgeUse{},
		&models.KnowledgeGap{}, &models.Employee{}, &models.EmployeeKnowledge{}, &models.EmployeeAgenda{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func mustLearn(t *testing.T, db *gorm.DB, n Note) *models.KnowledgeNode {
	t.Helper()
	node, _, err := Learn(db, n)
	if err != nil {
		t.Fatal(err)
	}
	return node
}

func TestDetectDomains(t *testing.T) {
	cases := map[string]string{
		"Perbaiki tombol di app/pages/settings.vue":    "web",
		"Tambah endpoint di internal/api/server.go":    "go",
		"Controller Laravel tanpa FormRequest":         "php",
		"Porting modul ke src/core/engine.gleam":       "gleam",
		"Widget Flutter rebuild berlebih":              "flutter",
		"Baca config di deploy/app.toml dan README.md": "devops",
	}
	for text, want := range cases {
		if got := DetectDomains(text); len(got) == 0 || got[0] != want {
			t.Errorf("DetectDomains(%q) = %v, mau %s di depan", text, got, want)
		}
	}
	for _, text := range []string{"return fmt.Errorf(\"x\")", "lihat versi v1.2 dan 2026.11", "buka https://example.com/docs"} {
		for _, d := range DetectDomains(text) {
			if IsEcosystem(d) {
				t.Errorf("DetectDomains(%q) tidak boleh menebak ekosistem, dapat %v", text, d)
			}
		}
	}
	if StackDomain("typescript", "nuxt") != "web" || StackDomain("go", "fiber") != "go" || StackDomain("dart", "") != "flutter" {
		t.Error("StackDomain salah memetakan stack komponen")
	}
}

func TestLearnReinforcesDuplicate(t *testing.T) {
	db := testDB(t)
	a := mustLearn(t, db, Note{Title: "Bungkus error dengan konteks", Content: "Gunakan fmt.Errorf dengan %w agar penyebab error tidak hilang saat dilempar ke atas.", Domain: "go"})
	b, merged, err := Learn(db, Note{Title: "Bungkus error dengan konteks", Content: "Gunakan fmt.Errorf dengan %w agar penyebab error tidak hilang.", Domain: "go"})
	if err != nil || !merged || b.ID != a.ID {
		t.Fatalf("butir hampir sama harus diperkuat, bukan digandakan: merged=%v id=%d/%d err=%v", merged, b.ID, a.ID, err)
	}
	var n models.KnowledgeNode
	db.First(&n, a.ID)
	if n.Helpful != 1 || n.Weight <= 60 {
		t.Fatalf("penguatan harus menambah bukti & bobot, dapat helpful=%d weight=%d", n.Helpful, n.Weight)
	}
	// Lingkup pribadi pegawai lain tidak dilebur ke pustaka bersama.
	emp := uint(7)
	c := mustLearn(t, db, Note{Title: "Bungkus error dengan konteks", Content: "Gunakan fmt.Errorf dengan %w.", Domain: "go", OwnerEmployeeID: &emp})
	if c.ID == a.ID {
		t.Fatal("pengetahuan pribadi tidak boleh dilebur ke node bersama")
	}
}

func seedCorpus(t *testing.T, db *gorm.DB) map[string]uint {
	t.Helper()
	ids := map[string]uint{}
	add := func(key, domain, title, content string) {
		ids[key] = mustLearn(t, db, Note{Kind: models.KnowledgeKindRule, Domain: domain, Title: title, Content: content}).ID
	}
	add("go-ctx", "go", "Teruskan context ke goroutine", "Goroutine wajib menerima context dan berhenti saat context dibatalkan agar tidak bocor (goroutine leak).")
	add("go-err", "go", "Jangan abaikan error", "Error dari query database harus diperiksa dan dibungkus konteks.")
	add("fl-dispose", "flutter", "Dispose controller", "Controller, stream, dan subscription Flutter wajib di-dispose di dispose().")
	add("fl-const", "flutter", "Pakai const constructor", "Widget statis memakai const agar rebuild hemat.")
	add("sec-input", "security", "Validasi input di satu lapisan", "Validasi input pengguna dilakukan di handler sebelum menyentuh database untuk mencegah injection.")
	add("web-a11y", "web", "Label tombol ikon", "Tombol ikon wajib punya aria-label agar aksesibel.")
	add("go-timeout", "go", "Context timeout pada HTTP client", "HTTP client keluar wajib memakai context dengan timeout agar goroutine pemanggil tidak menggantung.")
	topics := []string{"pembayaran", "notifikasi", "laporan", "unggahan", "pencarian", "jadwal", "keranjang", "invoice", "absensi", "sertifikat",
		"kupon", "ulasan", "langganan", "pengiriman", "inventaris", "obrolan", "kalender", "peta", "survei", "dompet"}
	for i, tp := range topics {
		mustLearn(t, db, Note{Kind: models.KnowledgeKindLesson, Domain: "testing", Title: "Test regresi modul " + tp,
			Content: fmt.Sprintf("Sebelum memperbaiki bug %s, tulis test yang gagal dulu (kasus %c).", tp, 'a'+i)})
	}
	return ids
}

func TestRecallFiltersEcosystemAndRanksRelevant(t *testing.T) {
	db := testDB(t)
	ids := seedCorpus(t, db)
	res := Recall(db, Query{Text: "Goroutine worker di internal/queue/worker.go bocor saat context dibatalkan", Limit: 5})
	if len(res.Hits) == 0 || res.Hits[0].ID != ids["go-ctx"] {
		t.Fatalf("node goroutine harus teratas, dapat %+v", res.Hits)
	}
	for _, h := range res.Hits {
		if h.Domain == "flutter" || h.Domain == "web" {
			t.Fatalf("patokan ekosistem lain tidak boleh ikut ke todo Go: %s (%s)", h.Title, h.Domain)
		}
	}
	res = Recall(db, Query{Text: "Validasi input form login sebelum query", Domains: []string{"flutter"}, Limit: 5})
	found := false
	for _, h := range res.Hits {
		found = found || h.ID == ids["sec-input"]
		if h.Domain == "go" {
			t.Fatalf("todo Flutter tidak boleh menerima patokan Go: %s", h.Title)
		}
	}
	if !found {
		t.Fatal("patokan area (security) harus tetap berlaku lintas ekosistem")
	}
}

func TestRecallRecordsUseAndReward(t *testing.T) {
	db := testDB(t)
	ids := seedCorpus(t, db)
	res := Recall(db, Query{Text: "dispose stream subscription Flutter di lib/home.dart", Context: "todo:9", EmployeeID: 3, Limit: 4})
	if len(res.Hits) == 0 {
		t.Fatal("harus ada hit")
	}
	Recall(db, Query{Text: "dispose stream subscription Flutter di lib/home.dart", Context: "todo:9", EmployeeID: 3, Limit: 4})
	var uses int64
	db.Model(&models.KnowledgeUse{}).Where("context = ?", "todo:9").Count(&uses)
	if int(uses) != len(res.Hits) {
		t.Fatalf("prompt yang dibangun ulang untuk todo sama tidak boleh menggandakan pemakaian: %d vs %d", uses, len(res.Hits))
	}
	if n := Reward(db, "todo:9", 88); n != len(res.Hits) {
		t.Fatalf("Reward harus menilai semua pemakaian, dapat %d", n)
	}
	var node models.KnowledgeNode
	db.First(&node, ids["fl-dispose"])
	if node.Helpful != 1 || node.Uses != 1 {
		t.Fatalf("node terpakai di todo bernilai bagus harus helpful+1 & uses 1, dapat helpful=%d uses=%d", node.Helpful, node.Uses)
	}
	if Reward(db, "todo:9", 20) != 0 {
		t.Fatal("pemakaian yang sudah dinilai tidak dinilai dua kali")
	}
}

func TestGapDetectedAndFilled(t *testing.T) {
	db := testDB(t)
	seedCorpus(t, db)
	res := Recall(db, Query{Text: "Porting parser ke src/core/parser.gleam", Context: "todo:1"})
	if len(res.Gaps) != 1 || res.Gaps[0] != "gleam" {
		t.Fatalf("ekosistem tanpa node harus jadi celah, dapat %v", res.Gaps)
	}
	Recall(db, Query{Text: "Lanjutkan src/core/lexer.gleam", Context: "todo:2"})
	gaps := OpenGaps(db, 5)
	if len(gaps) != 1 || gaps[0].Hits != 2 || !gaps[0].Auto {
		t.Fatalf("celah yang sama harus digabung, dapat %+v", gaps)
	}
	mustLearn(t, db, Note{Kind: models.KnowledgeKindConcept, Domain: "gleam", Title: "Dasar Gleam", Content: "Gleam bahasa bertipe statis di BEAM; build & test lewat `gleam build` / `gleam test`.", Source: models.KnowledgeSrcGap})
	if g, _ := GapByID(db, gaps[0].ID); g.Status != models.KnowledgeGapLearned {
		t.Fatalf("node pertama di domain celah harus menutup celah, status %s", g.Status)
	}
	if res := Recall(db, Query{Text: "Perbaiki src/core/parser.gleam", Context: "todo:3"}); len(res.Gaps) != 0 || len(res.Hits) == 0 || res.Hits[0].Domain != "gleam" {
		t.Fatalf("setelah belajar, todo Gleam harus mendapat node Gleam tanpa celah: %+v", res)
	}
}

func TestSyncSetKeepsEvidenceAndRetiresMissing(t *testing.T) {
	db := testDB(t)
	scope := bySource(models.KnowledgeSrcGuideline)
	notes := []Note{
		{Key: "guideline:SEC-001", Kind: "rule", Domain: "security", Source: models.KnowledgeSrcGuideline, Title: "SEC-001 Env", Content: "Jangan commit .env"},
		{Key: "guideline:SEC-002", Kind: "rule", Domain: "security", Source: models.KnowledgeSrcGuideline, Title: "SEC-002 Path", Content: "Tanpa path mesin"},
	}
	if n := syncSet(db, scope, notes); n != 2 {
		t.Fatalf("sinkron awal harus membuat 2 node, dapat %d", n)
	}
	var sec1 models.KnowledgeNode
	db.Where("source_key = ?", "guideline:SEC-001").First(&sec1)
	_, _ = Feedback(db, sec1.ID, true)
	notes[0].Content = "Jangan pernah commit .env ke git"
	if n := syncSet(db, scope, notes[:1]); n != 2 {
		t.Fatalf("1 berubah + 1 dipensiunkan = 2, dapat %d", n)
	}
	db.First(&sec1, sec1.ID)
	if sec1.Helpful != 1 || sec1.Content != "Jangan pernah commit .env ke git" {
		t.Fatalf("sinkron memperbarui teks tanpa menghapus bukti: %+v", sec1)
	}
	var sec2 models.KnowledgeNode
	db.Where("source_key = ?", "guideline:SEC-002").First(&sec2)
	if sec2.Status != models.KnowledgeNodeRetired {
		t.Fatal("butir yang hilang dari sumber harus dipensiunkan")
	}
	syncSet(db, scope, notes)
	db.First(&sec2, sec2.ID)
	if sec2.Status != models.KnowledgeNodeActive {
		t.Fatal("butir yang kembali ke sumber harus aktif lagi")
	}
}

func TestCurateMergesLinksClustersAndPrunes(t *testing.T) {
	db := testDB(t)
	seedCorpus(t, db)
	// Dua node asli hampir sama yang lolos dedup saat ditulis (mis. ditulis proses lain) dilebur kurator.
	a := models.KnowledgeNode{Kind: "lesson", Domain: "go", Title: "Tutup rows query", Content: "Selalu defer rows.Close setelah query database.", Weight: 60, Status: "active", Helpful: 2}
	b := models.KnowledgeNode{Kind: "lesson", Domain: "go", Title: "Tutup rows query", Content: "Selalu defer rows.Close setelah query database.", Weight: 50, Status: "active"}
	bad := models.KnowledgeNode{Kind: "lesson", Domain: "go", Title: "Abaikan error kecil", Content: "Error log boleh diabaikan.", Weight: 50, Status: "active", Harmful: 4}
	db.Create(&a)
	db.Create(&b)
	db.Create(&bad)
	invalidate()
	r := Curate(db)
	if r.Merged < 1 || r.Retired != 1 || r.Links == 0 || r.Clusters == 0 {
		t.Fatalf("kurasi harus melebur, memangkas, menautkan & mengklaster: %+v", r)
	}
	db.First(&b, b.ID)
	if b.Status != models.KnowledgeNodeMerged || b.MergedIntoID == nil || *b.MergedIntoID != a.ID {
		t.Fatalf("node lemah dilebur ke node kuat: %+v", b)
	}
	db.First(&bad, bad.ID)
	if bad.Status != models.KnowledgeNodeRetired {
		t.Fatal("node yang terbukti merugikan dipensiunkan")
	}
	var clustered int64
	db.Model(&models.KnowledgeNode{}).Where("cluster_id <> 0").Count(&clustered)
	if clustered < 2 {
		t.Fatal("node berklaster harus tercatat cluster_id-nya")
	}
}

func TestPromoteProvenPersonalKnowledge(t *testing.T) {
	db := testDB(t)
	emp := uint(5)
	p := models.KnowledgeNode{Kind: "procedure", Domain: "web", Title: "Debounce pencarian", Content: "Input pencarian memakai debounce 300ms.", Source: models.KnowledgeSrcEmployee,
		Key: "employee:1", OwnerEmployeeID: &emp, OwnerName: "Sari", Weight: 70, Helpful: 4, Status: "active"}
	db.Create(&p)
	if n := promote(db); n != 1 {
		t.Fatalf("pengetahuan pribadi yang terbukti harus dipromosikan, dapat %d", n)
	}
	if n := promote(db); n != 0 {
		t.Fatal("promosi idempoten")
	}
	res := Recall(db, Query{Text: "debounce input pencarian di app/pages/search.vue", EmployeeID: 99})
	if len(res.Hits) != 1 || res.Hits[0].OwnerEmployeeID != nil || res.Hits[0].OwnerName != "Sari" {
		t.Fatalf("pegawai lain menerima versi bersama (kredit tetap ke pemiliknya): %+v", res.Hits)
	}
}

func TestEmployeeSyncSkipsExecutiveBoard(t *testing.T) {
	db := testDB(t)
	eng := models.Employee{Name: "Rani", Division: "engineering", Status: models.EmployeeActive}
	ceo := models.Employee{Name: "Ananta", Division: execDivision, Status: models.EmployeeActive}
	db.Create(&eng)
	db.Create(&ceo)
	db.Create(&models.EmployeeKnowledge{EmployeeID: eng.ID, Topic: "Index gabungan", Content: "Filter dua kolom butuh index gabungan.", Kind: "technical", Weight: 70, Status: models.KnowledgeActive})
	db.Create(&models.EmployeeKnowledge{EmployeeID: ceo.ID, Topic: "Konteks Produk: Toko", Content: "Proyek Toko dipimpin dewan.", Kind: "lesson", Weight: 90, Status: models.KnowledgeActive})
	SyncSources(db)
	var nodes []models.KnowledgeNode
	db.Where("source = ?", models.KnowledgeSrcEmployee).Find(&nodes)
	if len(nodes) != 1 || nodes[0].OwnerName != "Rani" || nodes[0].Kind != models.KnowledgeKindProcedure {
		t.Fatalf("hanya keahlian umum pegawai non-eksekutif yang dicerminkan (fakta proyek milik dewan), dapat %+v", nodes)
	}
}

func TestSettleStaleClosesOnlyUsesThatWillNeverBeScored(t *testing.T) {
	db := testDB(t)
	if err := db.AutoMigrate(&models.Todo{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	node := mustLearn(t, db, Note{Title: "Tutup stream", Content: "Selalu batalkan StreamSubscription di dispose.", Kind: models.KnowledgeKindRule, Domain: "flutter"})
	score, old, fresh := 60, now.Add(-48*time.Hour), now.Add(-time.Hour)
	todos := []models.Todo{
		{Title: "selesai, dinilai, dikerjakan ulang", Status: models.TodoStatusCompleted, Score: &score, CompletedAt: &old},
		{Title: "selesai, baru saja dinilai", Status: models.TodoStatusCompleted, Score: &score, CompletedAt: &fresh},
		{Title: "selesai, belum dinilai", Status: models.TodoStatusCompleted, CompletedAt: &old},
		{Title: "masih dikerjakan", Status: models.TodoStatusInProgress, Score: &score, CompletedAt: &old},
	}
	if err := db.Create(&todos).Error; err != nil {
		t.Fatal(err)
	}
	todoCtx := func(i int) string { return fmt.Sprintf("todo:%d", todos[i].ID) }
	want := map[string]string{
		todoCtx(0):         models.KnowledgeOutcomeNeutral,
		todoCtx(1):         "",
		todoCtx(2):         "",
		todoCtx(3):         "",
		"todo:999999":      models.KnowledgeOutcomeNeutral, // todo sudah dihapus
		"audit:1:lama.go":  models.KnowledgeOutcomeNeutral,
		"plan:1":           models.KnowledgeOutcomeNeutral,
		"audit:1:baru.go":  "",
		"todo:bukan-angka": "",
	}
	for ctx := range want {
		at := now.Add(-8 * 24 * time.Hour)
		if ctx == "audit:1:baru.go" {
			at = now.Add(-24 * time.Hour)
		}
		if err := db.Create(&models.KnowledgeUse{NodeID: node.ID, Context: ctx, CreatedAt: at}).Error; err != nil {
			t.Fatal(err)
		}
	}
	done := models.KnowledgeUse{NodeID: node.ID, Context: "audit:1:lama.go", Outcome: models.KnowledgeOutcomeHelpful, CreatedAt: now.Add(-9 * 24 * time.Hour)}
	db.Create(&done)

	if n := SettleStale(db, now); n != 4 {
		t.Fatalf("harus menutup 4 pemakaian tanpa hasil, dapat %d", n)
	}
	var uses []models.KnowledgeUse
	db.Where("id <> ?", done.ID).Find(&uses)
	for _, u := range uses {
		if u.Outcome != want[u.Context] {
			t.Errorf("%s: hasil %q, seharusnya %q", u.Context, u.Outcome, want[u.Context])
		}
	}
	db.First(&done, done.ID)
	if done.Outcome != models.KnowledgeOutcomeHelpful {
		t.Errorf("pemakaian yang sudah dinilai tidak boleh berubah, dapat %q", done.Outcome)
	}
	var after models.KnowledgeNode
	db.First(&after, node.ID)
	if after.Helpful != node.Helpful || after.Harmful != node.Harmful {
		t.Errorf("penutupan netral tidak boleh mengubah bukti node: ✓%d ✗%d", after.Helpful, after.Harmful)
	}
	if n := SettleStale(db, now); n != 0 {
		t.Fatalf("putaran kedua tidak boleh menutup apa pun, dapat %d", n)
	}
}
