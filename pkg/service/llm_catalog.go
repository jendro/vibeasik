package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// DiscoveredModel adalah satu model dari katalog /models provider OpenAI-compatible.
// ID dikirim ke provider saat request; DisplayName hanya untuk tampilan di vibeasik.
type DiscoveredModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
	OwnedBy     string `json:"owned_by,omitempty"`
}

const (
	catalogTimeout  = 15 * time.Second
	catalogMaxBytes = 4 << 20
	catalogMaxItems = 500
)

var catalogClient = &http.Client{Timeout: catalogTimeout}

// catalogURL membentuk alamat katalog model dari base URL (".../v1" atau ".../v1/chat/completions").
func catalogURL(base string) string {
	u := strings.TrimRight(strings.TrimSpace(base), "/")
	u = strings.TrimSuffix(u, "/chat/completions")
	return strings.TrimRight(u, "/") + "/models"
}

// DiscoverLLMModels mengambil daftar model (id + display_name) dari GET {url}/models.
// Nama tampilan dibaca dari display_name, lalu name; kosong bila provider tidak memberinya.
func DiscoverLLMModels(ctx context.Context, baseURL, apiKey string) ([]DiscoveredModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogURL(baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("url katalog tidak valid: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if k := strings.TrimSpace(apiKey); k != "" {
		req.Header.Set("Authorization", "Bearer "+k)
	}
	resp, err := catalogClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("katalog model tidak terjangkau: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, catalogMaxBytes))
	if err != nil {
		return nil, fmt.Errorf("gagal membaca katalog model: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("katalog model menjawab status %d", resp.StatusCode)
	}
	return parseModelCatalog(body)
}

// parseModelCatalog menerima bentuk {"data":[...]} (OpenAI) maupun array langsung.
func parseModelCatalog(body []byte) ([]DiscoveredModel, error) {
	type rawModel struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		Name        string `json:"name"`
		OwnedBy     string `json:"owned_by"`
	}
	var wrapped struct {
		Data []rawModel `json:"data"`
	}
	var list []rawModel
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Data != nil {
		list = wrapped.Data
	} else if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("format katalog model tidak dikenali")
	}
	seen := map[string]bool{}
	out := make([]DiscoveredModel, 0, len(list))
	for _, m := range list {
		id := strings.TrimSpace(m.ID)
		if id == "" || seen[id] || len(id) > 255 {
			continue
		}
		seen[id] = true
		name := strings.TrimSpace(m.DisplayName)
		if name == "" {
			name = strings.TrimSpace(m.Name)
		}
		if name == id || len(name) > 255 {
			name = ""
		}
		out = append(out, DiscoveredModel{ID: id, DisplayName: name, OwnedBy: strings.TrimSpace(m.OwnedBy)})
		if len(out) >= catalogMaxItems {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
