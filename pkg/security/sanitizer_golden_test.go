package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeGolden(t *testing.T) {
	s := NewSanitizer()
	s.SetCustomKeywords([]CustomKeywordRule{
		{Keyword: "ProjectPhoenix", Category: "custom", IsActive: true},
		{Keyword: "acme-[0-9]+", Category: "client", IsActive: true, IsRegex: true},
		{Keyword: "inactive", IsActive: false},
		{Keyword: "  ", IsActive: true},
		{Keyword: "bad(", IsActive: true, IsRegex: true},
		{Keyword: "Secretword", IsActive: true},
	})
	key := func(parts ...string) string {
		out := ""
		for _, p := range parts {
			out += p
		}
		return out
	}
	inputs := []string{
		"",
		"   \n",
		"nothing secret here",
		"use " + key("sk-", "proj-", "abcdefghijklmnopqrstuvwxyz0123") + " and again " + key("sk-", "proj-", "abcdefghijklmnopqrstuvwxyz0123"),
		"DATABASE_URL=postgres://admin:hunter22pw@db.local:5432/app and redis://u:p4ssw0rd@h",
		"API_KEY: \"abcd1234efgh\" TOKEN=true PASSWORD=null SECRET='short' MY_SECRET=longsecretvalue",
		"Authorization: Bearer " + key("abcdef", "ghijklmnopqrstuvwxyz1234") + " done",
		"jwt " + key("eyJhbGciOiJIUzI1NiJ9", ".", "eyJzdWIiOiIxMjM0NTY3ODkwIn0", ".", "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"),
		"-----BEGIN RSA PRIV" + "ATE KEY-----\nMIIBOgIBAAJBAKj34GkxFhD90vcNLYLInFEX6Ppy1tPf9Cnzj4p4WGeKLs1Pt8Qu\n-----END RSA PRIV" + "ATE KEY-----",
		"aws " + key("AKIA", "IOSFODNN7EXAMPLE") + " gh " + key("ghp", "_", "abcdefghijklmnopqrstuvwxyz0123456789") + " slack " + key("xoxb", "-", "1234567890abcdef"),
		"ProjectPhoenix meets acme-42 and acme-7; secretword SECRETWORD [ALIASED_X_1]",
		"stripe " + key("sk", "_live_", "abcdefghijklmnopqrstuvwx") + " hook https://hooks.slack.com/services/T000/B000/XXXX",
		"hash " + key("abcdefghijklmnopqrstuvwxyz", "012345678901") + " end",
		"long " + string(make([]byte, 0)) + key("sk-ant-", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") + " " + key("ProjectPhoenix ", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"),
	}
	type row struct {
		Idx   int
		Out   string
		Audit *SanitizeAudit
	}
	var out []row
	for _, in := range inputs {
		o, a := s.Sanitize(in)
		out = append(out, row{len(out), o, a})
	}
	got, _ := json.MarshalIndent(out, "", " ")
	path := filepath.Join("testdata", "sanitize.golden.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(got) {
		t.Error("output Sanitize berubah")
	}
}
