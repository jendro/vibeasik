package guidelines

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// PortRow = satu baris tabel "Daftar Slot" PORTS.md yang ditulis engine dari DB (Aturan 35/45).
type PortRow struct {
	Slot        int
	ProjectID   uint
	ProjectName string
	Comps       []PortRowComp
}

// PortRowComp = komponen dalam blok: port rencana & run_port sekarang.
type PortRowComp struct {
	ID      uint
	Name    string
	Role    string
	Port    int // port rencana; 0 = blok penuh (tidak dapat offset)
	RunPort int
}

var (
	nextSlotText = regexp.MustCompile("(slot bebas terkecil berikutnya[^`]*`)(\\d+)(`)")
	statusNote   = "> Status: **dikelola otomatis oleh engine** (Aturan 35/45). Slot disimpan di DB (`projects.port_slot`), `run_port` tiap komponen\n" +
		"> mengikuti port rencana dan tabel ini ditulis ulang setiap kali ada perubahan. Jangan mengedit tabel secara manual."
)

// WritePortRegistry menulis ulang tabel "Daftar Slot" di PORTS.md. Baris proyek yang tidak ada di rows (proyek
// terhapus/pensiun, baris vibeasik "—") dipertahankan apa adanya agar slot tidak pernah dipakai ulang.
func WritePortRegistry(path string, rows []PortRow, nextSlot int) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	given := map[uint]bool{}
	for _, r := range rows {
		given[r.ProjectID] = true
	}

	lines := strings.Split(string(raw), "\n")
	type tableLine struct {
		slot int // -1 untuk "—", diletakkan di akhir
		text string
	}
	var out, kept []string
	var table []tableLine
	inTable, tableAt := false, -1
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "> Status:") {
			out = append(out, statusNote)
			for i+1 < len(lines) && strings.HasPrefix(lines[i+1], ">") {
				i++
			}
			continue
		}
		if strings.HasPrefix(line, "| Slot |") {
			inTable = true
			out = append(out, line)
			if i+1 < len(lines) {
				i++
				out = append(out, lines[i]) // pemisah header
			}
			tableAt = len(out)
			continue
		}
		if inTable {
			if !strings.HasPrefix(line, "|") {
				inTable = false
				out = append(out, line)
				continue
			}
			cols := strings.Split(line, "|")
			if len(cols) < 6 {
				continue
			}
			if pm := portRowProject.FindStringSubmatch(cols[3]); pm != nil {
				id, _ := strconv.Atoi(pm[1])
				if given[uint(id)] {
					continue
				}
			}
			slot, err := strconv.Atoi(strings.TrimSpace(cols[1]))
			if err != nil {
				slot = -1
			}
			table = append(table, tableLine{slot, line})
			continue
		}
		out = append(out, line)
	}
	if tableAt < 0 {
		return fmt.Errorf("tabel Daftar Slot tidak ditemukan di %s", path)
	}
	for _, r := range rows {
		table = append(table, tableLine{r.Slot, renderPortRow(r)})
	}
	sort.SliceStable(table, func(i, j int) bool {
		a, b := table[i].slot, table[j].slot
		if (a < 0) != (b < 0) {
			return b < 0
		}
		return a < b
	})
	for _, t := range table {
		kept = append(kept, t.text)
	}
	out = append(out[:tableAt], append(kept, out[tableAt:]...)...)
	content := nextSlotText.ReplaceAllString(strings.Join(out, "\n"), "${1}"+strconv.Itoa(nextSlot)+"${3}")
	if content == string(raw) {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func renderPortRow(r PortRow) string {
	base := 4100 + r.Slot*10
	var parts []string
	for _, c := range r.Comps {
		if c.Port == 0 {
			parts = append(parts, fmt.Sprintf("%s (#%d, %s) → blok penuh, butuh slot kedua", c.Name, c.ID, c.Role))
			continue
		}
		s := fmt.Sprintf("%s (#%d, %s) → **%d**", c.Name, c.ID, c.Role, c.Port)
		if c.RunPort != c.Port {
			now := "belum ada"
			if c.RunPort > 0 {
				now = strconv.Itoa(c.RunPort)
			}
			s += fmt.Sprintf(" _(sekarang %s)_", now)
		}
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		parts = append(parts, "_(belum ada komponen)_")
	}
	return fmt.Sprintf("| %d | %d–%d | %s (#%d) | %s |", r.Slot, base, base+9, r.ProjectName, r.ProjectID, strings.Join(parts, "<br>"))
}
