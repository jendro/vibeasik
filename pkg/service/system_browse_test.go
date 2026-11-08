package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/service"
)

func TestBrowseDirectoriesGolden(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFiles(t, home, map[string]string{
		"Documents/Nuxt/a/x.txt": "x", "Documents/go/b/x.txt": "x", "Documents/alpha/deep/deeper/deepest/deepmost/f": "x",
		"Documents/beta/.hidden/f": "x", "Documents/node_modules/q/f": "x", "Documents/vendor/q/f": "x",
		"Documents/Alphabet/f": "x", "Documents/file.txt": "x", "other/zeta/f": "x",
		"Documents/alpha/dist/alpha2/f": "x", "Documents/alpha/n/alpha3/f": "x",
	})
	sys := service.NewSystemService()
	ctx := context.Background()
	type call struct{ target, query string }
	calls := []call{
		{"", ""}, {"~/other", ""}, {home + "/Documents/alpha", ""}, {"/nonexistent/path", ""},
		{home + "/Documents/file.txt", ""}, {"", "alpha"}, {"~/other", "zeta"}, {"/nonexistent", "BETA"},
		{"", "deep"}, {"", "nomatch"}, {"  ~/Documents  ", "  "},
	}
	var out []any
	for _, c := range calls {
		r, err := sys.BrowseDirectories(ctx, c.target, c.query)
		if err != nil {
			out = append(out, err.Error())
			continue
		}
		out = append(out, r)
	}
	// Tanpa Documents: jatuh ke home.
	h2 := filepath.Join(home, "h2")
	t.Setenv("HOME", h2)
	_ = os.MkdirAll(filepath.Join(h2, "sub"), 0o755)
	r, err := sys.BrowseDirectories(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	out = append(out, r)
	golden(t, "browse_dirs", out, home)
}
