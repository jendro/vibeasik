package knowledgesync

import (
	"strings"

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

// Impact dampak teragregasi dari server (lihat vibeasik-cloud/internal/knowledgehub/impact.go).
type Impact struct {
	Devices   int      `json:"devices"`
	Total     WorkStat `json:"total"`
	Score     int      `json:"score"`
	MinDevice int      `json:"min_devices"`
	ByKind    []struct {
		Kind  string `json:"kind"`
		Label string `json:"label"`
		WorkStat
		Score int `json:"score"`
	} `json:"by_kind"`
	ByDomain []struct {
		Domain string `json:"domain"`
		WorkStat
		Score int `json:"score"`
	} `json:"by_domain"`
	Nodes []struct {
		Key string `json:"key"`
		WorkStat
		Score  int `json:"score"`
		ByKind []struct {
			Kind  string `json:"kind"`
			Label string `json:"label"`
			WorkStat
			Score int `json:"score"`
		} `json:"by_kind"`
	} `json:"nodes"`
}

// FetchImpact mengambil dampak pengetahuan seluruh pengguna dari server.
func FetchImpact() (*Impact, error) {
	var imp Impact
	if err := get("/api/v1/knowledge/impact", &imp); err != nil {
		return nil, err
	}
	return &imp, nil
}
