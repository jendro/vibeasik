// Package todochange mencatat commit hasil sebuah todo dan membacanya kembali sebagai perubahan yang bisa
// diperiksa pengguna: berkas, fungsi yang berubah, dan diff per bagian.
//
// Commit todo dikenali dari trailer squash "Vibeasik-Branch: vibeasik/todo-<id>-…" yang ditulis alur kerja todo,
// atau dari SHA yang dilaporkan agent lewat update_todo. SHA disimpan utuh (40 karakter) agar diff bisa diulang
// dan diverifikasi; tampilan boleh memendekkannya.
package todochange

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
	"gorm.io/gorm"
)

// SquashTrailer: trailer commit squash alur kerja todo; nilainya nama branch todo.
const SquashTrailer = "Vibeasik-Branch: "

// BranchPrefix: awal nama branch todo #id ("vibeasik/todo-<id>-<judul>").
func BranchPrefix(todoID uint) string { return fmt.Sprintf("vibeasik/todo-%d-", todoID) }

const gitTimeout = 20 * time.Second

var reSHA = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

// ErrNoCommit: commit todo tidak ditemukan di repo proyek.
var ErrNoCommit = errors.New("commit todo tidak ditemukan di repo proyek")

func git(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// ProjectRepos: repo git yang dipakai proyek — akar proyek dan folder tiap komponen (proyek multi-repo), tanpa
// duplikat, urut akar dulu.
func ProjectRepos(p models.Project) []string {
	dirs := []string{p.RootPath}
	for _, c := range p.Components {
		dirs = append(dirs, c.AbsolutePath)
	}
	seen := map[string]bool{}
	var out []string
	for _, d := range dirs {
		if d == "" || d == "-" {
			continue
		}
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
			continue
		}
		top, err := git(d, "rev-parse", "--show-toplevel")
		if top = strings.TrimSpace(top); err != nil || top == "" || seen[top] {
			continue
		}
		seen[top] = true
		out = append(out, top)
	}
	return out
}

// FindSquash mencari commit squash todo (trailer SquashTrailer + branch todo) di riwayat HEAD tiap repo; yang
// terbaru menang bila todo pernah di-squash lebih dari sekali.
func FindSquash(repos []string, todoID uint) (repo, sha string, err error) {
	var bestAt int64
	for _, r := range repos {
		out, gerr := git(r, "log", "-n", "1", "--format=%H %ct", "-F", "--grep="+SquashTrailer+BranchPrefix(todoID), "HEAD")
		h, at, ok := strings.Cut(strings.TrimSpace(out), " ")
		if gerr != nil || !ok {
			continue
		}
		var ts int64
		fmt.Sscan(at, &ts)
		if sha == "" || ts > bestAt {
			repo, sha, bestAt = r, h, ts
		}
	}
	if sha == "" {
		return "", "", ErrNoCommit
	}
	return repo, sha, nil
}

// Resolve mencari commit ref (SHA penuh atau pendek) di repo proyek dan mengembalikan SHA utuhnya.
func Resolve(repos []string, ref string) (repo, sha string, err error) {
	ref = strings.TrimSpace(ref)
	if !reSHA.MatchString(ref) {
		return "", "", fmt.Errorf("SHA commit %q tidak valid: pakai 7–40 karakter heksadesimal", ref)
	}
	for _, r := range repos {
		if out, gerr := git(r, "rev-parse", "--verify", "--quiet", ref+"^{commit}"); gerr == nil {
			return r, strings.TrimSpace(out), nil
		}
	}
	return "", "", fmt.Errorf("commit %s tidak ada di repo proyek (%s)", ref, strings.Join(repoNames(repos), ", "))
}

func repoNames(repos []string) []string {
	out := make([]string, 0, len(repos))
	for _, r := range repos {
		out = append(out, filepath.Base(r))
	}
	if len(out) == 0 {
		return []string{"proyek belum punya repo git"}
	}
	return out
}

// Attach menyimpan commit todo (dan sub todo gabungannya yang belum punya commit). ref kosong = cari commit
// squash todo itu. Mengembalikan SHA utuh yang tersimpan.
func Attach(db *gorm.DB, todo *models.Todo, ref string) (string, error) {
	repos := todoRepos(db, *todo)
	var repo, sha string
	var err error
	if strings.TrimSpace(ref) != "" {
		repo, sha, err = Resolve(repos, ref)
	} else {
		repo, sha, err = FindSquash(repos, todo.ID)
	}
	if err != nil {
		return "", err
	}
	return sha, Save(db, todo, repo, sha)
}

// Save mencatat commit todo yang sudah pasti (mis. hasil verifikasi squash orkestrator) beserta sub todonya.
func Save(db *gorm.DB, todo *models.Todo, repo, sha string) error {
	up := map[string]interface{}{"commit_sha": sha, "commit_repo": repo}
	if err := db.Model(&models.Todo{}).Where("id = ?", todo.ID).Updates(up).Error; err != nil {
		return err
	}
	todo.CommitSHA, todo.CommitRepo = sha, repo
	return db.Model(&models.Todo{}).Where("parent_id = ? AND (commit_sha = '' OR commit_sha IS NULL)", todo.ID).Updates(up).Error
}

// todoRepos: repo proyek todo; tanpa proyek, repo commit yang sudah tercatat.
func todoRepos(db *gorm.DB, t models.Todo) []string {
	var repos []string
	if t.ProjectID != nil {
		var p models.Project
		if db.Preload("Components").First(&p, *t.ProjectID).Error == nil {
			repos = ProjectRepos(p)
		}
	}
	if t.CommitRepo != "" && !contains(repos, t.CommitRepo) {
		repos = append(repos, t.CommitRepo)
	}
	return repos
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
