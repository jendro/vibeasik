package knowledgesync

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/account"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.KnowledgeNode{}, &models.KnowledgeLink{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestImportPackIdempotentAndSkipsLocalKeys(t *testing.T) {
	db := testDB(t)
	db.Create(&models.KnowledgeNode{Key: "guideline:SEC-001", Kind: "rule", Domain: "security", Title: "lokal", Content: "x", Source: models.KnowledgeSrcGuideline, Status: "active"})
	p := &pack{Format: 1, Nodes: []packNode{
		{Key: "guideline:SEC-001", Kind: "rule", Domain: "security", Title: "dari paket", Content: "y", Source: "guideline", Helpful: 99},
		{Key: "hub:go:7", Kind: "lesson", Domain: "go", Title: "a", Content: "b", Source: "qa_review", Helpful: 50},
		{Key: "hub:go:8", Kind: "lesson", Domain: "go", Title: "c", Content: "d", Source: "qa_review"},
	}, Links: []packLink{{From: "hub:go:7", To: "hub:go:8", Strength: 0.5}}}

	r, err := importPack(db, p)
	if err != nil || r.Added != 2 || r.Skipped != 1 || r.Links != 1 {
		t.Fatalf("first import: %+v %v", r, err)
	}
	r, _ = importPack(db, p)
	if r.Added != 0 || r.Updated != 2 || r.Links != 0 {
		t.Fatalf("second import must be idempotent: %+v", r)
	}
	var n models.KnowledgeNode
	db.Where("source_key = ?", "hub:go:7").First(&n)
	if n.Helpful != 0 || n.Source != models.KnowledgeSrcHub {
		t.Fatalf("pack evidence must not become local evidence: %+v", n)
	}
	var local models.KnowledgeNode
	db.Where("source_key = ?", "guideline:SEC-001").First(&local)
	if local.Title != "lokal" {
		t.Fatal("local source node must not be overwritten")
	}
}

func TestVerifySignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	old := account.PublicKey
	account.PublicKey = base64.StdEncoding.EncodeToString(pub)
	defer func() { account.PublicKey = old }()
	data := []byte("paket")
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, data))
	if err := verifySignature(data, sig); err != nil {
		t.Fatal(err)
	}
	if verifySignature([]byte("paket!"), sig) == nil || verifySignature(data, "bukan-base64") == nil {
		t.Fatal("tampered data or bad signature must be rejected")
	}
}
