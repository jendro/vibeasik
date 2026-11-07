// Package knowledgesync menyambungkan Pusat Pengetahuan lokal ke Knowledge Hub vibeasik:
//
//   - Pull: menarik paket bertanda tangan (free/full sesuai akun), memverifikasi, lalu mengimpornya;
//   - SendSignals: mengirim angka helpful/harmful node milik paket (anonim per perangkat, tanpa teks);
//   - Contribute: mengirim usulan node yang SUDAH Anda setujui, setelah disensor.
//
// Pencarian pengetahuan tetap sepenuhnya lokal; server hanya membagikan paket statis.
package knowledgesync

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/account"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/security"
)

const maxPackBytes = 32 << 20

var client = &http.Client{Timeout: 60 * time.Second}

// State catatan sinkronisasi di ~/.vibeasik/knowledge.json.
type State struct {
	Version      string    `json:"version"`
	Tier         string    `json:"tier"`
	PulledAt     time.Time `json:"pulled_at"`
	Nodes        int       `json:"nodes"`
	Contributed  []uint    `json:"contributed,omitempty"` // id node lokal yang sudah dikirim sebagai usulan
	LastSignalAt time.Time `json:"last_signal_at"`
}

func statePath() string { return filepath.Join(localauth.Dir(), "knowledge.json") }

// LoadState membaca catatan sinkronisasi (kosong bila belum pernah).
func LoadState() State {
	var s State
	if raw, err := os.ReadFile(statePath()); err == nil {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

func saveState(s State) error {
	if err := os.MkdirAll(localauth.Dir(), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(statePath(), raw, 0o600)
}

type packNode struct {
	Key     string   `json:"key"`
	Kind    string   `json:"kind"`
	Domain  string   `json:"domain"`
	Tags    []string `json:"tags"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Source  string   `json:"source"`
	Weight  int      `json:"weight"`
	Helpful int      `json:"helpful"`
	Harmful int      `json:"harmful"`
}

type packLink struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Strength float64 `json:"strength"`
}

type pack struct {
	Format int        `json:"format"`
	Tier   string     `json:"tier"`
	Nodes  []packNode `json:"nodes"`
	Links  []packLink `json:"links"`
}

// Result ringkasan hasil Pull.
type Result struct {
	Version  string `json:"version"`
	Tier     string `json:"tier"`
	Added    int    `json:"added"`
	Updated  int    `json:"updated"`
	Skipped  int    `json:"skipped"`
	Links    int    `json:"links"`
	UpToDate bool   `json:"up_to_date"`
}

// Pull menarik paket terbaru. Tanpa login mendapat tier free; akun berhak mendapat full.
func Pull(db *gorm.DB, force bool) (*Result, error) {
	var latest struct {
		Version string `json:"version"`
		Tier    string `json:"tier"`
		URL     string `json:"url"`
		File    struct {
			SHA256    string `json:"sha256"`
			Signature string `json:"signature"`
		} `json:"file"`
		Error string `json:"error"`
	}
	if err := get("/api/v1/knowledge/packs/latest", &latest); err != nil {
		return nil, err
	}
	if latest.Error != "" {
		return nil, errors.New(latest.Error)
	}
	st := LoadState()
	if !force && st.Version == latest.Version && st.Tier == latest.Tier {
		return &Result{Version: st.Version, Tier: st.Tier, UpToDate: true}, nil
	}
	raw, err := download(latest.URL)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != latest.File.SHA256 {
		return nil, errors.New("checksum paket tidak cocok: unduhan rusak atau dimanipulasi")
	}
	if err := verifySignature(raw, latest.File.Signature); err != nil {
		return nil, err
	}
	p, err := parse(raw)
	if err != nil {
		return nil, err
	}
	res, err := importPack(db, p)
	if err != nil {
		return nil, err
	}
	res.Version, res.Tier = latest.Version, latest.Tier
	st.Version, st.Tier, st.PulledAt, st.Nodes = latest.Version, latest.Tier, time.Now(), len(p.Nodes)
	return res, saveState(st)
}

func verifySignature(data []byte, sigB64 string) error {
	pub, err := base64.StdEncoding.DecodeString(account.PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("kunci publik di build ini rusak")
	}
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil || !ed25519.Verify(ed25519.PublicKey(pub), data, sig) {
		return errors.New("tanda tangan paket tidak valid: paket ditolak")
	}
	return nil
}

func parse(raw []byte) (*pack, error) {
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	body, err := io.ReadAll(io.LimitReader(zr, 4*maxPackBytes))
	if err != nil {
		return nil, err
	}
	var p pack
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	if p.Format != 1 {
		return nil, fmt.Errorf("format paket %d belum didukung; perbarui vibeasik", p.Format)
	}
	return &p, nil
}

// importPack memasukkan node paket sebagai sumber "hub". Node yang kuncinya sudah dimiliki sumber lokal
// (mis. guideline:SEC-001 dari rule bawaan) dilewati agar tidak ganda. Pencarian & klaster lokal
// memproses node baru pada kurasi berikutnya.
func importPack(db *gorm.DB, p *pack) (*Result, error) {
	res := &Result{}
	ids := map[string]uint{}
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, n := range p.Nodes {
			key := "hub:" + strings.TrimPrefix(n.Key, "hub:")
			var local models.KnowledgeNode
			// Sudah dimiliki sumber lokal lain dengan kunci asli → lewati.
			if n.Key != "" && !strings.HasPrefix(n.Key, "hub:") {
				if tx.Where("source_key = ? AND source <> ?", n.Key, models.KnowledgeSrcHub).Limit(1).Find(&local).RowsAffected > 0 {
					res.Skipped++
					continue
				}
			}
			found := tx.Where("source_key = ? AND source = ?", key, models.KnowledgeSrcHub).Limit(1).Find(&local).RowsAffected > 0
			if found {
				// Bukti helpful/harmful lokal dipertahankan; teks & bobot mengikuti paket.
				if err := tx.Model(&local).Updates(map[string]any{
					"title": n.Title, "content": n.Content, "kind": n.Kind, "domain": n.Domain, "weight": n.Weight, "status": models.KnowledgeNodeActive,
				}).Error; err != nil {
					return err
				}
				res.Updated++
			} else {
				local = models.KnowledgeNode{Key: key, Kind: n.Kind, Domain: n.Domain, Tags: n.Tags, Title: n.Title, Content: n.Content,
					Source: models.KnowledgeSrcHub, Weight: n.Weight, Status: models.KnowledgeNodeActive} // helpful/harmful paket = agregat pengguna lain; bukti lokal mulai dari nol agar sinyal yang dikirim hanya milik perangkat ini
				if err := tx.Create(&local).Error; err != nil {
					return err
				}
				res.Added++
			}
			ids[n.Key] = local.ID
		}
		for _, l := range p.Links {
			from, to := ids[l.From], ids[l.To]
			if from == 0 || to == 0 || from == to {
				continue
			}
			var c int64
			tx.Model(&models.KnowledgeLink{}).Where("from_id = ? AND to_id = ?", from, to).Count(&c)
			if c == 0 {
				if err := tx.Create(&models.KnowledgeLink{FromID: from, ToID: to, Strength: l.Strength}).Error; err != nil {
					return err
				}
				res.Links++
			}
		}
		return nil
	})
	return res, err
}

// SendSignals mengirim jumlah helpful/harmful node milik paket (hanya kunci & angka; tanpa teks).
func SendSignals(db *gorm.DB) (int, error) {
	tok := account.DeviceToken()
	if tok == "" {
		return 0, errors.New("belum login: sinyal hanya dikirim dari perangkat yang masuk akun")
	}
	var nodes []models.KnowledgeNode
	db.Where("source = ? AND (helpful > 0 OR harmful > 0)", models.KnowledgeSrcHub).Find(&nodes)
	type sig struct {
		NodeKey string `json:"node_key"`
		Helpful int    `json:"helpful"`
		Harmful int    `json:"harmful"`
	}
	out := make([]sig, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, sig{NodeKey: strings.TrimPrefix(n.Key, "hub:"), Helpful: n.Helpful, Harmful: n.Harmful})
	}
	if len(out) == 0 {
		return 0, nil
	}
	var resp struct {
		Accepted int `json:"accepted"`
	}
	if err := post("/api/v1/knowledge/signals", tok, map[string]any{"signals": out}, &resp); err != nil {
		return 0, err
	}
	st := LoadState()
	st.LastSignalAt = time.Now()
	_ = saveState(st)
	return resp.Accepted, nil
}

// Candidate usulan yang bisa dikirim (sudah disensor) — ditampilkan ke pengguna sebelum dikirim.
type Candidate struct {
	ID      uint   `json:"id"`
	Kind    string `json:"kind"`
	Domain  string `json:"domain"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Source  string `json:"source"`
	Key     string `json:"key"`
}

// Candidates node bersama buatan lokal (promosi, review QA, riset celah) yang belum pernah dikirim.
func Candidates(db *gorm.DB) []Candidate {
	st := LoadState()
	sent := map[uint]bool{}
	for _, id := range st.Contributed {
		sent[id] = true
	}
	var nodes []models.KnowledgeNode
	db.Where("owner_employee_id IS NULL AND status = ? AND source IN ?", models.KnowledgeNodeActive,
		[]string{models.KnowledgeSrcPromoted, models.KnowledgeSrcReview, models.KnowledgeSrcGap}).
		Where("helpful >= harmful").Order("helpful DESC, id").Limit(200).Find(&nodes)
	var out []Candidate
	for _, n := range nodes {
		if sent[n.ID] {
			continue
		}
		title, content := security.SanitizeText(n.Title), security.SanitizeText(n.Content)
		if looksPrivate(title) || looksPrivate(content) {
			continue
		}
		out = append(out, Candidate{ID: n.ID, Kind: n.Kind, Domain: n.Domain, Title: title, Content: content, Source: n.Source, Key: n.Key})
	}
	return out
}

func looksPrivate(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "/home/") || strings.Contains(l, "c:\\users") || strings.Contains(l, "localhost:") || strings.Contains(l, "@")
}

// Contribute mengirim usulan yang id-nya dipilih pengguna.
func Contribute(db *gorm.DB, ids []uint) (int, error) {
	tok := account.DeviceToken()
	if tok == "" {
		return 0, errors.New("belum login: usulan hanya bisa dikirim dari perangkat yang masuk akun")
	}
	want := map[uint]bool{}
	for _, id := range ids {
		want[id] = true
	}
	st := LoadState()
	n := 0
	for _, c := range Candidates(db) {
		if !want[c.ID] {
			continue
		}
		err := post("/api/v1/knowledge/contributions", tok, map[string]any{
			"kind": c.Kind, "domain": c.Domain, "title": c.Title, "content": c.Content, "source_key": c.Key,
		}, nil)
		if err != nil {
			return n, err
		}
		st.Contributed = append(st.Contributed, c.ID)
		n++
	}
	return n, saveState(st)
}

func get(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, account.CloudURL()+path, nil)
	if err != nil {
		return err
	}
	if tok := account.DeviceToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	return do(req, out)
}

func post(path, tok string, body, out any) error {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, account.CloudURL()+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	return do(req, out)
}

func do(req *http.Request, out any) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("server vibeasik (%s) tidak bisa dihubungi: %w", account.CloudURL(), err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if out != nil {
		_ = json.Unmarshal(raw, out)
	}
	if resp.StatusCode >= 400 {
		var e struct{ Error string }
		_ = json.Unmarshal(raw, &e)
		if e.Error == "" {
			e.Error = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return errors.New("server vibeasik: " + e.Error)
	}
	return nil
}

func download(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, account.CloudURL()+path, nil)
	if err != nil {
		return nil, err
	}
	if tok := account.DeviceToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unduh paket gagal: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unduh paket: status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxPackBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxPackBytes {
		return nil, errors.New("paket terlalu besar")
	}
	return raw, nil
}

// StartAutoPull menarik paket saat start (setelah jeda) lalu tiap 24 jam, dan mengirim sinyal bila sudah login.
// Dimatikan dengan env VIBEASIK_KNOWLEDGE_SYNC=off.
func StartAutoPull(ctx context.Context, db *gorm.DB) {
	if strings.EqualFold(os.Getenv("VIBEASIK_KNOWLEDGE_SYNC"), "off") {
		return
	}
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(45 * time.Second):
		}
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for {
			if res, err := Pull(db, false); err != nil {
				log.Printf("[knowledge] tarik paket dilewati: %v", err)
			} else if !res.UpToDate {
				log.Printf("[knowledge] paket %s (%s): %d baru, %d diperbarui", res.Version, res.Tier, res.Added, res.Updated)
			}
			if account.DeviceToken() != "" && LoadState().Version != "" {
				_, _ = SendSignals(db)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}
