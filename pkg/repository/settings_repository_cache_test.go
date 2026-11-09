package repository_test

import (
	"context"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
)

func TestSettingsRepositoryCacheAndOptimization(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Seed awal satu LLMSettings
	initSettings := models.LLMSettings{
		ActiveProvider:  models.ProviderSystemOne,
		SystemOneURL:    "http://localhost:8750/v1/systemone",
		SystemOneModel:  "laya",
		StandardLLMURL:  "https://openrouter.ai/api/v1",
		StandardLLMKey:  "sk-test-key",
	}
	if err := db.Create(&initSettings).Error; err != nil {
		t.Fatalf("failed to seed initial LLMSettings: %v", err)
	}

	repo := repository.NewSettingsRepository(db)

	var selectCount int32
	var updateCount int32

	cb := db.Callback()
	_ = cb.Query().After("gorm:query").Register("test_count_settings_query", func(d *gorm.DB) {
		atomic.AddInt32(&selectCount, 1)
	})
	_ = cb.Update().After("gorm:update").Register("test_count_settings_update", func(d *gorm.DB) {
		atomic.AddInt32(&updateCount, 1)
	})

	resetCounts := func() {
		atomic.StoreInt32(&selectCount, 0)
		atomic.StoreInt32(&updateCount, 0)
	}

	// 1. Fetch pertama: Cache miss -> query DB (1 SELECT)
	resetCounts()
	s1, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings #1 failed: %v", err)
	}
	if s1.ActiveProvider != models.ProviderSystemOne {
		t.Errorf("expected provider %s, got %s", models.ProviderSystemOne, s1.ActiveProvider)
	}
	if sc := atomic.LoadInt32(&selectCount); sc != 1 {
		t.Errorf("GetSettings #1: expected 1 query, got %d", sc)
	}

	// 2. Fetch kedua: Cache hit -> no DB query (0 SELECT)
	resetCounts()
	s2, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings #2 failed: %v", err)
	}
	if s2.SystemOneURL != "http://localhost:8750/v1/systemone" {
		t.Errorf("expected url preserved, got %s", s2.SystemOneURL)
	}
	if sc := atomic.LoadInt32(&selectCount); sc != 0 {
		t.Errorf("GetSettings #2 (cached): expected 0 query, got %d", sc)
	}

	// 3. Mutasi objek hasil tidak boleh merusak internal cache (copy-on-read isolation)
	s2.ActiveProvider = "mutated-in-caller"
	s3, _ := repo.GetSettings(ctx)
	if s3.ActiveProvider == "mutated-in-caller" {
		t.Errorf("cache corrupted by caller mutation, expected original %s", models.ProviderSystemOne)
	}

	// 4. UpdateSettings: invalidation-on-write & optimasi Updates (1 UPDATE)
	resetCounts()
	s1.ActiveProvider = models.ProviderStandardLLM
	s1.StandardLLMSelected = "anthropic/claude-3.5-sonnet"
	if err := repo.UpdateSettings(ctx, s1); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}
	if uc := atomic.LoadInt32(&updateCount); uc != 1 {
		t.Errorf("UpdateSettings: expected 1 update statement, got %d", uc)
	}

	// 5. Fetch ketiga: Cache langsung memuat data yang baru di-update tanpa query DB tambahan
	resetCounts()
	s4, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings #3 failed: %v", err)
	}
	if s4.ActiveProvider != models.ProviderStandardLLM {
		t.Errorf("expected updated provider %s, got %s", models.ProviderStandardLLM, s4.ActiveProvider)
	}
	if s4.StandardLLMSelected != "anthropic/claude-3.5-sonnet" {
		t.Errorf("expected selected model anthropic/claude-3.5-sonnet, got %s", s4.StandardLLMSelected)
	}
	if sc := atomic.LoadInt32(&selectCount); sc != 0 {
		t.Errorf("GetSettings #3 (refreshed cache): expected 0 query, got %d", sc)
	}
}

func BenchmarkSettingsRepositoryGetCached(b *testing.B) {
	db := setupTestDB(&testing.T{})
	ctx := context.Background()

	initSettings := models.LLMSettings{
		ActiveProvider: models.ProviderSystemOne,
		SystemOneURL:   "http://localhost:8750/v1/systemone",
	}
	_ = db.Create(&initSettings)
	repo := repository.NewSettingsRepository(db)

	// Warmup cache
	_, _ = repo.GetSettings(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetSettings(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
