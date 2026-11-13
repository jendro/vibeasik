package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Pelajaran Desain (before/after): satu contoh perbaikan desain — gambar awal (desain kurang tepat), gambar akhir
// (desain yang diperbaiki), dan narasi konten (mis. teks konten Instagram "warna A + B bagus, + C lebih bagus").
// Berbeda dengan Galeri Desain yang hanya mengamati tanpa menilai, pelajaran ini memang membandingkan: LLM vision
// mencatat kesalahan prinsip pada gambar awal, perubahan pada gambar akhir, dan prinsip yang bisa dipakai ulang.
// Gambar tetap di mesin pengguna; yang masuk Pusat Pengetahuan hanya teks pelajarannya.

//
// Jenis kedua, Teori desain (Kind = theory): satu konsep teori (mis. hierarki visual) + keterangan opsional + beberapa
// contoh gambar. Tiap contoh berupa gambar TUNGGAL bertanda benar atau salah, atau KOMPARASI (gambar benar + gambar
// salah berdampingan). LLM vision mempelajari apa yang membuat contoh benar menerapkan teori itu dan apa yang dilanggar
// contoh salah.

const (
	LessonSideBefore = "before"
	LessonSideAfter  = "after"

	LessonKindBeforeAfter = "before_after"
	LessonKindTheory      = "theory"

	LessonExampleSingle  = "single"  // satu gambar: GoodFile (benar) atau BadFile (salah)
	LessonExampleCompare = "compare" // dua gambar: GoodFile (benar) + BadFile (salah)

	LessonSlotGood = "good"
	LessonSlotBad  = "bad"
)

// DesignLessonExample: satu contoh pada pelajaran teori. Mode single mengisi tepat satu dari GoodFile/BadFile.
type DesignLessonExample struct {
	ID       string `json:"id"`
	Mode     string `json:"mode"` // single | compare
	Note     string `json:"note,omitempty"`
	GoodFile string `json:"good_file,omitempty"`
	BadFile  string `json:"bad_file,omitempty"`
}

type DesignLesson struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// Kind: before_after (bawaan; kosong = before_after untuk baris lama) | theory.
	Kind      string   `gorm:"size:20;index" json:"kind"`
	Title     string   `gorm:"size:200" json:"title"`
	Narrative string   `gorm:"type:text" json:"narrative"`  // narasi konten apa adanya (caption / transkrip)
	SourceURL string   `gorm:"size:1000" json:"source_url"` // opsional: tautan konten asal (Instagram, Reels, ...)
	Platform  string   `gorm:"size:20" json:"platform"`     // web | android | ios | graphic (poster/feed/umum)
	Tags      []string `gorm:"-" json:"tags"`
	TagsJSON  string   `gorm:"type:text" json:"-"`

	BeforeFile string `gorm:"size:255" json:"before_file,omitempty"` // nama berkas di designlib.DataDir
	AfterFile  string `gorm:"size:255" json:"after_file,omitempty"`

	// Pelajaran teori: id konsep katalog teori (designlib theory.json) + contoh gambar benar/salah.
	TheoryID     string                `gorm:"size:80;index" json:"theory_id,omitempty"`
	Examples     []DesignLessonExample `gorm:"-" json:"examples"`
	ExamplesJSON string                `gorm:"type:text" json:"-"`

	Status    string `gorm:"size:20;index" json:"status"` // new | analyzed | failed (sama dengan DesignStatus*)
	LastError string `gorm:"type:text" json:"last_error,omitempty"`

	Analysis      *DesignLessonAnalysis `gorm:"-" json:"analysis,omitempty"`
	AnalysisJSON  string                `gorm:"type:text" json:"-"`
	AnalysisModel string                `gorm:"size:200" json:"analysis_model,omitempty"`
	AnalyzedAt    *time.Time            `json:"analyzed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DesignLessonAnalysis adalah hasil LLM vision atas pasangan gambar + narasi.
type DesignLessonAnalysis struct {
	Summary    string                  `json:"summary"`     // apa yang berubah dan mengapa lebih baik, 2-3 kalimat
	BeforeDesc string                  `json:"before_desc"` // deskripsi gambar awal
	AfterDesc  string                  `json:"after_desc"`  // deskripsi gambar akhir
	Mistakes   []DesignLessonMistake   `json:"mistakes"`    // kesalahan prinsip pada gambar awal
	Changes    []DesignLessonChange    `json:"changes"`     // perubahan yang dilakukan pada gambar akhir
	Principles []DesignLessonPrinciple `json:"principles"`  // prinsip yang bisa dipakai ulang
	Palette    *DesignLessonPalette    `json:"palette,omitempty"`
	Takeaway   string                  `json:"takeaway"` // satu kalimat inti pelajaran
	// Pelajaran teori: yang membuat contoh benar menerapkan teori dengan baik (pelanggaran contoh salah ada di Mistakes,
	// beda pasangan komparasi di Changes).
	Strengths []DesignLessonStrength `json:"strengths,omitempty"`
	// Matched: prinsip sudah dicocokkan dengan pelajaran sebelumnya (MatchLessonPrinciples).
	Matched bool `json:"matched,omitempty"`
}

type DesignLessonMistake struct {
	Area      string `json:"area"`      // models.DesignAreas
	Principle string `json:"principle"` // prinsip yang dilanggar, mis. "hierarki visual"
	Problem   string `json:"problem"`   // apa yang salah
	Evidence  string `json:"evidence"`  // yang terlihat di gambar awal
}

// DesignLessonStrength: penerapan teori yang benar pada satu contoh.
type DesignLessonStrength struct {
	Area     string `json:"area"`
	Point    string `json:"point"`    // apa yang benar
	Evidence string `json:"evidence"` // yang terlihat di gambar (sebut nomor contoh)
}

type DesignLessonChange struct {
	Area   string `json:"area"`
	Change string `json:"change"` // apa yang diubah
	Why    string `json:"why"`    // mengapa membuatnya lebih baik
}

type DesignLessonPrinciple struct {
	Area       string `json:"area"`
	Rule       string `json:"rule"`         // instruksi yang bisa dipakai ulang (imperatif)
	HowToApply string `json:"how_to_apply"` // nilai konkret CSS/Tailwind/warna bila ada
	TheoryID   string `json:"theory_id,omitempty"`
	// SameAs: kunci node prinsip pelajaran sebelumnya yang menyatakan aturan yang sama (diputuskan LLM) — prinsip ini
	// lalu memperkuat node itu alih-alih membuat node baru.
	SameAs string `json:"same_as,omitempty"`
}

type DesignLessonPalette struct {
	Before []DesignColor `json:"before"`
	After  []DesignColor `json:"after"`
	Notes  string        `json:"notes"` // mis. "A+B harmonis; tambahan C sebagai aksen membuat CTA menonjol"
}

// IsTheory: pelajaran jenis teori desain.
func (l *DesignLesson) IsTheory() bool { return l.Kind == LessonKindTheory }

// ExampleFiles: semua berkas gambar contoh pelajaran teori.
func (l *DesignLesson) ExampleFiles() []string {
	var out []string
	for _, e := range l.Examples {
		for _, f := range []string{e.GoodFile, e.BadFile} {
			if f != "" {
				out = append(out, f)
			}
		}
	}
	return out
}

func (l *DesignLesson) BeforeSave(*gorm.DB) error {
	if l.Tags == nil {
		l.Tags = []string{}
	}
	b, _ := json.Marshal(l.Tags)
	l.TagsJSON = string(b)
	if l.Examples == nil {
		l.Examples = []DesignLessonExample{}
	}
	b, _ = json.Marshal(l.Examples)
	l.ExamplesJSON = string(b)
	if l.Kind == "" {
		l.Kind = LessonKindBeforeAfter
	}
	l.AnalysisJSON = ""
	if l.Analysis != nil {
		if b, err := json.Marshal(l.Analysis); err == nil {
			l.AnalysisJSON = string(b)
		}
	}
	if l.Status == "" {
		l.Status = DesignStatusNew
	}
	return nil
}

func (l *DesignLesson) AfterFind(*gorm.DB) error {
	l.Tags = []string{}
	_ = json.Unmarshal([]byte(l.TagsJSON), &l.Tags)
	l.Examples = []DesignLessonExample{}
	if l.ExamplesJSON != "" {
		_ = json.Unmarshal([]byte(l.ExamplesJSON), &l.Examples)
	}
	if l.Kind == "" {
		l.Kind = LessonKindBeforeAfter
	}
	if l.AnalysisJSON != "" {
		l.Analysis = &DesignLessonAnalysis{}
		_ = json.Unmarshal([]byte(l.AnalysisJSON), l.Analysis)
	}
	return nil
}
