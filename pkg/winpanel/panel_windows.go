//go:build windows

// Package winpanel adalah jendela kendali vibeasik untuk Windows: menyalakan/menghentikan engine dan
// membuka Studio dengan dua tombol, tanpa PowerShell. Dibangun langsung di atas Win32 (tanpa CGO).
package winpanel

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/localauth"
)

const (
	className = "vibeasikPanel"
	title     = "vibeasik Control Panel"
	website   = "https://vibeasik.com"

	idRun  = 101
	idOpen = 102

	wmAppLog    = wmApp + 1
	wmAppStatus = wmApp + 2
	wmAppExited = wmApp + 3

	wmCtlColorBtn = 0x0135
	wmPrintClient = 0x0318
	emSetMargins  = 0x00D3
	ecLeftMargin  = 0x1
	ecRightMargin = 0x2
	swpNoMove     = 0x0002
	maxLogChars   = 200_000
)

type state int

const (
	stStopped state = iota
	stStarting
	stRunning
	stExternal
	stStopping
)

type panel struct {
	hwnd, btnRun, btnOpen, edit uintptr
	dpi                         int
	fTitle, fSub, fBody, fBold  uintptr
	fSmall, fMono               uintptr
	brBg, brHeader, brWhite     uintptr
	icon                        uintptr
	linkRect                    rect

	engine string
	port   int
	job    uintptr

	mu      sync.Mutex
	version string
	logBuf  strings.Builder
	st      state
	cmd     *exec.Cmd
	exited  chan struct{}
	closing bool
}

var p = &panel{}

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

// Run menampilkan panel dan mengembalikan kode keluar proses.
func Run() int {
	runtime.LockOSThread()
	// CTRL_BREAK yang dikirim ke engine juga sampai ke panel; tangkap agar panel tidak ikut berhenti.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		for range sig {
		}
	}()

	dpiSet := false
	if pSetProcessDpiAwareCtx.Find() == nil {
		r, _, _ := pSetProcessDpiAwareCtx.Call(^uintptr(3)) // -4 = PER_MONITOR_AWARE_V2
		dpiSet = r != 0
	}
	if !dpiSet && pSetProcessDPIAware.Find() == nil {
		pSetProcessDPIAware.Call()
	}
	if _, _, err := pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(u16(`Local\vibeasik-panel`)))); err == syscall.Errno(183) {
		if h, _, _ := pFindWindowW.Call(uintptr(unsafe.Pointer(u16(className))), 0); h != 0 {
			pShowWindow.Call(h, swRestore)
			pSetForegroundWindow.Call(h)
		}
		return 0
	}

	exe, err := os.Executable()
	if err != nil {
		messageBox(0, err.Error(), title, mbIconError)
		return 1
	}
	p.engine = filepath.Join(filepath.Dir(exe), "vibeasik.exe")
	if _, err := os.Stat(p.engine); err != nil {
		messageBox(0, "vibeasik.exe tidak ditemukan di folder yang sama dengan panel ini.\n\n"+
			"Ekstrak seluruh isi .zip ke satu folder, lalu buka panel dari sana.", title, mbIconError)
		return 1
	}
	p.port = config.GetConfig().Port
	p.version = "…"
	p.job = newKillOnCloseJob()

	inst, _, _ := pGetModuleHandleW.Call(0)
	cursor, _, _ := pLoadCursorW.Call(0, idcArrow)
	bigIcon := loadIcon(inst, sysMetric(smCxIcon))
	smallIcon := loadIcon(inst, sysMetric(smCxSmIcon))
	p.brBg, _, _ = pCreateSolidBrush.Call(rgb(248, 250, 252))
	p.brHeader, _, _ = pCreateSolidBrush.Call(rgb(11, 16, 32))
	p.brWhite, _, _ = pCreateSolidBrush.Call(rgb(255, 255, 255))

	wc := wndClassEx{
		WndProc: syscall.NewCallback(wndProc), Instance: inst, Icon: bigIcon, Cursor: cursor,
		Background: p.brBg, ClassName: u16(className), IconSm: smallIcon,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if r, _, err := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		messageBox(0, "Gagal membuat jendela: "+err.Error(), title, mbIconError)
		return 1
	}
	style := uintptr(wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox)
	hwnd, _, err := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16(className))), uintptr(unsafe.Pointer(u16(title))), style,
		cwUseDefault, cwUseDefault, 640, 520, 0, 0, inst, 0)
	if hwnd == 0 {
		messageBox(0, "Gagal membuat jendela: "+err.Error(), title, mbIconError)
		return 1
	}
	pShowWindow.Call(hwnd, swShow)
	pUpdateWindow.Call(hwnd)

	go p.loadVersion()
	go p.pollStatus()

	var m msg
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if d, _, _ := pIsDialogMessageW.Call(hwnd, uintptr(unsafe.Pointer(&m))); d != 0 {
			continue
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return 0
}

func sysMetric(i int) int {
	r, _, _ := pGetSystemMetrics.Call(uintptr(i))
	return int(r)
}

// loadIcon memuat ikon aplikasi (resource ID 1) pada ukuran tertentu; 0 bila exe tanpa resource.
func loadIcon(inst uintptr, size int) uintptr {
	h, _, _ := pLoadImageW.Call(inst, 1, imageIcon, uintptr(size), uintptr(size), 0)
	return h
}

func (p *panel) s(v int) int32 { return int32(v * p.dpi / 96) }

func wndProc(hwnd, m, wp, lp uintptr) uintptr {
	switch m {
	case wmCreate:
		p.hwnd = hwnd
		p.dpi = 96
		if pGetDpiForWindow.Find() == nil {
			if r, _, _ := pGetDpiForWindow.Call(hwnd); r != 0 {
				p.dpi = int(r)
			}
		}
		p.createControls()
		p.applyDPI(nil)
		p.refresh()
		return 0
	case wmDpiChanged:
		p.dpi = int(wp & 0xFFFF)
		p.applyDPI(*(**rect)(unsafe.Pointer(&lp))) // lParam menunjuk RECT saran Windows
		return 0
	case wmPaint:
		p.paint()
		return 0
	case wmPrintClient:
		p.paintDC(wp)
		return 0
	case wmCommand:
		switch wp & 0xFFFF {
		case idRun:
			p.toggle()
		case idOpen:
			p.openStudio()
		}
		return 0
	case wmCtlColorStatic, wmCtlColorEdit:
		pSetBkColor.Call(wp, rgb(255, 255, 255))
		pSetTextColor.Call(wp, rgb(30, 41, 59))
		return p.brWhite
	case wmCtlColorBtn:
		return p.brBg
	case wmSetCursor:
		if p.cursorOnLink() {
			c, _, _ := pLoadCursorW.Call(0, idcHand)
			pSetCursor.Call(c)
			return 1
		}
	case wmLButtonUp:
		x, y := int32(int16(lp&0xFFFF)), int32(int16((lp>>16)&0xFFFF))
		if inRect(p.linkRect, x, y) {
			shellOpen(website)
		}
		return 0
	case wmAppLog:
		p.flushLog()
		return 0
	case wmAppStatus:
		p.refresh()
		return 0
	case wmAppExited:
		p.refresh()
		p.mu.Lock()
		closing := p.closing
		p.mu.Unlock()
		if closing {
			pDestroyWindow.Call(hwnd)
		}
		return 0
	case wmClose:
		p.mu.Lock()
		own := p.cmd != nil
		p.closing = true
		p.mu.Unlock()
		if own {
			p.stop()
			return 0 // jendela ditutup setelah engine berhenti (wmAppExited)
		}
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, m, wp, lp)
	return r
}

func inRect(r rect, x, y int32) bool {
	return x >= r.Left && x < r.Right && y >= r.Top && y < r.Bottom
}

func (p *panel) cursorOnLink() bool {
	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pScreenToClient.Call(p.hwnd, uintptr(unsafe.Pointer(&pt)))
	return inRect(p.linkRect, pt.X, pt.Y)
}

func (p *panel) createControls() {
	inst, _, _ := pGetModuleHandleW.Call(0)
	btn := func(id int, text string, extra uintptr) uintptr {
		h, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16("BUTTON"))), uintptr(unsafe.Pointer(u16(text))),
			wsChild|wsVisible|wsTabStop|extra, 0, 0, 10, 10, p.hwnd, uintptr(id), inst, 0)
		return h
	}
	p.btnRun = btn(idRun, "Jalankan", bsDefPushButton)
	p.btnOpen = btn(idOpen, "Buka Studio", bsPushButton)
	p.edit, _, _ = pCreateWindowExW.Call(wsExClientEdg, uintptr(unsafe.Pointer(u16("EDIT"))), 0,
		wsChild|wsVisible|wsVScroll|esMultiline|esAutoVScroll|esReadOnly, 0, 0, 10, 10, p.hwnd, 0, inst, 0)
	sendMessage(p.edit, emSetLimitText, maxLogChars*2, 0)
}

// applyDPI menyusun ulang ukuran jendela, font, ikon, dan posisi kontrol untuk DPI saat ini.
func (p *panel) applyDPI(suggested *rect) {
	for _, f := range []uintptr{p.fTitle, p.fSub, p.fBody, p.fBold, p.fSmall, p.fMono} {
		if f != 0 {
			pDeleteObject.Call(f)
		}
	}
	p.fTitle = newFont(18, fwSemiBold, p.dpi, "Segoe UI")
	p.fSub = newFont(9.5, fwNormal, p.dpi, "Segoe UI")
	p.fBody = newFont(9.5, fwNormal, p.dpi, "Segoe UI")
	p.fBold = newFont(11, fwSemiBold, p.dpi, "Segoe UI")
	p.fSmall = newFont(8.5, fwNormal, p.dpi, "Segoe UI")
	p.fMono = newFont(9, fwNormal, p.dpi, "Consolas")
	inst, _, _ := pGetModuleHandleW.Call(0)
	p.icon = loadIcon(inst, int(p.s(48)))
	for _, h := range []uintptr{p.btnRun, p.btnOpen} {
		sendMessage(h, wmSetFont, p.fBody, 1)
	}
	sendMessage(p.edit, wmSetFont, p.fMono, 1)
	m := uintptr(p.s(6))
	sendMessage(p.edit, emSetMargins, ecLeftMargin|ecRightMargin, m|m<<16)

	cw, ch := p.s(600), p.s(468)
	if suggested != nil {
		pSetWindowPos.Call(p.hwnd, 0, uintptr(suggested.Left), uintptr(suggested.Top),
			uintptr(suggested.Right-suggested.Left), uintptr(suggested.Bottom-suggested.Top), swpNoZOrder|swpNoActivate)
	} else {
		r := rect{0, 0, cw, ch}
		style := uintptr(wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox)
		if pAdjustWindowRectExForDpi.Find() == nil {
			pAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(&r)), style, 0, 0, uintptr(p.dpi))
		} else {
			pAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&r)), style, 0, 0)
		}
		pSetWindowPos.Call(p.hwnd, 0, 0, 0, uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), swpNoMove|swpNoZOrder|swpNoActivate)
	}
	move := func(h uintptr, x, y, w, hh int32) {
		pSetWindowPos.Call(h, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(hh), swpNoZOrder|swpNoActivate)
	}
	move(p.btnRun, p.s(20), p.s(156), p.s(150), p.s(34))
	move(p.btnOpen, p.s(180), p.s(156), p.s(150), p.s(34))
	move(p.edit, p.s(20), p.s(228), cw-p.s(40), p.s(160))
	pInvalidateRect.Call(p.hwnd, 0, 1)
}

func (p *panel) drawText(hdc uintptr, font uintptr, color uintptr, text string, r rect, flags uintptr) {
	pSelectObject.Call(hdc, font)
	pSetTextColor.Call(hdc, color)
	t, _ := syscall.UTF16FromString(text)
	pDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&t[0])), uintptr(len(t)-1), uintptr(unsafe.Pointer(&r)), flags|dtNoPrefix)
}

func (p *panel) paint() {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(p.hwnd, uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(p.hwnd, uintptr(unsafe.Pointer(&ps)))
	p.paintDC(hdc)
}

// paintDC menggambar isi jendela ke hdc mana pun (layar, atau WM_PRINTCLIENT untuk tangkapan layar).
func (p *panel) paintDC(hdc uintptr) {
	w := p.s(600)
	body := rect{0, p.s(84), w, p.s(600)}
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&body)), p.brBg)
	pSetBkMode.Call(hdc, transparent)

	// Kepala: latar gelap seperti logo, ikon, nama, versi, tautan situs.
	head := rect{0, 0, w, p.s(84)}
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&head)), p.brHeader)
	if p.icon != 0 {
		pDrawIconEx.Call(hdc, uintptr(p.s(20)), uintptr(p.s(18)), p.icon, uintptr(p.s(48)), uintptr(p.s(48)), 0, 0, diNormal)
	}
	p.mu.Lock()
	version, st, port := p.version, p.st, p.port
	p.mu.Unlock()
	p.drawText(hdc, p.fTitle, rgb(255, 255, 255), "vibeasik", rect{p.s(80), p.s(14), w - p.s(200), p.s(46)}, dtLeft|dtSingleLine|dtVCenter)
	p.drawText(hdc, p.fSub, rgb(148, 163, 184), "Control Panel  ·  "+version, rect{p.s(80), p.s(46), w - p.s(200), p.s(68)}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipse)
	p.linkRect = rect{w - p.s(160), p.s(30), w - p.s(20), p.s(54)}
	p.drawText(hdc, p.fBody, rgb(125, 211, 252), "vibeasik.com ↗", p.linkRect, dtRight|dtSingleLine|dtVCenter)

	// Status engine: titik warna + teks (warna tidak berdiri sendiri, selalu disertai kata).
	var dot uintptr
	var head1, head2 string
	switch st {
	case stStopped:
		dot, head1, head2 = rgb(148, 163, 184), "Engine berhenti", "Klik Jalankan untuk menyalakan vibeasik."
	case stStarting:
		dot, head1, head2 = rgb(217, 119, 6), "Menyalakan engine…", "Menyiapkan database dan Studio, biasanya beberapa detik."
	case stRunning:
		dot, head1, head2 = rgb(22, 163, 74), "Engine berjalan", fmt.Sprintf("Studio siap di http://localhost:%d  ·  klik Buka Studio", port)
	case stExternal:
		dot, head1, head2 = rgb(22, 163, 74), "Engine sudah berjalan", "Dinyalakan di luar panel ini (mis. dari terminal). Buka Studio tetap bisa."
	case stStopping:
		dot, head1, head2 = rgb(217, 119, 6), "Menghentikan engine…", "Menutup dev server yang dijalankan vibeasik."
	}
	br, _, _ := pCreateSolidBrush.Call(dot)
	oldBr, _, _ := pSelectObject.Call(hdc, br)
	np, _, _ := pGetStockObject.Call(nullPen)
	oldPen, _, _ := pSelectObject.Call(hdc, np)
	pEllipse.Call(hdc, uintptr(p.s(20)), uintptr(p.s(105)), uintptr(p.s(32)), uintptr(p.s(117)))
	pSelectObject.Call(hdc, oldBr)
	pSelectObject.Call(hdc, oldPen)
	pDeleteObject.Call(br)
	p.drawText(hdc, p.fBold, rgb(15, 23, 42), head1, rect{p.s(40), p.s(98), w - p.s(20), p.s(124)}, dtLeft|dtSingleLine|dtVCenter)
	p.drawText(hdc, p.fBody, rgb(100, 116, 139), head2, rect{p.s(40), p.s(124), w - p.s(20), p.s(146)}, dtLeft|dtSingleLine|dtVCenter|dtEndEllipse)

	p.drawText(hdc, p.fSmall, rgb(100, 116, 139), "Log engine", rect{p.s(20), p.s(206), w - p.s(20), p.s(226)}, dtLeft|dtSingleLine|dtVCenter)
	foot := "Engine hanya melayani komputer ini (127.0.0.1) dengan token lokal; kode dan data proyek tidak diunggah.\r\n" +
		"Menutup jendela ini menghentikan engine. Hubungkan agent coding lewat Studio atau: vibeasik mcp install"
	p.drawText(hdc, p.fSmall, rgb(100, 116, 139), foot, rect{p.s(20), p.s(398), w - p.s(20), p.s(460)}, dtLeft|dtWordBreak)
}

// refresh menyesuaikan teks & keadaan tombol dengan status engine.
func (p *panel) refresh() {
	p.mu.Lock()
	st := p.st
	p.mu.Unlock()
	run, runOn, openOn := "▶  Jalankan", true, false
	switch st {
	case stStarting:
		run, runOn = "Menyalakan…", false
	case stRunning:
		run, openOn = "■  Hentikan", true
	case stExternal:
		run, runOn, openOn = "■  Hentikan", false, true
	case stStopping:
		run, runOn = "Menghentikan…", false
	}
	setText(p.btnRun, run)
	enable(p.btnRun, runOn)
	enable(p.btnOpen, openOn)
	pInvalidateRect.Call(p.hwnd, 0, 1)
}

func (p *panel) setState(st state) {
	p.mu.Lock()
	changed := p.st != st
	p.st = st
	p.mu.Unlock()
	if changed {
		pPostMessageW.Call(p.hwnd, wmAppStatus, 0, 0)
	}
}

func (p *panel) toggle() {
	p.mu.Lock()
	own, st := p.cmd != nil, p.st
	p.mu.Unlock()
	switch {
	case own && (st == stRunning || st == stStarting):
		p.stop()
	case !own && st == stStopped:
		p.start()
	}
}

func hidden(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow | createNewProcessGroup}
}

func (p *panel) start() {
	cmd := exec.Command(p.engine, "serve")
	cmd.Dir = filepath.Dir(p.engine)
	cmd.Env = config.ChildEnviron() // .env dibaca engine sendiri; jangan wariskan salinan milik panel
	hidden(cmd)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		p.appendLog("Gagal menjalankan engine: " + err.Error())
		return
	}
	if h, err := syscall.OpenProcess(0x0100|0x0001, false, uint32(cmd.Process.Pid)); err == nil {
		pAssignProcessToJobObject.Call(p.job, uintptr(h)) // ikut mati bila panel tertutup paksa
		syscall.CloseHandle(h)
	}
	exited := make(chan struct{})
	p.mu.Lock()
	p.cmd, p.exited = cmd, exited
	p.mu.Unlock()
	p.setState(stStarting)
	p.appendLog(fmt.Sprintf("▶ Menjalankan %s serve (port %d)", p.engine, p.port))
	go p.pipe(stdout)
	go p.pipe(stderr)
	go func() {
		err := cmd.Wait()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		p.mu.Lock()
		p.cmd = nil
		stopping := p.st == stStopping
		p.mu.Unlock()
		close(exited)
		if stopping {
			p.appendLog("■ Engine dihentikan.")
		} else {
			p.appendLog(fmt.Sprintf("■ Engine berhenti sendiri (kode %d). Lihat log di atas.", code))
		}
		p.setState(stStopped)
		pPostMessageW.Call(p.hwnd, wmAppExited, 0, 0)
	}()
}

// stop meminta engine berhenti dengan rapi (CTRL_BREAK, seperti Ctrl+C di terminal) lalu memaksa bila
// tidak berhenti dalam 10 detik.
func (p *panel) stop() {
	p.mu.Lock()
	cmd, exited := p.cmd, p.exited
	p.mu.Unlock()
	if cmd == nil {
		return
	}
	p.setState(stStopping)
	go func() {
		pid := uintptr(cmd.Process.Pid)
		if r, _, _ := pAttachConsole.Call(pid); r != 0 {
			pGenerateConsoleCtrlEvent.Call(ctrlBreakEvent, pid)
			pFreeConsole.Call()
		}
		select {
		case <-exited:
		case <-time.After(10 * time.Second):
			p.appendLog("Engine tidak berhenti dalam 10 detik; dihentikan paksa.")
			pTerminateJobObject.Call(p.job, 1)
		}
	}()
}

func (p *panel) pipe(r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		p.appendLog(sc.Text())
	}
}

// appendLog menambah satu baris ke log; ditampilkan oleh thread UI (wmAppLog).
func (p *panel) appendLog(line string) {
	line = ansi.ReplaceAllString(line, "")
	line = strings.Map(func(r rune) rune {
		if r > 0xFFFF { // emoji tidak tersedia di font Consolas
			return -1
		}
		return r
	}, line)
	p.mu.Lock()
	p.logBuf.WriteString(line)
	p.logBuf.WriteString("\r\n")
	p.mu.Unlock()
	pPostMessageW.Call(p.hwnd, wmAppLog, 0, 0)
}

func (p *panel) flushLog() {
	p.mu.Lock()
	text := p.logBuf.String()
	p.logBuf.Reset()
	p.mu.Unlock()
	if text == "" {
		return
	}
	if n := sendMessage(p.edit, wmGetTextLen, 0, 0); int(n)+len(text) > maxLogChars {
		setText(p.edit, "(log lama dipangkas)\r\n")
	}
	n := sendMessage(p.edit, wmGetTextLen, 0, 0)
	sendMessage(p.edit, emSetSel, n, n)
	sendMessage(p.edit, emReplaceSel, 0, uintptr(unsafe.Pointer(u16(text))))
	sendMessage(p.edit, emScrollCaret, 0, 0)
}

func (p *panel) openStudio() {
	token, err := localauth.Token()
	if err != nil {
		messageBox(p.hwnd, "Token lokal tidak bisa dibaca: "+err.Error(), title, mbIconError)
		return
	}
	shellOpen(fmt.Sprintf("http://localhost:%d/auth?t=%s", p.port, url.QueryEscape(token)))
}

func (p *panel) loadVersion() {
	cmd := exec.Command(p.engine, "version")
	hidden(cmd)
	out, err := cmd.Output()
	v := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(out)), "vibeasik "))
	if err != nil || v == "" {
		v = "versi tidak diketahui"
	}
	p.mu.Lock()
	p.version = v
	p.mu.Unlock()
	pInvalidateRect.Call(p.hwnd, 0, 0)
}

// pollStatus memeriksa /health engine secara berkala, termasuk engine yang dijalankan di luar panel.
func (p *panel) pollStatus() {
	client := &http.Client{Timeout: 800 * time.Millisecond}
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", p.port)
	for {
		ok := false
		if resp, err := client.Get(healthURL); err == nil {
			ok = resp.StatusCode == http.StatusOK
			resp.Body.Close()
		}
		p.mu.Lock()
		own, st := p.cmd != nil, p.st
		p.mu.Unlock()
		switch {
		case st == stStopping:
		case own && ok:
			p.setState(stRunning)
		case own:
			p.setState(stStarting) // sedang menyala atau mulai ulang (mis. setelah wizard)
		case ok:
			p.setState(stExternal)
		default:
			p.setState(stStopped)
		}
		time.Sleep(1500 * time.Millisecond)
	}
}

// newKillOnCloseJob membuat Job Object: semua proses engine (termasuk hasil mulai ulang dan dev server)
// ikut berhenti bila panel tertutup, juga saat panel berhenti tidak wajar.
func newKillOnCloseJob() uintptr {
	job, _, _ := pCreateJobObjectW.Call(0, 0)
	if job == 0 {
		return 0
	}
	var info jobExtendedLimit
	info.Basic.LimitFlags = jobObjectLimitKillOnJobClose
	pSetInformationJobObject.Call(job, jobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
	return job
}
