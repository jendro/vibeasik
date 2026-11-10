package vibeasiktraffic

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// Event "error" dan "heartbeat". Env tambahan: VIBEASIK_RELEASE (versi/commit rilis, maks 64 karakter).
// Stack hanya berisi nama fungsi + berkas:baris; tidak pernah nilai variabel, body, header, maupun query.

const (
	kindError      = "error"
	kindHeartbeat  = "heartbeat"
	heartbeatEvery = 30 * time.Second
	maxStackFrames = 40
	maxStackChars  = 8000
)

var (
	release = clipStr(os.Getenv("VIBEASIK_RELEASE"), 64)
	bootAt  = time.Now()
)

func clipStr(v string, n int) string {
	if len(v) > n {
		return v[:n]
	}
	return v
}

// push memasukkan event ke buffer tanpa filter path (dipakai error & heartbeat). Tidak pernah memblokir.
func push(ev Event) {
	if ev.Release == "" {
		ev.Release = release
	}
	mu.Lock()
	if len(buf) < maxBuffer {
		buf = append(buf, ev)
	}
	mu.Unlock()
}

// Heartbeat mencatat status proses (rilis, memori, uptime). Dipanggil tiap 30 detik oleh goroutine latar.
func Heartbeat() {
	if !Enabled() {
		return
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	push(Event{
		Kind: kindHeartbeat, TS: time.Now().UnixMilli(), Instance: instance,
		MemBytes: int64(ms.Sys), UptimeS: int64(time.Since(bootAt).Seconds()),
	})
}

func heartbeatLoop() {
	Heartbeat()
	for range time.Tick(heartbeatEvery) {
		Heartbeat()
	}
}

// ensureStarted menyalakan pengirim berkala + heartbeat (idempoten) sehingga heartbeat jalan walau belum ada request.
func ensureStarted() { once.Do(start) }

// panicStack mengambil frame setelah runtime.gopanic (tempat panic terjadi) sampai maxStackFrames.
// Harus dipanggil dari fungsi yang dijalankan oleh defer saat panic.
func panicStack() string {
	pcs := make([]uintptr, 64)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var lines []string
	seenPanic := false
	for {
		f, more := frames.Next()
		switch {
		case !seenPanic:
			seenPanic = f.Function == "runtime.gopanic"
		case len(lines) == 0 && strings.HasPrefix(f.Function, "runtime."):
			// panicmem/sigpanic dll: bukan frame aplikasi
		case len(lines) < maxStackFrames:
			lines = append(lines, fmt.Sprintf("%s (%s:%d)", f.Function, f.File, f.Line))
		}
		if !more {
			break
		}
	}
	return clipStr(strings.Join(lines, "\n"), maxStackChars)
}

// RecordPanic melaporkan panic yang tertangkap lewat recover(). Panggil langsung dari fungsi defer.
func RecordPanic(p any, method, route, path string, status int) {
	if !Enabled() || p == http.ErrAbortHandler {
		return
	}
	push(Event{
		Kind: kindError, TS: time.Now().UnixMilli(), Method: method, Route: route, Path: path, Status: status,
		ErrType: clipStr(fmt.Sprintf("%T", p), 128), ErrMsg: clipStr(fmt.Sprint(p), 500), Stack: panicStack(), Side: "server",
	})
}

// RecordError melaporkan error yang dikembalikan handler (tanpa stack; dikelompokkan berdasar tipe + pesan).
func RecordError(err error, method, route, path string, status int) {
	if !Enabled() || err == nil {
		return
	}
	push(Event{
		Kind: kindError, TS: time.Now().UnixMilli(), Method: method, Route: route, Path: path, Status: status,
		ErrType: clipStr(fmt.Sprintf("%T", err), 128), ErrMsg: clipStr(err.Error(), 500), Side: "server",
	})
}
