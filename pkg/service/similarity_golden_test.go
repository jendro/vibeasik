package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestCompareTwoProjectsGolden(t *testing.T) {
	services, db := setupTestServices(t)
	ctx := context.Background()

	n := 0
	mk := func(name, root, desc, tags, status string, comps ...models.Component) uint {
		n++
		p := models.Project{Name: name, Slug: fmt.Sprintf("slug-%d", n), RootPath: root,
			Description: desc, Tags: tags, Status: status, Components: comps}
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
		return p.ID
	}
	be := models.Component{Name: "api", Role: models.RoleBackend, Language: "go", Framework: "fiber", AbsolutePath: "/w/shop/api"}
	fe := models.Component{Name: "web", Role: models.RoleFrontendMobile, Language: "typescript", AbsolutePath: "/w/shop/web"}
	fe2 := models.Component{Name: "site", Role: "frontend_web", Language: "auto", AbsolutePath: "/w/other/site"}
	be2 := models.Component{Name: "api2", Role: models.RoleBackend, AbsolutePath: "/w/shop/api"}
	noPath := models.Component{Name: "np", Role: models.RoleBackend}

	ids := map[string]uint{
		"a1": mk("ShopBackend", "/w/shop-be", "toko online produk keranjang", "shop,erp", "", be),
		"a2": mk("shop-frontend", "/w/shop-fe", "toko online produk katalog", "shop", "", fe2),
		"a3": mk("Shop Mobile", "/w/shop-be/mobile", "toko online", "", models.StatusDeprecated, fe),
		"a4": mk("Zebra", "/w/zzz", "hewan", "", "", be2),
		"a5": mk("Other", "/x/zzz", "lain sama sekali", "", "", noPath),
		"a6": mk("Other2", "-", "", "", "", noPath),
		"a7": mk("Other3", "-", "", "", ""),
		"a8": mk("ShopBackend", "/w/shop-be", "toko online produk keranjang", "shop,erp", ""),
	}
	pairs := [][2]string{{"a1", "a2"}, {"a1", "a3"}, {"a2", "a3"}, {"a1", "a4"}, {"a1", "a5"}, {"a5", "a6"}, {"a6", "a7"},
		{"a1", "a8"}, {"a4", "a5"}, {"a3", "a8"}, {"a2", "a8"}}
	var out []any
	for _, pr := range pairs {
		m, err := services.Similarity.CompareTwoProjects(ctx, ids[pr[0]], ids[pr[1]])
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	golden(t, "similarity_pairs", out, "")
}
