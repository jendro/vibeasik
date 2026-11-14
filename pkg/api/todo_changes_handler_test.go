package api

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/todochange"
)

func gitT(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.test", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.test")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

type changesResp struct {
	Commit *struct {
		SHA   string `json:"sha"`
		Files []struct {
			Path      string   `json:"path"`
			Functions []string `json:"functions"`
		} `json:"files"`
	} `json:"commit"`
	Functions []string `json:"functions"`
	Reason    string   `json:"reason"`
	ViaTodoID uint     `json:"via_todo_id"`
}

func getChanges(t *testing.T, s *Server, id uint) changesResp {
	t.Helper()
	resp := doReq(t, s, "GET", fmt.Sprintf("/api/todos/%d/changes", id), "localhost", map[string]string{localauth.Header: testLocalToken})
	body, _ := io.ReadAll(resp.Body)
	var out changesResp
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("respons %d tidak valid: %s", resp.StatusCode, body)
	}
	return out
}

// Todo selesai yang belum mencatat commit menemukan commit squash-nya saat diperiksa, menyebut fungsi yang
// berubah, dan sub todo gabungannya memakai commit induk. Klien tidak bisa menimpa commit lewat PUT.
func TestTodoChangesFindsSquashCommit(t *testing.T) {
	s := syncServer(t)
	if err := s.db.AutoMigrate(&models.Todo{}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	p := models.Project{Name: "Toko", Slug: "toko-changes", RootPath: dir}
	s.db.Create(&p)
	todo := models.Todo{ProjectID: &p.ID, Title: "Tolak nama kosong", Status: models.TodoStatusCompleted}
	s.db.Create(&todo)
	sub := models.Todo{ProjectID: &p.ID, ParentID: &todo.ID, Title: "sub", Status: models.TodoStatusCompleted}
	s.db.Create(&sub)
	pending := models.Todo{ProjectID: &p.ID, Title: "belum", Status: models.TodoStatusPending}
	s.db.Create(&pending)

	gitT(t, dir, "init", "-q", "-b", "main")
	src := "package api\n\nfunc Save(name string) error {\n\tif name == \"\" {\n\t\treturn nil\n\t}\n\treturn nil\n}\n"
	os.WriteFile(filepath.Join(dir, "user.go"), []byte(src), 0o644)
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "chore: awal")
	os.WriteFile(filepath.Join(dir, "user.go"), []byte(strings.Replace(src, "return nil\n\t}", "return errEmpty\n\t}", 1)), 0o644)
	gitT(t, dir, "commit", "-q", "-am", "fix(api): tolak nama kosong", "-m", todochange.SquashTrailer+todochange.BranchPrefix(todo.ID)+"tolak")

	got := getChanges(t, s, todo.ID)
	if got.Commit == nil || len(got.Commit.SHA) != 40 || len(got.Commit.Files) != 1 || got.Commit.Files[0].Path != "user.go" || len(got.Functions) != 1 || got.Functions[0] != "Save" {
		t.Fatalf("perubahan todo: %+v", got)
	}
	var saved models.Todo
	s.db.First(&saved, todo.ID)
	if saved.CommitSHA != got.Commit.SHA || saved.CommitRepo == "" {
		t.Fatalf("commit hasil pencarian disimpan: %+v", saved)
	}
	if viaSub := getChanges(t, s, sub.ID); viaSub.Commit == nil || viaSub.Commit.SHA != got.Commit.SHA {
		t.Errorf("sub todo memakai commit induk: %+v", viaSub)
	}
	if none := getChanges(t, s, pending.ID); none.Commit != nil || !strings.Contains(none.Reason, "belum selesai") {
		t.Errorf("todo belum selesai: %+v", none)
	}

	doReqWithBody(t, s, "PUT", fmt.Sprintf("/api/todos/%d", todo.ID), "localhost", `{"title":"Tolak nama kosong","commit_sha":"deadbeef","commit_repo":"/tmp"}`,
		map[string]string{localauth.Header: testLocalToken})
	s.db.First(&saved, todo.ID)
	if saved.CommitSHA != got.Commit.SHA || saved.CommitRepo == "/tmp" {
		t.Errorf("PUT tidak boleh menimpa commit todo: %+v", saved)
	}
}
