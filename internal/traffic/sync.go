package traffic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
)

// Interval tarik yang boleh dipilih (detik). 5 = "realtime".
var SyncIntervals = []int{5, 15, 60, 300, 900, 1800, 3600}

const maxPagesPerSync = 20

var httpClient = &http.Client{Timeout: 20 * time.Second}

type cloudRegistered struct {
	UID       string `json:"uid"`
	IngestKey string `json:"ingest_key"`
	PullKey   string `json:"pull_key"`
	Error     string `json:"error"`
}

type pulledEvent struct {
	ID       uint   `json:"id"`
	Instance string `json:"instance"`
	Event    Event  `json:"event"`
}

type pullResponse struct {
	Source struct {
		Dropped      int64      `json:"dropped"`
		LastIngestAt *time.Time `json:"last_ingest_at"`
	} `json:"source"`
	Events  []pulledEvent `json:"events"`
	Cursor  uint          `json:"cursor"`
	HasMore bool          `json:"has_more"`
	Pending int64         `json:"pending"`
	Error   string        `json:"error"`
}

func cloudDo(method, url, header, key string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if header != "" {
		req.Header.Set(header, key)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vibeasik live tidak terjangkau (%s): %w", url, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &e)
		if e.Error == "" {
			e.Error = strings.TrimSpace(string(data))
		}
		return fmt.Errorf("vibeasik live membalas %d: %s", resp.StatusCode, clip(e.Error, 300))
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

// RegisterCloud mendaftarkan sumber di vibeasik live dan mengisi kunci ingest/pull.
func RegisterCloud(src *models.TrafficSource) error {
	base := strings.TrimRight(src.CloudURL, "/")
	var out cloudRegistered
	header, token := "", os.Getenv("VIBEASIK_TRAFFIC_REGISTER_TOKEN")
	if token != "" {
		header = "X-Vibeasik-Register-Token"
	}
	if err := cloudDo(http.MethodPost, base+"/api/v1/traffic/sources", header, token, map[string]string{"name": src.Name}, &out); err != nil {
		return err
	}
	if out.IngestKey == "" || out.PullKey == "" {
		return fmt.Errorf("vibeasik live tidak mengembalikan kunci traffic")
	}
	src.CloudUID, src.IngestKey, src.PullKey = out.UID, out.IngestKey, out.PullKey
	return nil
}

// RotateCloudKey mengganti kunci SDK di vibeasik live (kunci lama langsung ditolak).
func RotateCloudKey(src *models.TrafficSource) error {
	var out cloudRegistered
	if err := cloudDo(http.MethodPost, strings.TrimRight(src.CloudURL, "/")+"/api/v1/traffic/rotate-key", "X-Vibeasik-Pull-Key", src.PullKey, nil, &out); err != nil {
		return err
	}
	src.IngestKey = out.IngestKey
	return nil
}

func DeleteCloud(src *models.TrafficSource) error {
	return cloudDo(http.MethodDelete, strings.TrimRight(src.CloudURL, "/")+"/api/v1/traffic/source", "X-Vibeasik-Pull-Key", src.PullKey, nil, nil)
}

var (
	busyMu sync.Mutex
	busy   = map[uint]bool{}
)

// SyncNow menarik semua event yang menunggu di vibeasik live untuk satu sumber (maks maxPagesPerSync halaman).
func SyncNow(db *gorm.DB, sourceID uint) (int, error) {
	busyMu.Lock()
	if busy[sourceID] {
		busyMu.Unlock()
		return 0, nil
	}
	busy[sourceID] = true
	busyMu.Unlock()
	defer func() {
		busyMu.Lock()
		delete(busy, sourceID)
		busyMu.Unlock()
	}()

	var src models.TrafficSource
	if err := db.First(&src, sourceID).Error; err != nil {
		return 0, err
	}
	if src.Mode != models.TrafficModeCloud || src.PullKey == "" {
		return 0, nil
	}
	base := strings.TrimRight(src.CloudURL, "/")
	total := 0
	var lastErr error
	var resp pullResponse
	for page := 0; page < maxPagesPerSync; page++ {
		resp = pullResponse{}
		url := fmt.Sprintf("%s/api/v1/traffic/pull?after=%d&limit=%d", base, src.Cursor, MaxEventsPerIngest)
		if lastErr = cloudDo(http.MethodGet, url, "X-Vibeasik-Pull-Key", src.PullKey, nil, &resp); lastErr != nil {
			break
		}
		if len(resp.Events) > 0 {
			events := make([]Event, len(resp.Events))
			for i, pe := range resp.Events {
				events[i] = pe.Event
				if events[i].Instance == "" {
					events[i].Instance = pe.Instance
				}
			}
			n, err := Ingest(db, &src, events, "")
			if err != nil {
				lastErr = err
				break
			}
			total += n
		}
		// Cursor disimpan segera: tarikan berikutnya meng-ack (menghapus) event ini di vibeasik live.
		src.Cursor = resp.Cursor
		db.Model(&models.TrafficSource{}).Where("id = ?", src.ID).Update("cursor", src.Cursor)
		if !resp.HasMore {
			break
		}
	}

	now := time.Now()
	updates := map[string]any{"last_sync_at": now, "last_error": ""}
	if lastErr != nil {
		updates["last_error"] = clip(lastErr.Error(), 1000)
	} else {
		updates["pending"] = max(resp.Pending-int64(len(resp.Events)), 0)
		updates["dropped"] = resp.Source.Dropped
		if resp.Source.LastIngestAt != nil {
			updates["last_ingest_at"] = *resp.Source.LastIngestAt
		}
	}
	db.Model(&models.TrafficSource{}).Where("id = ?", src.ID).Updates(updates)
	return total, lastErr
}

// StartSyncer menarik setiap sumber cloud sesuai interval masing-masing & memangkas rangkuman lama tiap jam.
func StartSyncer(ctx context.Context, db *gorm.DB) {
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		lastPrune := time.Time{}
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			var sources []models.TrafficSource
			db.Where("enabled = ? AND mode = ? AND pull_key <> ''", true, models.TrafficModeCloud).Find(&sources)
			now := time.Now()
			for _, s := range sources {
				interval := time.Duration(max(s.SyncIntervalSec, 5)) * time.Second
				if s.LastSyncAt != nil && now.Sub(*s.LastSyncAt) < interval-time.Second {
					continue
				}
				// Gagal tarik dicatat di last_error (tampil di halaman), tidak di log agar tidak membanjir tiap 5 detik.
				go func(id uint) { _, _ = SyncNow(db, id) }(s.ID)
			}
			if now.Sub(lastPrune) >= time.Hour {
				lastPrune = now
				Prune(db)
			}
		}
	}()
}
