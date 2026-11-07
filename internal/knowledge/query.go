package knowledge

import (
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
)

// Overview merangkum isi Pusat Pengetahuan untuk studio & MCP.
type Overview struct {
	Total       int                       `json:"total"`
	Shared      int                       `json:"shared"`
	Personal    int                       `json:"personal"`
	ByKind      map[string]int            `json:"by_kind"`
	BySource    map[string]int            `json:"by_source"`
	Domains     []DomainStat              `json:"domains"`
	Clusters    []models.KnowledgeCluster `json:"clusters"`
	Gaps        []models.KnowledgeGap     `json:"gaps"`
	NeedsReview []models.KnowledgeNode    `json:"needs_review"`
	Tested      int                       `json:"tested"`  // node yang sudah punya bukti helpful/harmful
	Utility     float64                   `json:"utility"` // rata-rata utilitas node teruji
	Links       int64                     `json:"links"`
	UsesPending int64                     `json:"uses_pending"`
}

// DomainStat: jumlah node aktif per domain.
type DomainStat struct {
	Domain    string `json:"domain"`
	Nodes     int    `json:"nodes"`
	Ecosystem bool   `json:"ecosystem"`
}

// GetOverview menghitung ringkasan pusat.
func GetOverview(db *gorm.DB) Overview {
	var rows []models.KnowledgeNode
	db.Select("id", "kind", "source", "domain", "owner_employee_id", "helpful", "harmful").Where("status = ?", models.KnowledgeNodeActive).Find(&rows)
	o := Overview{Total: len(rows), ByKind: map[string]int{}, BySource: map[string]int{}}
	domains := map[string]int{}
	util := 0.0
	for _, r := range rows {
		o.ByKind[r.Kind]++
		o.BySource[r.Source]++
		if r.OwnerEmployeeID == nil {
			o.Shared++
		} else {
			o.Personal++
		}
		if r.Domain != "" {
			domains[r.Domain]++
		}
		if r.Helpful+r.Harmful > 0 {
			o.Tested++
			util += r.Utility()
		}
	}
	if o.Tested > 0 {
		o.Utility = round3(util / float64(o.Tested))
	}
	for _, d := range sortedKeys(domains) {
		o.Domains = append(o.Domains, DomainStat{Domain: d, Nodes: domains[d], Ecosystem: IsEcosystem(d)})
	}
	db.Order("size DESC, id ASC").Limit(60).Find(&o.Clusters)
	o.Gaps = ListGaps(db, "", 50)
	db.Where("status = ? AND needs_review = ?", models.KnowledgeNodeActive, true).Order("harmful DESC").Limit(30).Find(&o.NeedsReview)
	db.Model(&models.KnowledgeLink{}).Count(&o.Links)
	db.Model(&models.KnowledgeUse{}).Where("outcome = ?", "").Count(&o.UsesPending)
	return o
}

// Filter untuk daftar node.
type Filter struct {
	Text      string
	Domain    string
	Kind      string
	Source    string
	ClusterID uint
	Status    string
	Limit     int
}

// List mengembalikan node sesuai filter. Dengan Text, urutannya memakai Recall (skor lengkap, tanpa mencatat pemakaian).
func List(db *gorm.DB, f Filter) []Hit {
	if f.Limit <= 0 || f.Limit > 300 {
		f.Limit = 100
	}
	if strings.TrimSpace(f.Text) != "" && (f.Status == "" || f.Status == models.KnowledgeNodeActive) {
		q := Query{Text: f.Text, Limit: 50, EmployeeID: 0}
		if f.Domain != "" {
			q.Domains = []string{f.Domain}
		}
		if f.Kind != "" {
			q.Kinds = []string{f.Kind}
		}
		hits := Recall(db, q).Hits
		out := hits[:0]
		for _, h := range hits {
			if (f.Source == "" || h.Source == f.Source) && (f.ClusterID == 0 || h.ClusterID == f.ClusterID) {
				out = append(out, h)
			}
		}
		return out
	}
	q := db.Order("weight + 5 * (helpful - harmful) DESC, id DESC").Limit(f.Limit)
	q = q.Where("status = ?", firstNonEmpty(f.Status, models.KnowledgeNodeActive))
	if f.Domain != "" {
		q = q.Where("domain = ?", f.Domain)
	}
	if f.Kind != "" {
		q = q.Where("kind = ?", f.Kind)
	}
	if f.Source != "" {
		q = q.Where("source = ?", f.Source)
	}
	if f.ClusterID > 0 {
		q = q.Where("cluster_id = ?", f.ClusterID)
	}
	var nodes []models.KnowledgeNode
	q.Find(&nodes)
	out := make([]Hit, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, Hit{KnowledgeNode: n, UtilityV: round3(n.Utility())})
	}
	return out
}

// Detail: satu node beserta tetangga bertautnya & riwayat pemakaian.
type Detail struct {
	Node      models.KnowledgeNode   `json:"node"`
	Utility   float64                `json:"utility"`
	Neighbors []Neighbor             `json:"neighbors"`
	Uses      []models.KnowledgeUse  `json:"uses"`
	MergedTo  *models.KnowledgeNode  `json:"merged_to,omitempty"`
	Merged    []models.KnowledgeNode `json:"merged,omitempty"`
}

// Neighbor: node tetangga + kekuatan tautannya.
type Neighbor struct {
	models.KnowledgeNode
	Strength float64 `json:"strength"`
}

// GetDetail memuat detail node.
func GetDetail(db *gorm.DB, id uint) (*Detail, error) {
	var n models.KnowledgeNode
	if err := db.First(&n, id).Error; err != nil {
		return nil, err
	}
	d := &Detail{Node: n, Utility: round3(n.Utility())}
	var links []models.KnowledgeLink
	db.Where("from_id = ? OR to_id = ?", id, id).Order("strength DESC").Limit(12).Find(&links)
	for _, l := range links {
		other := l.ToID
		if other == id {
			other = l.FromID
		}
		var o models.KnowledgeNode
		if db.First(&o, other).Error == nil {
			d.Neighbors = append(d.Neighbors, Neighbor{KnowledgeNode: o, Strength: l.Strength})
		}
	}
	db.Where("node_id = ?", id).Order("id DESC").Limit(30).Find(&d.Uses)
	if n.MergedIntoID != nil {
		var m models.KnowledgeNode
		if db.First(&m, *n.MergedIntoID).Error == nil {
			d.MergedTo = &m
		}
	}
	db.Where("merged_into_id = ?", id).Order("id DESC").Limit(20).Find(&d.Merged)
	return d, nil
}

// GraphNode: node ringkas untuk visualisasi jaring Pusat Pengetahuan (tanpa isi lengkap).
type GraphNode struct {
	ID        uint    `json:"id"`
	Title     string  `json:"title"`
	Kind      string  `json:"kind"`
	Domain    string  `json:"domain"`
	Source    string  `json:"source"`
	ClusterID uint    `json:"cluster_id"`
	Personal  bool    `json:"personal"`
	Weight    int     `json:"weight"`
	Helpful   int     `json:"helpful"`
	Harmful   int     `json:"harmful"`
	Uses      int     `json:"uses"`
	Utility   float64 `json:"utility"`
}

// GraphLink: tautan kemiripan antar dua node aktif.
type GraphLink struct {
	Source   uint    `json:"source"`
	Target   uint    `json:"target"`
	Strength float64 `json:"strength"`
}

// Graph: seluruh node aktif + tautannya, untuk peta jaring di studio.
type Graph struct {
	Nodes    []GraphNode               `json:"nodes"`
	Links    []GraphLink               `json:"links"`
	Clusters []models.KnowledgeCluster `json:"clusters"`
}

// GetGraph memuat jaring pengetahuan: node aktif (dibatasi limit, yang terkuat didahulukan) dan tautan di antaranya.
func GetGraph(db *gorm.DB, limit int) Graph {
	if limit <= 0 || limit > 5000 {
		limit = 800
	}
	var rows []models.KnowledgeNode
	db.Select("id", "title", "kind", "domain", "source", "cluster_id", "owner_employee_id", "weight", "helpful", "harmful", "uses").
		Where("status = ?", models.KnowledgeNodeActive).
		Order("weight + 5 * (helpful - harmful) + uses DESC, id ASC").Limit(limit).Find(&rows)
	g := Graph{Nodes: make([]GraphNode, 0, len(rows)), Links: []GraphLink{}}
	in := make(map[uint]bool, len(rows))
	for _, r := range rows {
		in[r.ID] = true
		g.Nodes = append(g.Nodes, GraphNode{
			ID: r.ID, Title: r.Title, Kind: r.Kind, Domain: r.Domain, Source: r.Source, ClusterID: r.ClusterID,
			Personal: r.OwnerEmployeeID != nil, Weight: r.Weight, Helpful: r.Helpful, Harmful: r.Harmful, Uses: r.Uses,
			Utility: round3(r.Utility()),
		})
	}
	var links []models.KnowledgeLink
	db.Find(&links)
	for _, l := range links {
		if in[l.FromID] && in[l.ToID] {
			g.Links = append(g.Links, GraphLink{Source: l.FromID, Target: l.ToID, Strength: l.Strength})
		}
	}
	db.Order("size DESC, id ASC").Find(&g.Clusters)
	return g
}
