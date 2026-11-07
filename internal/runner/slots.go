package runner

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/guidelines"
	"github.com/vibeasik/vibeasik/internal/models"
)

// Blok port per proyek (Aturan 35/45): slot N = RangeStart+N×10 .. +9, satu proyek satu slot selamanya.
// +0..+4 frontend, +5..+9 backend/service. Slot disimpan di projects.port_slot (-1 = vibeasik sendiri, 87xx);
// PORTS.md ditulis ulang dari DB. Port bawaan framework (3000/8080) tidak lagi dipakai sebagai run_port.
const (
	SlotSize = 10
	MaxSlot  = (RangeEnd-RangeStart+1)/SlotSize - 1
	// OverflowStart–OverflowEnd: cadangan untuk komponen yang tidak mendapat offset (blok penuh) atau
	// port rencananya dipegang proses lain; di luar semua blok agar tidak memakan slot proyek lain.
	OverflowStart = 5000
	OverflowEnd   = 5999
)

// RegistryPath menunjuk PORTS.md. Di bawah `go test` selalu kosong agar tes paket mana pun tidak pernah
// menulis registri asli repo (tes yang menguji penulisan memakai berkas sementara).
var RegistryPath = func() string {
	if testing.Testing() {
		return ""
	}
	return guidelines.FindPortsRegistry()
}

func slotBase(slot int) int { return RangeStart + slot*SlotSize }

// SlotForbidden: blok yang berisi port tool populer (skipInRange) tidak dipakai.
func SlotForbidden(slot int) bool {
	for p := slotBase(slot); p < slotBase(slot)+SlotSize; p++ {
		if skipInRange[p] {
			return true
		}
	}
	return false
}

func frontSide(c models.Component) bool { return strings.HasPrefix(c.Role, "frontend") }

func roleRank(c models.Component) int {
	switch c.Role {
	case models.RoleFrontendClient, models.RoleBackend:
		return 0
	case models.RoleFrontendAdmin, models.RoleWorker:
		return 1
	case models.RoleFrontendMobile:
		return 2
	}
	return 3
}

// PlanSlotPorts membagi offset blok ke komponen. keep = port rencana lama (PORTS.md / run_port di dalam blok)
// yang dipertahankan agar URL tidak berpindah-pindah; sisanya diberi offset kosong pertama di sisinya,
// lalu sisi lain bila sisinya penuh. Komponen tanpa offset tidak masuk hasil (blok penuh).
func PlanSlotPorts(slot int, comps []models.Component, keep map[uint]int) map[uint]int {
	base := slotBase(slot)
	plan, used := map[uint]int{}, map[int]bool{}
	var rest []models.Component
	for _, c := range comps {
		if p, ok := keep[c.ID]; ok && p >= base && p < base+SlotSize && !used[p] {
			plan[c.ID], used[p] = p, true
			continue
		}
		rest = append(rest, c)
	}
	sort.SliceStable(rest, func(i, j int) bool {
		a, b := rest[i], rest[j]
		if frontSide(a) != frontSide(b) {
			return frontSide(a)
		}
		if roleRank(a) != roleRank(b) {
			return roleRank(a) < roleRank(b)
		}
		return a.ID < b.ID
	})
	for _, c := range rest {
		own, other := []int{0, 1, 2, 3, 4}, []int{5, 6, 7, 8, 9}
		if !frontSide(c) {
			own, other = other, own
		}
		for _, off := range append(own, other...) {
			if p := base + off; !used[p] {
				plan[c.ID], used[p] = p, true
				break
			}
		}
	}
	return plan
}

// registryKeep membaca PORTS.md sekali (boleh nil bila berkas tidak ada).
func loadRegistry() *guidelines.PortRegistry {
	return guidelines.LoadPortRegistry(RegistryPath())
}

// nextFreeSlot = slot terbesar yang pernah dipakai (DB atau PORTS.md) + 1, melewati slot terlarang.
// Slot tidak dipakai ulang walau proyeknya dihapus.
func nextFreeSlot(db *gorm.DB, reg *guidelines.PortRegistry) int {
	maxUsed := -1
	var slots []int
	db.Model(&models.Project{}).Where("port_slot IS NOT NULL").Pluck("port_slot", &slots)
	for _, s := range slots {
		maxUsed = max(maxUsed, s)
	}
	if reg != nil {
		for _, p := range reg.Projects {
			maxUsed = max(maxUsed, p.Slot)
		}
		maxUsed = max(maxUsed, reg.NextSlot-1)
	}
	for s := maxUsed + 1; s <= MaxSlot; s++ {
		if !SlotForbidden(s) {
			return s
		}
	}
	return -2
}

// ensureSlot memberi proyek slot tetap: yang tersimpan di DB, lalu yang tercatat di PORTS.md, lalu slot baru.
func ensureSlot(db *gorm.DB, p *models.Project, reg *guidelines.PortRegistry) (int, error) {
	if p.PortSlot != nil {
		return *p.PortSlot, nil
	}
	slot := -2
	if reg != nil {
		if plan, ok := reg.Projects[p.ID]; ok {
			slot = plan.Slot
		}
	}
	if slot == -2 {
		if slot = nextFreeSlot(db, reg); slot == -2 {
			return 0, fmt.Errorf("semua slot port 4100–4999 sudah terpakai")
		}
	}
	if err := db.Model(&models.Project{}).Where("id = ?", p.ID).Update("port_slot", slot).Error; err != nil {
		return 0, err
	}
	p.PortSlot = &slot
	return slot, nil
}

// EnsureProjectPorts memastikan proyek punya slot dan setiap komponennya memakai port rencana di bloknya.
// Komponen lain (proyek lain) yang masih memegang port rencana dilepas (run_port=0) dan mendapat port
// rencananya sendiri saat proyeknya dipastikan / dijalankan. PORTS.md ditulis ulang bila ada.
func EnsureProjectPorts(db *gorm.DB, projectID uint) ([]models.Component, error) {
	changed, err := ensureProjectPorts(db, projectID, loadRegistry())
	if err == nil && len(changed) > 0 {
		_ = SyncPortsRegistry(db)
	}
	return changed, err
}

// EnsureAllProjectPorts menjalankan EnsureProjectPorts untuk semua proyek (migrasi saat engine start).
func EnsureAllProjectPorts(db *gorm.DB) ([]models.Component, error) {
	var ids []uint
	if err := db.Model(&models.Project{}).Order("id ASC").Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	reg := loadRegistry()
	var all []models.Component
	for _, id := range ids {
		changed, err := ensureProjectPorts(db, id, reg)
		if err != nil {
			return all, err
		}
		all = append(all, changed...)
	}
	_ = SyncPortsRegistry(db)
	return all, nil
}

func ensureProjectPorts(db *gorm.DB, projectID uint, reg *guidelines.PortRegistry) ([]models.Component, error) {
	allocMu.Lock()
	defer allocMu.Unlock()

	var p models.Project
	if err := db.Preload("Components", func(q *gorm.DB) *gorm.DB { return q.Order("id ASC") }).First(&p, projectID).Error; err != nil {
		return nil, err
	}
	slot, err := ensureSlot(db, &p, reg)
	if err != nil || slot < 0 {
		return nil, err
	}
	plan := PlanSlotPorts(slot, p.Components, keepPorts(p, reg))

	var changed []models.Component
	for _, c := range p.Components {
		want, ok := plan[c.ID]
		if !ok || c.RunPort == want {
			continue
		}
		// Port rencana masih dipegang komponen proyek lain (sisa alokasi lama): lepaskan dulu.
		if err := db.Model(&models.Component{}).Where("run_port = ? AND id <> ?", want, c.ID).Update("run_port", 0).Error; err != nil {
			return changed, err
		}
		if err := db.Model(&models.Component{}).Where("id = ?", c.ID).Update("run_port", want).Error; err != nil {
			return changed, err
		}
		c.RunPort = want
		changed = append(changed, c)
	}
	return changed, nil
}

// keepPorts: port rencana di PORTS.md, atau run_port yang sudah berada di blok proyek.
func keepPorts(p models.Project, reg *guidelines.PortRegistry) map[uint]int {
	keep := map[uint]int{}
	for _, c := range p.Components {
		if c.RunPort > 0 {
			keep[c.ID] = c.RunPort
		}
	}
	if reg != nil {
		for id, port := range reg.Projects[p.ID].Components {
			keep[id] = port
		}
	}
	return keep
}

// PlannedPort mengembalikan port rencana komponen dari slot proyeknya (0 bila belum ada slot / blok penuh).
func PlannedPort(db *gorm.DB, comp models.Component) int {
	var p models.Project
	if err := db.Preload("Components", func(q *gorm.DB) *gorm.DB { return q.Order("id ASC") }).First(&p, comp.ProjectID).Error; err != nil || p.PortSlot == nil || *p.PortSlot < 0 {
		return 0
	}
	return PlanSlotPorts(*p.PortSlot, p.Components, keepPorts(p, nil))[comp.ID]
}

// SyncPortsRegistry menulis ulang tabel PORTS.md dari DB (tidak melakukan apa pun bila berkas tidak ada).
func SyncPortsRegistry(db *gorm.DB) error {
	path := RegistryPath()
	if path == "" {
		return nil
	}
	var projects []models.Project
	if err := db.Preload("Components", func(q *gorm.DB) *gorm.DB { return q.Order("id ASC") }).
		Where("port_slot IS NOT NULL AND port_slot >= 0").Order("port_slot ASC").Find(&projects).Error; err != nil {
		return err
	}
	var rows []guidelines.PortRow
	for _, p := range projects {
		plan := PlanSlotPorts(*p.PortSlot, p.Components, keepPorts(p, nil))
		row := guidelines.PortRow{Slot: *p.PortSlot, ProjectID: p.ID, ProjectName: p.Name}
		comps := append([]models.Component(nil), p.Components...)
		order := func(c models.Component) int { // komponen tanpa offset (blok penuh) di akhir
			if plan[c.ID] == 0 {
				return 1 << 30
			}
			return plan[c.ID]
		}
		sort.SliceStable(comps, func(i, j int) bool { return order(comps[i]) < order(comps[j]) })
		for _, c := range comps {
			row.Comps = append(row.Comps, guidelines.PortRowComp{ID: c.ID, Name: c.Name, Role: c.Role, Port: plan[c.ID], RunPort: c.RunPort})
		}
		rows = append(rows, row)
	}
	return guidelines.WritePortRegistry(path, rows, nextFreeSlot(db, loadRegistry()))
}
