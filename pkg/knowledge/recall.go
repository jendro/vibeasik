package knowledge

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Query adalah permintaan temu-kembali pengetahuan untuk satu pekerjaan.
type Query struct {
	Text        string   `json:"text"`
	Domains     []string `json:"domains,omitempty"`     // ekosistem/area yang sudah diketahui pemanggil (ditambah deteksi dari Text)
	EmployeeID  uint     `json:"employee_id,omitempty"` // pengetahuan pribadi pegawai ini ikut; milik pegawai lain tidak
	Kinds       []string `json:"kinds,omitempty"`
	SkipSources []string `json:"skip_sources,omitempty"`
	// Context: kunci pekerjaan (todo:12, audit:3:path, plan:4). Diisi = pemakaian dicatat untuk dinilai & celah dideteksi.
	Context string `json:"context,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	// DryRun: hitung celah seperti pekerjaan sungguhan, tetapi jangan catat pemakaian maupun celah (pratinjau prompt).
	DryRun bool `json:"-"`
}

// Hit adalah satu node hasil temu-kembali beserta rincian skornya.
type Hit struct {
	models.KnowledgeNode
	Score     float64 `json:"score"`
	Relevance float64 `json:"relevance"`
	UtilityV  float64 `json:"utility"`
	Via       string  `json:"via,omitempty"` // "tautan" = ditemukan lewat tautan node lain (bukan kecocokan langsung)
}

// Result: node terpilih, domain yang dikenali, dan ekosistem yang belum punya pengetahuan sama sekali.
type Result struct {
	Hits    []Hit    `json:"hits"`
	Domains []string `json:"domains"`
	Gaps    []string `json:"gaps,omitempty"`
}

const (
	minRelevance  = 0.08
	domainBonus   = 0.12 // node di domain yang diminta tetap terangkat walau katanya tidak sama persis
	candidatePool = 80
	recencyDays   = 45.0
	minGapCorpus  = 20 // celah baru dicatat setelah pusat terisi (bukan saat sinkron awal belum jalan)
)

// Recall memilih node paling berguna untuk sebuah pekerjaan:
//
//	skor = 0.55·relevansi (kosinus TF-IDF + bonus domain, dinormalisasi) + 0.20·utilitas (Beta helpful/harmful)
//	     + 0.15·kepentingan (bobot) + 0.10·kebaruan (peluruhan 45 hari sejak terakhir dipakai)
//
// Node ekosistem lain disaring (patokan Flutter tidak masuk todo Go), node merugikan (✗≥3, utilitas <0.25)
// dilewati, tetangga bertaut dari node teratas ikut dipertimbangkan, dan tiap klaster dibatasi agar hasil beragam.
func Recall(db *gorm.DB, q Query) Result {
	if db == nil {
		return Result{}
	}
	if q.Limit <= 0 || q.Limit > 50 {
		q.Limit = 10
	}
	ix := current(db)
	domains := uniq(append(lowerAll(q.Domains), DetectDomains(q.Text)...))
	res := Result{Domains: domains}
	eco := map[string]bool{}
	for _, d := range domains {
		if IsEcosystem(d) {
			eco[d] = true
		}
	}
	kinds := setOf(q.Kinds)
	skip := setOf(q.SkipSources)
	allowed := func(e *entry) bool {
		if e.owner != 0 && e.owner != q.EmployeeID {
			return false
		}
		if (len(kinds) > 0 && !kinds[e.kind]) || skip[e.source] {
			return false
		}
		return len(eco) == 0 || e.domain == "" || !IsEcosystem(e.domain) || eco[e.domain]
	}

	rel := map[uint]float64{}
	qv := ix.queryVector(q.Text, "", strings.Join(domains, " "), nil)
	for _, nb := range ix.nearest(qv, 0, 0.0001, func(e *entry) bool { return !allowed(e) }) {
		rel[nb.e.id] = nb.sim
	}
	want := setOf(domains)
	for _, e := range ix.entries {
		if want[e.domain] && allowed(e) {
			rel[e.id] += domainBonus
		}
	}
	ids := make([]uint, 0, len(rel))
	for id, r := range rel {
		if r >= minRelevance {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return rel[ids[i]] > rel[ids[j]] || rel[ids[i]] == rel[ids[j]] && ids[i] < ids[j] })
	if len(ids) > candidatePool {
		ids = ids[:candidatePool]
	}
	maxRel := 0.0
	for _, id := range ids {
		maxRel = math.Max(maxRel, rel[id])
	}

	hits := scoreNodes(db, ids, func(id uint) float64 { return rel[id] / maxRel })
	for i := range hits {
		hits[i].Relevance = round3(rel[hits[i].ID])
	}
	sortHits(hits)
	hits = append(hits, expand(db, ix, hits, q.Limit, allowed)...)
	sortHits(hits)
	res.Hits = diversify(hits, q.Limit)

	if q.Context != "" {
		if ix.n >= minGapCorpus {
			for d := range eco {
				if ix.domainN[d] == 0 {
					res.Gaps = append(res.Gaps, d)
					if !q.DryRun {
						recordAutoGap(db, d, q.Text)
					}
				}
			}
			sort.Strings(res.Gaps)
		}
		if q.DryRun {
			return res
		}
		hitIDs := make([]uint, 0, len(res.Hits))
		for _, h := range res.Hits {
			hitIDs = append(hitIDs, h.ID)
		}
		recordUse(db, q.Context, q.EmployeeID, hitIDs)
	}
	return res
}

// scoreNodes memuat node kandidat dan menghitung skornya (relN = relevansi ternormalisasi 0–1).
func scoreNodes(db *gorm.DB, ids []uint, relN func(uint) float64) []Hit {
	if len(ids) == 0 {
		return nil
	}
	var nodes []models.KnowledgeNode
	db.Where("id IN ? AND status = ?", ids, models.KnowledgeNodeActive).Find(&nodes)
	now := time.Now()
	out := make([]Hit, 0, len(nodes))
	for _, n := range nodes {
		u := n.Utility()
		if models.KnowledgeAuthoritative(n.Source) {
			u = 1 // sumber berkedudukan tinggi (Galeri Desain): tidak digeser bukti dari hasil kerja pegawai
		} else if n.Harmful >= 3 && u < 0.25 {
			continue
		}
		last := n.UpdatedAt
		if n.LastUsedAt != nil && n.LastUsedAt.After(last) {
			last = *n.LastUsedAt
		}
		recency := math.Exp(-now.Sub(last).Hours() / 24 / recencyDays)
		score := 0.55*relN(n.ID) + 0.20*u + 0.15*float64(n.Weight)/100 + 0.10*recency
		out = append(out, Hit{KnowledgeNode: n, Score: round3(score), UtilityV: round3(u)})
	}
	return out
}

// expand: tetangga bertaut dari node teratas ikut dinilai (aktivasi menyebar satu langkah, gaya HippoRAG/A-MEM).
func expand(db *gorm.DB, ix *index, hits []Hit, limit int, allowed func(*entry) bool) []Hit {
	if len(hits) == 0 {
		return nil
	}
	seeds := map[uint]float64{}
	have := map[uint]bool{}
	for i, h := range hits {
		have[h.ID] = true
		if i < limit {
			seeds[h.ID] = h.Score
		}
	}
	seedIDs := make([]uint, 0, len(seeds))
	for id := range seeds {
		seedIDs = append(seedIDs, id)
	}
	var links []models.KnowledgeLink
	db.Where("from_id IN ? OR to_id IN ?", seedIDs, seedIDs).Find(&links)
	via := map[uint]float64{}
	for _, l := range links {
		seed, other := l.FromID, l.ToID
		if _, ok := seeds[seed]; !ok {
			seed, other = other, seed
		}
		if have[other] {
			continue
		}
		e := ix.byID[other]
		if e == nil || !allowed(e) {
			continue
		}
		via[other] = math.Max(via[other], seeds[seed]*l.Strength)
	}
	if len(via) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(via))
	for id := range via {
		ids = append(ids, id)
	}
	out := scoreNodes(db, ids, func(uint) float64 { return 0 })
	for i := range out {
		// Skor tetangga = 0.7 × skor asal × kekuatan tautan + 0.3 × skornya sendiri tanpa relevansi langsung.
		out[i].Score = round3(via[out[i].ID]*0.7 + out[i].Score*0.3)
		out[i].Via = "tautan"
	}
	return out
}

func sortHits(h []Hit) {
	sort.SliceStable(h, func(i, j int) bool { return h[i].Score > h[j].Score || h[i].Score == h[j].Score && h[i].ID < h[j].ID })
}

// diversify membatasi jumlah node per klaster agar satu konsep tidak memenuhi seluruh prompt.
func diversify(h []Hit, limit int) []Hit {
	perCluster := max(2, (limit+1)/2)
	count := map[uint]int{}
	out := make([]Hit, 0, limit)
	for _, x := range h {
		if len(out) >= limit {
			break
		}
		if x.ClusterID != 0 {
			if count[x.ClusterID] >= perCluster {
				continue
			}
			count[x.ClusterID]++
		}
		out = append(out, x)
	}
	return out
}

var kindLabel = map[string]string{
	models.KnowledgeKindRule: "patokan", models.KnowledgeKindProcedure: "prosedur",
	models.KnowledgeKindLesson: "pelajaran", models.KnowledgeKindConcept: "konsep",
}

// Lines menulis hit sebagai butir prompt: "- [K12 · patokan · go] Judul: isi (✓3 ✗0)".
func Lines(hits []Hit) string {
	var b strings.Builder
	for _, h := range hits {
		tag := "K" + fmt.Sprint(h.ID) + " · " + kindLabel[h.Kind]
		if h.Domain != "" {
			tag += " · " + h.Domain
		}
		fmt.Fprintf(&b, "- [%s] %s: %s", tag, h.Title, clip(h.Content, 320))
		if h.Helpful+h.Harmful > 0 {
			fmt.Fprintf(&b, " (✓%d ✗%d)", h.Helpful, h.Harmful)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// GapLine: pesan untuk agent bila pekerjaannya menyentuh ekosistem yang belum dikenal kantor.
func GapLine(gaps []string) string {
	if len(gaps) == 0 {
		return ""
	}
	return "- Celah pengetahuan: kantor belum punya pengetahuan tentang " + strings.Join(gaps, ", ") +
		". Baca dokumentasi resmi/kode yang ada dulu, jangan menebak dari bahasa lain, lalu tulis temuanmu di PELAJARAN agar menjadi node baru Pusat Pengetahuan.\n"
}

// FeedbackLine meminta agent melaporkan butir yang keliru (bukti untuk pembelajaran pusat).
const FeedbackLine = "- Butir berlabel K<id> berasal dari Pusat Pengetahuan. Bila ada yang keliru/menyesatkan untuk kasus ini, sebut ID-nya di catatan penyelesaian atau laporkan lewat MCP vibeasik_knowledge_feedback.\n"

// Brief = butir terpilih + celah (tanpa judul bagian); kosong bila tidak ada apa pun.
func Brief(db *gorm.DB, q Query) string {
	res := Recall(db, q)
	if len(res.Hits) == 0 && len(res.Gaps) == 0 {
		return ""
	}
	return Lines(res.Hits) + GapLine(res.Gaps)
}

// Section = Brief dengan judul bagian & ajakan umpan balik, siap ditempel ke prompt.
func Section(db *gorm.DB, q Query, heading string) string {
	body := Brief(db, q)
	if body == "" {
		return ""
	}
	return "\n" + heading + ":\n" + body + FeedbackLine
}

func uniq(xs []string) []string {
	seen := map[string]bool{}
	out := xs[:0:0]
	for _, x := range xs {
		if x != "" && !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func lowerAll(xs []string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, strings.ToLower(strings.TrimSpace(x)))
	}
	return out
}

func setOf(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		if x != "" {
			m[x] = true
		}
	}
	return m
}
