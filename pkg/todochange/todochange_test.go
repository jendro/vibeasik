package todochange

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/vibeasik/vibeasik/pkg/models"
	"gorm.io/gorm"
)

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.test", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// repoWithTodoCommit: repo dengan satu commit awal lalu satu commit squash todo #id yang mengubah satu fungsi.
func repoWithTodoCommit(t *testing.T, id string) string {
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	write(t, dir, "api/user.go", "package api\n\nfunc Keep() int { return 1 }\n\nfunc Save(name string) error {\n\tif name == \"\" {\n\t\treturn nil\n\t}\n\treturn nil\n}\n")
	write(t, dir, "old.txt", "lama\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-q", "-m", "chore: awal")
	write(t, dir, "api/user.go", "package api\n\nfunc Keep() int { return 1 }\n\nfunc Save(name string) error {\n\tif name == \"\" {\n\t\treturn errEmpty\n\t}\n\treturn nil\n}\n")
	write(t, dir, "api/errors.go", "package api\n\nvar errEmpty = errorString(\"kosong\")\n")
	run(t, dir, "rm", "-q", "old.txt")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-q", "-m", "fix(api): tolak nama kosong", "-m", SquashTrailer+BranchPrefix(42)+"tolak-nama-kosong")
	return dir
}

func TestFindSquashResolveAndRead(t *testing.T) {
	dir := repoWithTodoCommit(t, "42")
	repos := []string{dir}
	repo, sha, err := FindSquash(repos, 42)
	if err != nil || repo != dir || len(sha) != 40 {
		t.Fatalf("FindSquash = %q %q %v", repo, sha, err)
	}
	if _, _, err := FindSquash(repos, 4); err != ErrNoCommit {
		t.Errorf("todo #4 tidak boleh cocok dengan branch todo #42: %v", err)
	}
	if _, full, err := Resolve(repos, sha[:8]); err != nil || full != sha {
		t.Errorf("Resolve SHA pendek = %q %v", full, err)
	}
	if _, _, err := Resolve(repos, "HEAD; rm -rf /"); err == nil {
		t.Error("ref selain heksadesimal ditolak")
	}
	if _, _, err := Resolve(repos, strings.Repeat("a", 40)); err == nil {
		t.Error("SHA yang tidak ada ditolak")
	}

	c, err := Read(dir, sha)
	if err != nil {
		t.Fatal(err)
	}
	if c.SHA != sha || c.Subject != "fix(api): tolak nama kosong" || !strings.Contains(c.Body, BranchPrefix(42)) || c.Repo != filepath.Base(dir) || len(c.Files) != 3 {
		t.Fatalf("commit terbaca: %+v", c)
	}
	byPath := map[string]FileChange{}
	for _, f := range c.Files {
		byPath[f.Path] = f
	}
	user := byPath["api/user.go"]
	if user.Status != "modified" || user.Added != 1 || user.Deleted != 1 || len(user.Functions) != 1 || user.Functions[0] != "Save" || user.Hunks[0].Function != "Save" {
		t.Fatalf("perubahan api/user.go harus menyebut fungsi Save: %+v", user)
	}
	var add, del bool
	for _, ln := range user.Hunks[0].Lines {
		add = add || (ln.Kind == "add" && ln.New == 7 && strings.Contains(ln.Text, "errEmpty"))
		del = del || (ln.Kind == "del" && ln.Old == 7)
	}
	if !add || !del {
		t.Errorf("baris diff harus bernomor sesuai berkas: %+v", user.Hunks[0].Lines)
	}
	if byPath["api/errors.go"].Status != "added" || byPath["old.txt"].Status != "deleted" || c.Added != 4 || c.Deleted != 2 {
		t.Errorf("status berkas & jumlah baris: %+v / +%d -%d", byPath, c.Added, c.Deleted)
	}
}

func TestAttachSavesTodoAndSubTodos(t *testing.T) {
	dir := repoWithTodoCommit(t, "42")
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "t.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.Todo{}); err != nil {
		t.Fatal(err)
	}
	p := models.Project{Name: "Toko", RootPath: dir}
	db.Create(&p)
	todo := models.Todo{ID: 42, ProjectID: &p.ID, Title: "Tolak nama kosong", Status: models.TodoStatusCompleted}
	db.Create(&todo)
	sub := models.Todo{ProjectID: &p.ID, ParentID: &todo.ID, Title: "Sub", Status: models.TodoStatusCompleted}
	db.Create(&sub)

	sha, err := Attach(db, &todo, "")
	if err != nil || len(sha) != 40 || todo.CommitSHA != sha {
		t.Fatalf("Attach = %q %v", sha, err)
	}
	var got, gotSub models.Todo
	db.First(&got, todo.ID)
	db.First(&gotSub, sub.ID)
	if got.CommitSHA != sha || got.CommitRepo != dir || gotSub.CommitSHA != sha {
		t.Fatalf("commit tersimpan di todo & sub todo: %+v / %+v", got, gotSub)
	}
	other := models.Todo{ProjectID: &p.ID, Title: "Lain", Status: models.TodoStatusCompleted}
	db.Create(&other)
	if _, err := Attach(db, &other, ""); err != ErrNoCommit {
		t.Errorf("todo tanpa commit squash: %v", err)
	}
	if s, err := Attach(db, &other, sha[:10]); err != nil || s != sha {
		t.Errorf("SHA yang dilaporkan agent dipakai: %q %v", s, err)
	}
}
