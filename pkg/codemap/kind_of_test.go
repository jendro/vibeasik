package codemap

import "testing"

func TestKindOf_ClassifiesFromPathOnly(t *testing.T) {
	cases := []struct{ path, lang, kind string }{
		{"internal/handlers/order_handler.go", "go", "handler"},
		{"app/pages/items.vue", "vue", "page"},
		{"README.md", "markdown", "doc"},
		{"pkg/store/store_test.go", "go", "test"},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			if lang, kind := KindOf(c.path); lang != c.lang || kind != c.kind {
				t.Fatalf("KindOf(%s) = %s/%s, mau %s/%s", c.path, lang, kind, c.lang, c.kind)
			}
		})
	}
}
