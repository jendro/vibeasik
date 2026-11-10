package traffic

import (
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Jenis event.
const (
	KindRequest   = "request"
	KindError     = "error"
	KindHeartbeat = "heartbeat"

	SideServer  = "server"
	SideBrowser = "browser"

	maxErrType = 128
	maxErrMsg  = 500
	maxStack   = 8000
	maxRelease = 64
)

var (
	reAddr      = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	reOrigin    = regexp.MustCompile(`https?://[^/\s)]+`)
	reLineCol   = regexp.MustCompile(`(:\d+){1,2}\b|[, ]+line \d+`)
	reBundle    = regexp.MustCompile(`[._-][0-9a-fA-F]{6,}(\.(?:m?js|cjs|css|ts))`)
	reAbsPath   = regexp.MustCompile(`(?:[A-Za-z]:)?[\\/][^\s()]+`)
	reDigits    = regexp.MustCompile(`\d+`)
	libMarkers  = []string{"/node_modules/", "/vendor/", "/site-packages/", "/dist-packages/"}
	noiseFrames = []string{"vibeasik_traffic", "vibeasiktraffic", "VibeasikTraffic", "vibeasik-traffic", "node:internal", "vibeasik-errors"}
)

// IsSpecialKind = event bukan request (tidak dihitung di statistik rute/dimensi).
func IsSpecialKind(kind string) bool { return kind == KindError || kind == KindHeartbeat }

// relativePath mengubah path absolut jadi relatif: ke penanda pustaka bila ada, kalau tidak dua segmen terakhir.
func relativePath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	for _, m := range libMarkers {
		if i := strings.LastIndex(p, m); i >= 0 {
			return p[i+1:]
		}
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return strings.Join(parts, "/")
}

// NormalizeFrame membuang nomor baris/kolom, alamat memori, hash bundel, dan origin URL; path absolut jadi relatif.
func NormalizeFrame(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "at ")
	line = strings.ReplaceAll(line, "file://", "")
	line = reAddr.ReplaceAllString(line, "0x?")
	line = reOrigin.ReplaceAllString(line, "")
	line = reLineCol.ReplaceAllString(line, "")
	line = reBundle.ReplaceAllString(line, "$1")
	line = reAbsPath.ReplaceAllStringFunc(line, relativePath)
	return strings.Join(strings.Fields(line), " ")
}

func isNoiseFrame(line string) bool {
	for _, n := range noiseFrames {
		if strings.Contains(line, n) {
			return true
		}
	}
	return false
}

// TopFrame = frame aplikasi paling atas (baris pertama bukan noise SDK) yang sudah dinormalisasi.
func TopFrame(stack string) string {
	for _, l := range strings.Split(stack, "\n") {
		if strings.TrimSpace(l) != "" && !isNoiseFrame(l) {
			return clip(NormalizeFrame(l), 300)
		}
	}
	return ""
}

// Fingerprint = hash(tipe + frame teratas). Tanpa stack, pesan dengan angka diseragamkan menggantikan frame.
func Fingerprint(errType, stack, msg string) string {
	basis := TopFrame(stack)
	if basis == "" {
		basis = "msg:" + reDigits.ReplaceAllString(reAddr.ReplaceAllString(msg, ""), "N")
	}
	sum := sha1.Sum([]byte(errType + "\n" + basis))
	return hex.EncodeToString(sum[:])
}

// cleanStack memotong stack di batas baris dan membuang baris kosong.
func cleanStack(stack string) string {
	var b strings.Builder
	for _, l := range strings.Split(stack, "\n") {
		l = strings.TrimRight(l, " \t\r")
		if strings.TrimSpace(l) == "" {
			continue
		}
		if b.Len()+len(l)+1 > maxStack {
			break
		}
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// errorIncident = satu kejadian error yang sudah dibersihkan.
type errorIncident struct {
	at                           time.Time
	fp, typ, msg, stack, top     string
	route, method, side, release string
}

func incidentOf(n normEvent) errorIncident {
	ev := n.ev
	typ := clip(strings.TrimSpace(ev.ErrType), maxErrType)
	if typ == "" {
		typ = "Error"
	}
	msg := clip(ev.ErrMsg, maxErrMsg)
	stack := cleanStack(ev.Stack)
	side := SideServer
	if ev.Side == SideBrowser {
		side = SideBrowser
	}
	return errorIncident{
		at: n.at, fp: Fingerprint(typ, stack, msg), typ: typ, msg: msg, stack: stack, top: TopFrame(stack),
		route: n.route, method: n.method, side: side, release: clip(strings.TrimSpace(ev.Release), maxRelease),
	}
}

// recordErrors menggabungkan kejadian error ke tabel grup (satu transaksi per batch).
func recordErrors(tx *gorm.DB, sourceID uint, list []errorIncident) error {
	for _, in := range list {
		var g models.TrafficErrorGroup
		err := tx.Where("source_id = ? AND fingerprint = ?", sourceID, in.fp).Take(&g).Error
		if err == gorm.ErrRecordNotFound {
			g = models.TrafficErrorGroup{
				SourceID: sourceID, Fingerprint: in.fp, ErrType: in.typ, Message: in.msg, TopFrame: in.top, Stack: in.stack,
				Route: in.route, Method: in.method, Side: in.side, Count: 1, FirstSeen: in.at, LastSeen: in.at,
				FirstRelease: in.release, LastRelease: in.release, Status: models.TrafficErrorOpen,
			}
			if err = tx.Create(&g).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err = tx.Model(&g).Updates(groupUpdate(g, in)).Error; err != nil {
			return err
		}
	}
	return nil
}

func groupUpdate(g models.TrafficErrorGroup, in errorIncident) map[string]any {
	u := map[string]any{"count": g.Count + 1}
	if in.at.After(g.LastSeen) {
		u["last_seen"] = in.at
		if in.release != "" {
			u["last_release"] = in.release
		}
	}
	if in.at.Before(g.FirstSeen) {
		u["first_seen"] = in.at
		if in.release != "" {
			u["first_release"] = in.release
		}
	}
	if g.FirstRelease == "" && in.release != "" {
		u["first_release"] = in.release
	}
	if g.Status == models.TrafficErrorResolved {
		u["status"] = models.TrafficErrorOpen // muncul lagi setelah ditandai selesai
	}
	return u
}

// saveProcess mencatat status proses dari heartbeat (upsert per sumber + instance).
func saveProcess(tx *gorm.DB, sourceID uint, n normEvent) error {
	p := models.TrafficProcess{
		SourceID: sourceID, Instance: n.instance, Release: clip(strings.TrimSpace(n.ev.Release), maxRelease),
		MemBytes: max(n.ev.MemBytes, 0), UptimeS: max(n.ev.UptimeS, 0), FirstSeen: n.at, LastSeen: n.at,
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_id"}, {Name: "instance"}},
		DoUpdates: clause.AssignmentColumns([]string{"release", "mem_bytes", "uptime_s", "last_seen"}),
	}).Create(&p).Error
}

// splitKinds memisahkan event request dari event error/heartbeat.
func splitKinds(events []Event) (reqs, special []Event) {
	for _, e := range events {
		if IsSpecialKind(e.Kind) {
			special = append(special, e)
		} else {
			reqs = append(reqs, e)
		}
	}
	return
}

// ingestSpecial menyimpan event error/heartbeat. Tidak menyentuh statistik request.
func ingestSpecial(db *gorm.DB, sourceID uint, events []Event, defaultInstance string) error {
	if len(events) == 0 {
		return nil
	}
	now := timeNow()
	return db.Transaction(func(tx *gorm.DB) error {
		var incidents []errorIncident
		for _, ev := range events {
			n := normalizeEvent(ev, now, defaultInstance)
			if ev.Kind == KindHeartbeat {
				if err := saveProcess(tx, sourceID, n); err != nil {
					return err
				}
				continue
			}
			incidents = append(incidents, incidentOf(n))
		}
		return recordErrors(tx, sourceID, incidents)
	})
}

// ErrorGroups mengembalikan grup error sumber (terbaru dulu) tanpa stack; limit+1 dibaca untuk has_more.
func ErrorGroups(db *gorm.DB, sourceID uint, status string, limit, offset int) ([]models.TrafficErrorGroup, bool, error) {
	limit = min(max(limit, 1), 100)
	q := db.Select("id, fingerprint, err_type, message, top_frame, route, method, side, count, first_seen, last_seen, first_release, last_release, status").
		Where("source_id = ?", sourceID).Order("last_seen DESC, id DESC").Offset(max(offset, 0)).Limit(limit + 1)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []models.TrafficErrorGroup
	if err := q.Find(&rows).Error; err != nil {
		return nil, false, err
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	return rows, more, nil
}

// ProcessAliveSec = heartbeat dianggap hidup bila terakhir terlihat dalam batas ini (heartbeat tiap 30 detik).
const ProcessAliveSec = 90

// ProcessView = status proses beserta kesimpulan hidup/mati.
type ProcessView struct {
	models.TrafficProcess
	Alive bool `json:"alive"`
}

func Processes(db *gorm.DB, sourceID uint) ([]ProcessView, error) {
	var rows []models.TrafficProcess
	if err := db.Where("source_id = ?", sourceID).Order("last_seen DESC, id DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	now := timeNow()
	out := make([]ProcessView, len(rows))
	for i, r := range rows {
		out[i] = ProcessView{TrafficProcess: r, Alive: now.Sub(r.LastSeen) < ProcessAliveSec*time.Second}
	}
	return out, nil
}

// BrowserEventsOnly menyaring kiriman yang diautentikasi dengan kunci di badan (kunci itu tampil di JS publik):
// hanya event error dari browser yang diterima, supaya statistik request & heartbeat tidak bisa dipalsukan.
func BrowserEventsOnly(events []Event) []Event {
	out := make([]Event, 0, len(events))
	for _, e := range events {
		if e.Kind != KindError {
			continue
		}
		e.Side = SideBrowser
		e.Instance = ""
		out = append(out, e)
	}
	return out
}
