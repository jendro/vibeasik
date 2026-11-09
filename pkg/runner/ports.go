package runner

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Kebijakan port vibeasik (port authority lokal, Aturan 35/45):
//  1. Setiap proyek punya blok 10 port tetap (slots.go); run_port komponen = port rencana di bloknya,
//     sehingga semua proyek bisa dijalankan serentak tanpa bentrok.
//  2. Port rencana dipegang proses lain → offset kosong lain di blok yang sama, lalu cadangan 5000–5999.
//  3. Port bawaan framework (3000/8080) tidak dipakai sebagai run_port.
//
// Port yang sedang dipakai proses lain tidak pernah dibunuh — cukup dilewati.
const (
	RangeStart = 4100
	RangeEnd   = 4999
)

// reservedPorts adalah port milik vibeasik sendiri (studio dev, engine, cloud, System One).
var reservedPorts = map[int]bool{8750: true, 8760: true, 8765: true, 8770: true}

func init() {
	// Port vibeasik yang di-override lewat env (dev.sh / .env) juga tidak boleh dialokasikan ke proyek.
	for _, key := range []string{"VIBEASIK_PORT", "VIBEASIK_STUDIO_PORT", "VIBEASIK_ENGINE_PORT", "VIBEASIK_CLOUD_PORT"} {
		if p, err := strconv.Atoi(os.Getenv(key)); err == nil && p > 0 {
			reservedPorts[p] = true
		}
	}
}

// skipInRange adalah port default tool populer di dalam rentang alokasi (Angular, Vite preview, Astro, dll).
var skipInRange = map[int]bool{4173: true, 4200: true, 4242: true, 4321: true, 4443: true, 4444: true, 4567: true, 4646: true, 4848: true}

var allocMu sync.Mutex

const listenProbeTimeout = 50 * time.Millisecond

// IsListening reports whether something accepts TCP connections on 127.0.0.1:port.
func IsListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), listenProbeTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// IsPortFree reports whether the port can be bound right now (nothing listening on loopback or wildcard).
func IsPortFree(port int) bool {
	if port <= 0 || port > 65535 || IsListening(port) {
		return false
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// IsReserved reports whether the port belongs to vibeasik itself.
func IsReserved(port int) bool {
	return reservedPorts[port]
}

// claimedPorts returns run ports registered to every component except excludeID.
func claimedPorts(db *gorm.DB, excludeID uint) (map[int]uint, error) {
	var rows []models.Component
	if err := db.Select("id", "run_port").Where("id <> ? AND run_port > 0", excludeID).Find(&rows).Error; err != nil {
		return nil, err
	}
	claimed := make(map[int]uint, len(rows))
	for _, r := range rows {
		claimed[r.RunPort] = r.ID
	}
	return claimed, nil
}

func ensureComponentSlot(db *gorm.DB, comp *models.Component) ([]int, error) {
	var slot []int
	db.Model(&models.Project{}).Where("id = ? AND port_slot IS NOT NULL", comp.ProjectID).Pluck("port_slot", &slot)
	if len(slot) == 0 && comp.ProjectID > 0 {
		if _, err := EnsureProjectPorts(db, comp.ProjectID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		db.Model(&models.Component{}).Where("id = ?", comp.ID).Pluck("run_port", &comp.RunPort)
	}
	return slot, nil
}

func isPortUsable(p int, currentPort int, claimed map[int]uint) bool {
	if p < 1024 || p > 65535 || p == currentPort || reservedPorts[p] {
		return false
	}
	if _, taken := claimed[p]; taken {
		return false
	}
	return IsPortFree(p)
}

func findStablePort(comp models.Component, reassign bool, claimed map[int]uint) int {
	if reassign || comp.RunPort <= 0 {
		return 0
	}
	if _, taken := claimed[comp.RunPort]; !taken && !reservedPorts[comp.RunPort] {
		return comp.RunPort
	}
	return 0
}

func findFirstUsablePort(start, end int, currentPort int, claimed map[int]uint) int {
	for p := start; p <= end; p++ {
		if isPortUsable(p, currentPort, claimed) {
			return p
		}
	}
	return 0
}

func chooseRunPort(comp models.Component, reassign bool, planned int, slot []int, claimed map[int]uint) (int, error) {
	if port := findStablePort(comp, reassign, claimed); port > 0 {
		return port, nil
	}
	if planned > 0 && isPortUsable(planned, comp.RunPort, claimed) {
		return planned, nil
	}
	if len(slot) > 0 && slot[0] >= 0 {
		base := slotBase(slot[0])
		if port := findFirstUsablePort(base, base+SlotSize-1, comp.RunPort, claimed); port > 0 {
			return port, nil
		}
	}
	if port := findFirstUsablePort(OverflowStart, OverflowEnd, comp.RunPort, claimed); port > 0 {
		return port, nil
	}
	return 0, fmt.Errorf("tidak ada port bebas di blok proyek maupun rentang cadangan %d-%d", OverflowStart, OverflowEnd)
}

func persistRunPort(db *gorm.DB, comp *models.Component, chosen int) error {
	if chosen == comp.RunPort {
		return nil
	}
	if err := db.Model(&models.Component{}).Where("id = ?", comp.ID).Update("run_port", chosen).Error; err != nil {
		return err
	}
	comp.RunPort = chosen
	return nil
}

// Allocate picks and persists the run port for comp. With reassign=true the current RunPort is
// abandoned (e.g. because a foreign process holds it) and a different free port is chosen.
func Allocate(db *gorm.DB, comp *models.Component, reassign bool) (int, error) {
	slot, err := ensureComponentSlot(db, comp)
	if err != nil {
		return 0, err
	}
	planned := PlannedPort(db, *comp)

	allocMu.Lock()
	defer allocMu.Unlock()

	claimed, err := claimedPorts(db, comp.ID)
	if err != nil {
		return 0, err
	}

	chosen, err := chooseRunPort(*comp, reassign, planned, slot, claimed)
	if err != nil {
		return 0, err
	}

	if err := persistRunPort(db, comp, chosen); err != nil {
		return 0, err
	}
	return chosen, nil
}


// Assign sets a user-chosen run port after checking it is not owned by vibeasik or another component.
func Assign(db *gorm.DB, comp *models.Component, port int) error {
	allocMu.Lock()
	defer allocMu.Unlock()

	if port < 1024 || port > 65535 {
		return fmt.Errorf("port harus di antara 1024 dan 65535")
	}
	if reservedPorts[port] {
		return fmt.Errorf("port %d dipakai oleh vibeasik sendiri", port)
	}
	claimed, err := claimedPorts(db, comp.ID)
	if err != nil {
		return err
	}
	if owner, taken := claimed[port]; taken {
		var other models.Component
		db.Select("id", "name").First(&other, owner)
		return fmt.Errorf("port %d sudah dialokasikan ke komponen '%s'", port, other.Name)
	}
	if err := db.Model(&models.Component{}).Where("id = ?", comp.ID).Update("run_port", port).Error; err != nil {
		return err
	}
	comp.RunPort = port
	return nil
}

// EnsureUniquePorts memastikan setiap komponen proyek (atau semua proyek bila projectID == 0) memakai
// port rencana blok slot proyeknya (Aturan 35/45), jadi unik di dalam proyek maupun lintas proyek.
func EnsureUniquePorts(db *gorm.DB, projectID uint) ([]models.Component, error) {
	if projectID == 0 {
		return EnsureAllProjectPorts(db)
	}
	return EnsureProjectPorts(db, projectID)
}
