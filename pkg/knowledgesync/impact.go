package knowledgesync

import (
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Dampak pengetahuan. Setiap pemakaian node (KnowledgeUse) dicatat dengan konteks pekerjaan seperti
// "todo:12" atau "audit:3:path/berkas.go". Konteks itu milik pengguna dan TIDAK dikirim; hanya jenisnya
// (awalan sebelum ':') yang dipetakan ke daftar baku di bawah, lalu dihitung angkanya.

// WorkStat pemakaian dan nilai node pada satu jenis pekerjaan.
type WorkStat struct {
	Uses    int `json:"uses"`
	Helpful int `json:"helpful"`
	Harmful int `json:"harmful"`
}

// workKinds daftar baku jenis pekerjaan besar yang boleh dilaporkan. Kunci lain menjadi "lain".
var workKinds = map[string]bool{"todo": true, "audit": true, "plan": true}

// WorkKind memetakan konteks pekerjaan ke jenis bakunya. Mengembalikan "lain" untuk yang tak dikenal.
func WorkKind(context string) string {
	k, _, _ := strings.Cut(context, ":")
	if workKinds[k] {
		return k
	}
	return "lain"
}

// workByNode menghitung pemakaian dan nilai tiap node per jenis pekerjaan dari catatan lokal.
func workByNode(db *gorm.DB, nodes []models.KnowledgeNode) map[uint]map[string]WorkStat {
	out := map[uint]map[string]WorkStat{}
	if len(nodes) == 0 {
		return out
	}
	ids := make([]uint, 0, len(nodes))
	for _, n := range nodes {
		ids = append(ids, n.ID)
	}
	var uses []models.KnowledgeUse
	db.Select("node_id", "context", "outcome").Where("node_id IN ?", ids).Find(&uses)
	for _, u := range uses {
		m := out[u.NodeID]
		if m == nil {
			m = map[string]WorkStat{}
			out[u.NodeID] = m
		}
		k := WorkKind(u.Context)
		w := m[k]
		w.Uses++
		switch u.Outcome {
		case models.KnowledgeOutcomeHelpful:
			w.Helpful++
		case models.KnowledgeOutcomeHarmful:
			w.Harmful++
		}
		m[k] = w
	}
	return out
}

// KindImpact dampak pada satu jenis pekerjaan besar.
type KindImpact struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	WorkStat
	Score int `json:"score"` // 0-100 persen berguna; -1 = belum dinilai
}

// Impact ringkasan dampak teragregasi dari server (lihat vibeasik-cloud/internal/knowledgehub/impact.go).
type Impact struct {
	Devices   int          `json:"devices"`
	Total     WorkStat     `json:"total"`
	Score     int          `json:"score"`
	MinDevice int          `json:"min_devices"`
	ByKind    []KindImpact `json:"by_kind"`
	ByDomain  []struct {
		Domain string `json:"domain"`
		WorkStat
		Score int `json:"score"`
	} `json:"by_domain"`
}

// NodeImpact dampak satu node; Available=false bila belum cukup perangkat untuk ditampilkan.
type NodeImpact struct {
	Available  bool `json:"available"`
	MinDevices int  `json:"min_devices"`
	Node       *struct {
		Key string `json:"key"`
		WorkStat
		Score  int          `json:"score"`
		ByKind []KindImpact `json:"by_kind"`
	} `json:"node,omitempty"`
}

// Cache di sisi engine: Studio boleh membuka halaman berkali-kali, server cukup ditanya sesekali.
const (
	impactTTL      = 10 * time.Minute
	impactMaxNodes = 500
)

type cacheEntry[T any] struct {
	at  time.Time
	val T
}

var impactCache struct {
	mu      sync.Mutex
	summary *cacheEntry[*Impact]
	nodes   map[string]cacheEntry[*NodeImpact]
}

// FetchImpact mengambil ringkasan dampak seluruh pengguna (di-cache 10 menit; bila server gagal, data lama dipakai).
func FetchImpact() (*Impact, error) {
	c := &impactCache
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.summary != nil && time.Since(c.summary.at) < impactTTL {
		return c.summary.val, nil
	}
	var imp Impact
	if err := get("/api/v1/knowledge/impact", &imp); err != nil {
		if c.summary != nil {
			return c.summary.val, nil
		}
		return nil, err
	}
	c.summary = &cacheEntry[*Impact]{at: time.Now(), val: &imp}
	return &imp, nil
}

// FetchNodeImpact mengambil dampak satu node (kunci server, tanpa awalan "hub:"), di-cache per node.
func FetchNodeImpact(key string) (*NodeImpact, error) {
	key = strings.TrimPrefix(strings.TrimSpace(key), "hub:")
	if key == "" || len(key) > 191 {
		return nil, errors.New("kunci node tidak valid")
	}
	c := &impactCache
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.nodes[key]; ok && time.Since(e.at) < impactTTL {
		return e.val, nil
	}
	var n NodeImpact
	if err := get("/api/v1/knowledge/impact?key="+url.QueryEscape(key), &n); err != nil {
		if e, ok := c.nodes[key]; ok {
			return e.val, nil
		}
		return nil, err
	}
	if c.nodes == nil || len(c.nodes) >= impactMaxNodes {
		c.nodes = map[string]cacheEntry[*NodeImpact]{}
	}
	c.nodes[key] = cacheEntry[*NodeImpact]{at: time.Now(), val: &n}
	return &n, nil
}
