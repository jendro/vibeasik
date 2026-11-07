package models

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRejectGuard(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Todo{}); err != nil {
		t.Fatal(err)
	}
	pid := uint(7)
	db.Create(&Todo{ProjectID: &pid, Title: "Amankan Endpoint Reset Password Admin", Status: TodoStatusRejected,
		GuidelineRef: "ROUTER-UNGUARDED-MUTATION", TargetApps: "Bimtek (routes.go:L183)"})
	db.Create(&Todo{ProjectID: &pid, Title: "Validasi Ketat Payload Resend", Status: TodoStatusRejected,
		AuditKey: "DEEP-FORM-MISSING-VALIDATION|handlers.AdminResendCertMail"})

	cases := []struct {
		todo    Todo
		blocked bool
	}{
		{Todo{Title: "Tambahkan Guard Admin pada Endpoint Reset Password", CreatorType: "ai_agent"}, true},
		{Todo{Title: "Proteksi rute lain", GuidelineRef: "ROUTER-UNGUARDED-MUTATION", TargetApps: "X (routes/routes.go:L200)", CreatorType: "ai_agent"}, true},
		{Todo{Title: "Perbaiki apa saja", AuditKey: "DEEP-FORM-MISSING-VALIDATION|handlers.AdminResendCertMail", CreatorType: "ai_agent"}, true},
		{Todo{Title: "Amankan Endpoint Penghapusan Kategori", CreatorType: "ai_agent"}, false},
		{Todo{Title: "Amankan Endpoint Reset Password Admin", CreatorType: "human"}, false},
	}
	for _, c := range cases {
		td := c.todo
		td.ProjectID = &pid
		td.Status = TodoStatusPending
		if err := db.Create(&td).Error; err != nil {
			t.Fatal(err)
		}
		got := td.Status == TodoStatusRejected
		if got != c.blocked {
			t.Errorf("%q: blocked=%v, want %v", td.Title, got, c.blocked)
		}
		if got && !strings.Contains(td.Rationale, "Ditolak otomatis") {
			t.Errorf("%q: rationale tanpa alasan", td.Title)
		}
	}
	other := uint(8)
	td := Todo{ProjectID: &other, Title: "Amankan Endpoint Reset Password Admin", Status: TodoStatusPending}
	db.Create(&td)
	if td.Status != TodoStatusPending {
		t.Error("proyek lain tidak boleh ikut tersaring")
	}
}
