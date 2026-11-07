package knowledge

import (
	"context"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Kurator (peran "Curator" di ACE): berjalan berkala di `vibeasik serve` dan bisa dipicu manual.
//  1. sinkron sumber (guidelines, pegawai, checklist audit)
//  2. lebur node asli yang hampir sama (kosinus ≥ 0.85)
//  3. promosikan pengetahuan pribadi yang terbukti berguna (✓≥3, utilitas ≥0.75) ke pustaka bersama
//  4. pangkas node asli yang terbukti merugikan; patokan berkas yang sering keliru ditandai perlu ditinjau
//  5. hitung ulang tautan (3 tetangga terdekat) & klaster konsep
//  6. buang catatan pemakaian > 30 hari

// Report merangkum satu putaran kurasi.
type Report struct {
	Synced   int    `json:"synced"`
	Merged   int    `json:"merged"`
	Promoted int    `json:"promoted"`
	Retired  int    `json:"retired"`
	Flagged  int    `json:"flagged"`
	Links    int    `json:"links"`
	Clusters int    `json:"clusters"`
	Nodes    int    `json:"nodes"`
	Took     string `json:"took"`
}

const (
	mergeMin       = 0.85
	clusterMin     = 0.3
	linksPerNode   = 3
	curateInterval = 15 * time.Minute
	useRetention   = 30 * 24 * time.Hour
)

var (
	curateMu     sync.Mutex
	extraSources []Source
)

// Start menjalankan kurator berkala (hanya dari `vibeasik serve`). extra = sumber tambahan, mis. checklist audit.
func Start(ctx context.Context, db *gorm.DB, extra ...Source) {
	extraSources = extra
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(20 * time.Second):
		}
		r := Curate(db)
		log.Printf("[Knowledge] kurasi awal: %d node, %d disinkron, %d tautan, %d klaster (%s)", r.Nodes, r.Synced, r.Links, r.Clusters, r.Took)
		t := time.NewTicker(curateInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				Curate(db)
			}
		}
	}()
}

// Curate menjalankan satu putaran kurasi penuh.
func Curate(db *gorm.DB) Report {
	curateMu.Lock()
	defer curateMu.Unlock()
	began := time.Now()
	var r Report
	r.Synced = SyncSources(db, extraSources...)
	r.Merged = mergeDuplicates(db)
	r.Promoted = promote(db)
	r.Retired, r.Flagged = prune(db)
	ix := current(db)
	r.Nodes = ix.n
	r.Links = relink(db, ix)
	r.Clusters = recluster(db, ix)
	db.Where("created_at < ?", time.Now().Add(-useRetention)).Delete(&models.KnowledgeUse{})
	r.Took = time.Since(began).Round(time.Millisecond).String()
	return r
}

func loadActive(db *gorm.DB, q func(*gorm.DB) *gorm.DB) map[uint]models.KnowledgeNode {
	var rows []models.KnowledgeNode
	q(db.Where("status = ?", models.KnowledgeNodeActive)).Find(&rows)
	out := make(map[uint]models.KnowledgeNode, len(rows))
	for _, r := range rows {
		out[r.ID] = r
	}
	return out
}

func strength(n models.KnowledgeNode) int { return n.Weight + 5*(n.Helpful-n.Harmful) + n.Uses }

// mergeDuplicates melebur node asli yang hampir sama di lingkup pemilik yang sama: bukti dijumlah, isi dilengkapi.
func mergeDuplicates(db *gorm.DB) int {
	ix := current(db)
	nodes := loadActive(db, func(q *gorm.DB) *gorm.DB { return q.Where("source_key = '' OR source_key LIKE 'promoted:%'") })
	done := map[uint]bool{}
	merged := 0
	for _, e := range ix.entries {
		a, ok := nodes[e.id]
		if !ok || done[e.id] {
			continue
		}
		for _, nb := range ix.nearest(e.v, 4, mergeMin, func(o *entry) bool { return o.id == e.id || o.owner != e.owner || done[o.id] }) {
			b, ok := nodes[nb.e.id]
			if !ok {
				continue
			}
			keep, drop := a, b
			if strength(b) > strength(a) {
				keep, drop = b, a
			}
			keep.Content = synthesize(keep.Content, drop.Content)
			keep.Helpful += drop.Helpful
			keep.Harmful += drop.Harmful
			keep.Uses += drop.Uses
			keep.Weight = clamp(max(keep.Weight, drop.Weight)+3, 1, 99)
			db.Model(&models.KnowledgeNode{}).Where("id = ?", keep.ID).Updates(map[string]interface{}{
				"content": keep.Content, "helpful": keep.Helpful, "harmful": keep.Harmful, "uses": keep.Uses, "weight": keep.Weight,
			})
			db.Model(&models.KnowledgeNode{}).Where("id = ?", drop.ID).Updates(map[string]interface{}{
				"status": models.KnowledgeNodeMerged, "merged_into_id": keep.ID, "status_reason": "Dilebur ke K" + itoa(keep.ID) + " (" + clip(keep.Title, 80) + ")",
			})
			nodes[keep.ID] = keep
			done[drop.ID] = true
			merged++
			if drop.ID == a.ID {
				break
			}
			a = keep
		}
	}
	if merged > 0 {
		invalidate()
	}
	return merged
}

// promote: pengetahuan pribadi yang berulang kali terbukti membantu dibagikan ke pustaka bersama kantor.
func promote(db *gorm.DB) int {
	var rows []models.KnowledgeNode
	db.Where("status = ? AND owner_employee_id IS NOT NULL AND helpful >= ?", models.KnowledgeNodeActive, 3).Find(&rows)
	n := 0
	for _, r := range rows {
		if r.Utility() < 0.75 {
			continue
		}
		_, created, err := upsertKey(db, Note{
			Key: "promoted:" + itoa(r.ID), Kind: r.Kind, Domain: r.Domain, Title: r.Title, Content: r.Content,
			Source: models.KnowledgeSrcPromoted, Tags: append([]string{"promosi"}, r.Tags...), OwnerName: r.OwnerName, Weight: r.Weight,
		})
		if err == nil && created {
			n++
		}
	}
	return n
}

// prune: node asli yang terbukti merugikan dipensiunkan; patokan bersumber berkas cukup ditandai perlu ditinjau
// (berkasnya tetap sumber kebenaran) — tanda dilepas lagi bila utilitasnya pulih.
func prune(db *gorm.DB) (retired, flagged int) {
	var bad []models.KnowledgeNode
	db.Where("status = ? AND harmful >= ?", models.KnowledgeNodeActive, 3).Find(&bad)
	for _, n := range bad {
		u := n.Utility()
		switch {
		case native(n.Key) && u < 0.3:
			db.Model(&n).Updates(map[string]interface{}{"status": models.KnowledgeNodeRetired,
				"status_reason": "Terbukti lebih sering merugikan (✓" + itoa(uint(n.Helpful)) + " ✗" + itoa(uint(n.Harmful)) + ")"})
			retired++
		case (n.Source == models.KnowledgeSrcGuideline || n.Source == models.KnowledgeSrcChecklist) && u < 0.4 && !n.NeedsReview:
			db.Model(&n).UpdateColumn("needs_review", true)
			flagged++
		}
	}
	var recovered []models.KnowledgeNode
	db.Where("status = ? AND needs_review = ?", models.KnowledgeNodeActive, true).Find(&recovered)
	for _, n := range recovered {
		if n.Utility() >= 0.5 {
			db.Model(&n).UpdateColumn("needs_review", false)
		}
	}
	if retired > 0 {
		invalidate()
	}
	return retired, flagged
}

// relink menghitung ulang tautan: tiap node ke 3 tetangga terdekat (kosinus ≥ 0.25). Node pribadi hanya bertaut
// ke node pemilik yang sama atau pustaka bersama.
func relink(db *gorm.DB, ix *index) int {
	type pair struct{ a, b uint }
	best := map[pair]float64{}
	for _, e := range ix.entries {
		for _, nb := range ix.nearest(e.v, linksPerNode, linkMin, func(o *entry) bool {
			return o.id == e.id || (o.owner != 0 && e.owner != 0 && o.owner != e.owner)
		}) {
			p := pair{min(e.id, nb.e.id), max(e.id, nb.e.id)}
			if nb.sim > best[p] {
				best[p] = nb.sim
			}
		}
	}
	links := make([]models.KnowledgeLink, 0, len(best))
	for p, s := range best {
		links = append(links, models.KnowledgeLink{FromID: p.a, ToID: p.b, Strength: round3(s)})
	}
	sort.Slice(links, func(i, j int) bool {
		return links[i].FromID < links[j].FromID || links[i].FromID == links[j].FromID && links[i].ToID < links[j].ToID
	})
	db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&models.KnowledgeLink{}).Error; err != nil {
			return err
		}
		if len(links) == 0 {
			return nil
		}
		return tx.CreateInBatches(&links, 200).Error
	})
	return len(links)
}

// recluster: klaster konsep dengan leader clustering satu lintasan pada vektor TF-IDF (node berbobot didahulukan
// sebagai pusat). Klaster berukuran ≥2 disimpan beserta label kata pusatnya; node tunggal tidak berklaster.
func recluster(db *gorm.DB, ix *index) int {
	meta := loadActive(db, func(q *gorm.DB) *gorm.DB { return q.Select("id", "weight", "helpful", "harmful", "uses", "domain") })
	order := make([]*entry, 0, len(ix.entries))
	for _, e := range ix.entries {
		if _, ok := meta[e.id]; ok {
			order = append(order, e)
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return strength(meta[order[i].id]) > strength(meta[order[j].id]) })

	type cluster struct {
		sum     vec
		norm    float64
		members []uint
		domains map[string]int
	}
	var cs []*cluster
	for _, e := range order {
		var bestC *cluster
		bestS := clusterMin
		for _, c := range cs {
			if s := e.v.dot(c.sum) / c.norm; s >= bestS {
				bestC, bestS = c, s
			}
		}
		if bestC == nil {
			bestC = &cluster{sum: vec{}, domains: map[string]int{}}
			cs = append(cs, bestC)
		}
		for t, w := range e.v {
			bestC.sum[t] += w
		}
		bestC.norm = normOf(bestC.sum)
		bestC.members = append(bestC.members, e.id)
		bestC.domains[e.domain]++
	}

	rows := make([]models.KnowledgeCluster, 0)
	members := map[int][]uint{}
	for _, c := range cs {
		if len(c.members) < 2 {
			continue
		}
		util := 0.0
		for _, id := range c.members {
			util += meta[id].Utility()
		}
		terms := topTerms(c.sum, 4)
		dom := ""
		if ds := sortedKeys(c.domains); len(ds) > 0 {
			dom = ds[0]
		}
		rows = append(rows, models.KnowledgeCluster{
			Label: strings.Join(terms, " · "), Domain: dom, Terms: terms, Size: len(c.members), Utility: round3(util / float64(len(c.members))),
		})
		members[len(rows)-1] = c.members
	}
	db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&models.KnowledgeCluster{}).Error; err != nil {
			return err
		}
		tx.Model(&models.KnowledgeNode{}).Where("cluster_id <> 0").UpdateColumn("cluster_id", 0)
		for i := range rows {
			if err := tx.Create(&rows[i]).Error; err != nil {
				return err
			}
			tx.Model(&models.KnowledgeNode{}).Where("id IN ?", members[i]).UpdateColumn("cluster_id", rows[i].ID)
		}
		return nil
	})
	return len(rows)
}

func normOf(v vec) float64 {
	s := 0.0
	for _, w := range v {
		s += w * w
	}
	if s == 0 {
		return 1
	}
	return math.Sqrt(s)
}

func itoa(n uint) string { return strconv.FormatUint(uint64(n), 10) }

func topTerms(v vec, n int) []string {
	terms := make([]string, 0, len(v))
	for t := range v {
		terms = append(terms, t)
	}
	sort.Slice(terms, func(i, j int) bool {
		return v[terms[i]] > v[terms[j]] || v[terms[i]] == v[terms[j]] && terms[i] < terms[j]
	})
	if len(terms) > n {
		terms = terms[:n]
	}
	return terms
}
