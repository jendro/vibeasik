package todochange

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/diagnostics"
)

// Change adalah satu commit todo yang dibaca ulang dari git.
type Change struct {
	SHA     string `json:"sha"`
	Short   string `json:"short"`
	Subject string `json:"subject"`
	Body    string `json:"body,omitempty"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	// Repo: nama folder repo (path lengkap tidak dikirim ke tampilan).
	Repo      string       `json:"repo"`
	Added     int          `json:"added"`
	Deleted   int          `json:"deleted"`
	Files     []FileChange `json:"files"`
	Truncated bool         `json:"truncated,omitempty"`
}

// FileChange: perubahan satu berkas pada commit.
type FileChange struct {
	Path    string `json:"path"`
	OldPath string `json:"old_path,omitempty"`
	// Status: added, modified, deleted, renamed.
	Status  string `json:"status"`
	Added   int    `json:"added"`
	Deleted int    `json:"deleted"`
	Binary  bool   `json:"binary,omitempty"`
	// Functions: fungsi bernama yang disentuh perubahan, urut kemunculan.
	Functions []string `json:"functions,omitempty"`
	Hunks     []Hunk   `json:"hunks,omitempty"`
	Truncated bool     `json:"truncated,omitempty"`
}

// Hunk: satu bagian diff berurutan.
type Hunk struct {
	Header   string `json:"header"`
	Function string `json:"function,omitempty"`
	Lines    []Line `json:"lines"`
}

// Line: satu baris diff. Kind: "add", "del", atau "ctx". Old/New: nomor baris di berkas lama/baru (0 = tidak ada).
type Line struct {
	Kind string `json:"kind"`
	Old  int    `json:"old,omitempty"`
	New  int    `json:"new,omitempty"`
	Text string `json:"text"`
}

const (
	maxFiles     = 60
	maxFileLines = 400
	maxPatch     = 1 << 20
)

// Read membaca commit sha di repo sebagai perubahan per berkas & fungsi.
func Read(repo, sha string) (*Change, error) {
	if !reSHA.MatchString(sha) {
		return nil, fmt.Errorf("SHA commit %q tidak valid", sha)
	}
	head, err := git(repo, "show", "-s", "--format=%H%x00%h%x00%an%x00%aI%x00%s%x00%b", sha)
	if err != nil {
		return nil, err
	}
	f := strings.SplitN(strings.TrimRight(head, "\n"), "\x00", 6)
	if len(f) < 6 {
		return nil, fmt.Errorf("commit %s tidak terbaca", sha)
	}
	c := &Change{SHA: f[0], Short: f[1], Author: f[2], Date: f[3], Subject: f[4], Body: strings.TrimSpace(f[5]), Repo: filepath.Base(repo)}
	patch, err := git(repo, "show", "--format=", "--no-color", "--no-ext-diff", "-M", "-U3", c.SHA)
	if err != nil {
		return nil, err
	}
	if len(patch) > maxPatch {
		patch, c.Truncated = patch[:maxPatch], true
	}
	c.Files = parsePatch(patch)
	if len(c.Files) > maxFiles {
		c.Files, c.Truncated = c.Files[:maxFiles], true
	}
	for i := range c.Files {
		fc := &c.Files[i]
		c.Added += fc.Added
		c.Deleted += fc.Deleted
		nameFunctions(repo, c.SHA, fc)
	}
	return c, nil
}

var (
	reDiffGit  = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	reHunkHead = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@ ?(.*)$`)
)

// parsePatch membaca keluaran `git show` menjadi berkas & bagian diff.
func parsePatch(patch string) []FileChange {
	var files []FileChange
	var cur *FileChange
	var oldN, newN int
	for _, l := range strings.Split(patch, "\n") {
		if m := reDiffGit.FindStringSubmatch(l); m != nil {
			files = append(files, FileChange{Path: m[2], Status: "modified"})
			cur = &files[len(files)-1]
			if m[1] != m[2] {
				cur.OldPath = m[1]
			}
			continue
		}
		if cur == nil {
			continue
		}
		switch {
		case strings.HasPrefix(l, "new file mode"):
			cur.Status = "added"
		case strings.HasPrefix(l, "deleted file mode"):
			cur.Status = "deleted"
		case strings.HasPrefix(l, "rename from "):
			cur.Status = "renamed"
		case strings.HasPrefix(l, "Binary files "):
			cur.Binary = true
		case strings.HasPrefix(l, "--- "), strings.HasPrefix(l, "+++ "), strings.HasPrefix(l, "index "), strings.HasPrefix(l, "similarity index"), strings.HasPrefix(l, "rename to "):
		case strings.HasPrefix(l, "@@"):
			m := reHunkHead.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			oldN, _ = strconv.Atoi(m[1])
			newN, _ = strconv.Atoi(m[2])
			cur.Hunks = append(cur.Hunks, Hunk{Header: l, Function: strings.TrimSpace(m[3])})
		case len(cur.Hunks) > 0:
			oldN, newN = addLine(cur, l, oldN, newN)
		}
	}
	return files
}

// addLine menambah satu baris diff ke bagian terakhir berkas dan memajukan nomor baris lama/baru.
func addLine(fc *FileChange, l string, oldN, newN int) (int, int) {
	h := &fc.Hunks[len(fc.Hunks)-1]
	var ln Line
	switch {
	case strings.HasPrefix(l, "+"):
		ln = Line{Kind: "add", New: newN, Text: l[1:]}
		fc.Added++
		newN++
	case strings.HasPrefix(l, "-"):
		ln = Line{Kind: "del", Old: oldN, Text: l[1:]}
		fc.Deleted++
		oldN++
	case strings.HasPrefix(l, " "):
		ln = Line{Kind: "ctx", Old: oldN, New: newN, Text: l[1:]}
		oldN++
		newN++
	default: // "\ No newline at end of file" dan baris kosong penutup
		return oldN, newN
	}
	if countLines(fc) >= maxFileLines {
		fc.Truncated = true
		return oldN, newN
	}
	h.Lines = append(h.Lines, ln)
	return oldN, newN
}

func countLines(fc *FileChange) int {
	n := 0
	for _, h := range fc.Hunks {
		n += len(h.Lines)
	}
	return n
}

// nameFunctions mengisi nama fungsi yang disentuh tiap bagian diff: baris baru dicari di isi berkas sesudah commit,
// baris yang dihapus di isi sebelum commit. Fungsi bagian (Hunk.Function) diganti nama fungsi yang benar-benar
// memuat perubahan; kepala hunk git hanya menebak dari baris di atasnya.
func nameFunctions(repo, sha string, fc *FileChange) {
	if fc.Binary || len(fc.Hunks) == 0 {
		return
	}
	newLines := fileAt(repo, sha, fc.Path, fc.Status != "deleted")
	oldPath := fc.Path
	if fc.OldPath != "" {
		oldPath = fc.OldPath
	}
	oldLines := fileAt(repo, sha+"^", oldPath, fc.Status != "added")
	seen := map[string]bool{}
	for i := range fc.Hunks {
		h := &fc.Hunks[i]
		h.Function = ""
		for _, ln := range h.Lines {
			name := ""
			switch {
			case ln.Kind == "add" && newLines != nil:
				name = funcAt(fc.Path, newLines, ln.New)
			case ln.Kind == "del" && oldLines != nil:
				name = funcAt(oldPath, oldLines, ln.Old)
			}
			if name == "" {
				continue
			}
			if h.Function == "" {
				h.Function = name
			}
			if !seen[name] {
				seen[name] = true
				fc.Functions = append(fc.Functions, name)
			}
		}
	}
}

func funcAt(path string, lines []string, line int) string {
	if line <= 0 {
		return ""
	}
	name, _, _, ok := diagnostics.EnclosingFunction(path, lines, line)
	if !ok {
		return ""
	}
	return name
}

// fileAt: isi berkas path pada rev (nil bila tidak ada atau want=false).
func fileAt(repo, rev, path string, want bool) []string {
	if !want {
		return nil
	}
	out, err := git(repo, "show", rev+":"+path)
	if err != nil || len(out) > 2<<20 {
		return nil
	}
	return strings.Split(out, "\n")
}

// SortedFunctions: semua fungsi yang berubah di commit, urut abjad, untuk ringkasan.
func (c *Change) SortedFunctions() []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range c.Files {
		for _, fn := range f.Functions {
			if !seen[fn] {
				seen[fn] = true
				out = append(out, fn)
			}
		}
	}
	sort.Strings(out)
	return out
}
