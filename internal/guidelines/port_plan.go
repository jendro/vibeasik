package guidelines

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// PortPlan = rencana blok port satu proyek dari registri PORTS.md (Aturan 35).
type PortPlan struct {
	Slot       int          // -1 = proyek milik vibeasik sendiri (87xx, Aturan 8)
	Block      string       // mis. "4120–4129"
	Components map[uint]int // component_id → port rencana
}

// PortRegistry = isi PORTS.md: rencana per project_id + slot bebas berikutnya.
type PortRegistry struct {
	Path     string
	Projects map[uint]PortPlan
	NextSlot int
}

var (
	portRowProject = regexp.MustCompile(`\(#(\d+)\)`)
	portRowComp    = regexp.MustCompile(`\(#(\d+)[,)][^→]*→ \*\*(\d+)\*\*`)
	portNextSlot   = regexp.MustCompile("slot bebas terkecil berikutnya[^`]*`(\\d+)`")
)

// FindPortsRegistry mencari PORTS.md di sebelah folder guidelines (repo) atau di ~/.vibeasik/.
func FindPortsRegistry() string {
	var candidates []string
	if dir := FindGuidelinesDir(); dir != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(dir), "PORTS.md"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".vibeasik", "PORTS.md"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// LoadPortRegistry membaca tabel "Daftar Slot" PORTS.md; nil bila berkas tidak ada.
func LoadPortRegistry(path string) *PortRegistry {
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return ParsePortRegistry(path, string(raw))
}

// ParsePortRegistry mengurai isi PORTS.md (dipisah dari LoadPortRegistry agar mudah dites).
func ParsePortRegistry(path, content string) *PortRegistry {
	reg := &PortRegistry{Path: path, Projects: map[uint]PortPlan{}, NextSlot: -1}
	if m := portNextSlot.FindStringSubmatch(content); m != nil {
		reg.NextSlot, _ = strconv.Atoi(m[1])
	}
	for _, line := range strings.Split(content, "\n") {
		cols := strings.Split(line, "|")
		if len(cols) < 6 {
			continue
		}
		slotCol, block, project, comps := strings.TrimSpace(cols[1]), strings.TrimSpace(cols[2]), cols[3], cols[4]
		pm := portRowProject.FindStringSubmatch(project)
		if pm == nil {
			continue
		}
		slot, err := strconv.Atoi(slotCol)
		if err != nil {
			if slotCol != "—" && slotCol != "-" {
				continue
			}
			slot = -1
		}
		plan := PortPlan{Slot: slot, Block: block, Components: map[uint]int{}}
		for _, cm := range portRowComp.FindAllStringSubmatch(comps, -1) {
			id, _ := strconv.Atoi(cm[1])
			port, _ := strconv.Atoi(cm[2])
			plan.Components[uint(id)] = port
		}
		id, _ := strconv.Atoi(pm[1])
		reg.Projects[uint(id)] = plan
	}
	return reg
}
