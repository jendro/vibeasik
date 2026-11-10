// Aplikasi contoh yang sengaja error (net/http + SDK traffic). Dijalankan oleh uji ujung-ke-ujung di pkg/api.
// Mengirim 3 request ke /boom (panic) dan 2 ke /ok ke dirinya sendiri, lalu mengirim buffer SDK ke engine.
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	vt "github.com/vibeasik/vibeasik/pkg/traffic/sdk/go"
)

type DBError struct{ ID int }

func (e *DBError) Error() string { return fmt.Sprintf("db down id=%d", e.ID) }

func loadUser(id int, token string) {
	_ = token // nilai argumen ini tidak boleh muncul di stack
	panic(&DBError{ID: id})
}

func main() {
	var n atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/boom", func(w http.ResponseWriter, r *http.Request) {
		loadUser(int(n.Add(1)), r.Header.Get("Authorization"))
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "ok") })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: vt.Middleware(mux), ErrorLog: log.New(io.Discard, "", 0)}
	go func() { _ = srv.Serve(ln) }()

	base := "http://" + ln.Addr().String()
	for _, p := range []string{"/boom", "/boom", "/ok", "/boom", "/ok"} {
		req, _ := http.NewRequest(http.MethodPost, base+p+"?token=SECRETQ", strings.NewReader(`{"password":"SECRETB"}`))
		req.Header.Set("Authorization", "Bearer SECRETH")
		req.Header.Set("Cookie", "sid=SECRETC")
		if resp, err := http.DefaultClient.Do(req); err == nil {
			_ = resp.Body.Close()
		}
	}
	vt.Flush()
	fmt.Println("selesai")
	os.Exit(0)
}
