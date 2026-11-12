//go:build windows

package winpanel

import (
	"syscall"
	"unsafe"
)

// Binding Win32 minimal untuk panel (tanpa CGO, tanpa pustaka GUI pihak ketiga).

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pGetMessageW              = user32.NewProc("GetMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pIsDialogMessageW         = user32.NewProc("IsDialogMessageW")
	pPostQuitMessage          = user32.NewProc("PostQuitMessage")
	pPostMessageW             = user32.NewProc("PostMessageW")
	pSendMessageW             = user32.NewProc("SendMessageW")
	pShowWindow               = user32.NewProc("ShowWindow")
	pUpdateWindow             = user32.NewProc("UpdateWindow")
	pDestroyWindow            = user32.NewProc("DestroyWindow")
	pSetWindowPos             = user32.NewProc("SetWindowPos")
	pSetWindowTextW           = user32.NewProc("SetWindowTextW")
	pEnableWindow             = user32.NewProc("EnableWindow")
	pInvalidateRect           = user32.NewProc("InvalidateRect")
	pBeginPaint               = user32.NewProc("BeginPaint")
	pEndPaint                 = user32.NewProc("EndPaint")
	pFillRect                 = user32.NewProc("FillRect")
	pDrawTextW                = user32.NewProc("DrawTextW")
	pDrawIconEx               = user32.NewProc("DrawIconEx")
	pLoadImageW               = user32.NewProc("LoadImageW")
	pLoadCursorW              = user32.NewProc("LoadCursorW")
	pSetCursor                = user32.NewProc("SetCursor")
	pAdjustWindowRectExForDpi = user32.NewProc("AdjustWindowRectExForDpi")
	pAdjustWindowRectEx       = user32.NewProc("AdjustWindowRectEx")
	pGetDpiForWindow          = user32.NewProc("GetDpiForWindow")
	pSetProcessDpiAwareCtx    = user32.NewProc("SetProcessDpiAwarenessContext")
	pSetProcessDPIAware       = user32.NewProc("SetProcessDPIAware")
	pGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	pSystemParametersInfoW    = user32.NewProc("SystemParametersInfoW")
	pMessageBoxW              = user32.NewProc("MessageBoxW")
	pFindWindowW              = user32.NewProc("FindWindowW")
	pSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	pIsIconic                 = user32.NewProc("IsIconic")
	pGetCursorPos             = user32.NewProc("GetCursorPos")
	pScreenToClient           = user32.NewProc("ScreenToClient")

	pCreateFontW      = gdi32.NewProc("CreateFontW")
	pCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	pDeleteObject     = gdi32.NewProc("DeleteObject")
	pSelectObject     = gdi32.NewProc("SelectObject")
	pSetBkMode        = gdi32.NewProc("SetBkMode")
	pSetBkColor       = gdi32.NewProc("SetBkColor")
	pSetTextColor     = gdi32.NewProc("SetTextColor")
	pEllipse          = gdi32.NewProc("Ellipse")
	pGetStockObject   = gdi32.NewProc("GetStockObject")

	pGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
	pCreateMutexW             = kernel32.NewProc("CreateMutexW")
	pCreateJobObjectW         = kernel32.NewProc("CreateJobObjectW")
	pSetInformationJobObject  = kernel32.NewProc("SetInformationJobObject")
	pAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	pTerminateJobObject       = kernel32.NewProc("TerminateJobObject")
	pAttachConsole            = kernel32.NewProc("AttachConsole")
	pFreeConsole              = kernel32.NewProc("FreeConsole")
	pSetConsoleCtrlHandler    = kernel32.NewProc("SetConsoleCtrlHandler")
	pGenerateConsoleCtrlEvent = kernel32.NewProc("GenerateConsoleCtrlEvent")

	pShellExecuteW = shell32.NewProc("ShellExecuteW")
)

const (
	wmCreate         = 0x0001
	wmDestroy        = 0x0002
	wmPaint          = 0x000F
	wmClose          = 0x0010
	wmSetFont        = 0x0030
	wmSetCursor      = 0x0020
	wmCommand        = 0x0111
	wmCtlColorEdit   = 0x0133
	wmCtlColorStatic = 0x0138
	wmLButtonUp      = 0x0202
	wmDpiChanged     = 0x02E0
	wmApp            = 0x8000

	wsOverlapped  = 0x00000000
	wsCaption     = 0x00C00000
	wsSysMenu     = 0x00080000
	wsMinimizeBox = 0x00020000
	wsChild       = 0x40000000
	wsVisible     = 0x10000000
	wsTabStop     = 0x00010000
	wsVScroll     = 0x00200000
	wsExClientEdg = 0x00000200

	bsPushButton    = 0x0
	bsDefPushButton = 0x1
	esMultiline     = 0x0004
	esAutoVScroll   = 0x0040
	esReadOnly      = 0x0800

	emSetSel       = 0x00B1
	emScrollCaret  = 0x00B7
	emReplaceSel   = 0x00C2
	emSetLimitText = 0x00C5
	wmGetTextLen   = 0x000E

	swShow       = 5
	swRestore    = 9
	cwUseDefault = 0x80000000

	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010

	dtLeft       = 0x0000
	dtRight      = 0x0002
	dtVCenter    = 0x0004
	dtWordBreak  = 0x0010
	dtSingleLine = 0x0020
	dtNoPrefix   = 0x0800
	dtEndEllipse = 0x8000

	imageIcon    = 1
	idcArrow     = 32512
	idcHand      = 32649
	smCxIcon     = 11
	smCxSmIcon   = 49
	diNormal     = 3
	nullPen      = 8
	transparent  = 1
	fwNormal     = 400
	fwSemiBold   = 600
	fwBold       = 700
	cleartypeQ   = 5
	defCharset   = 1
	mbOK         = 0x0
	mbIconError  = 0x10
	mbIconWarn   = 0x30
	spiNCMetrics = 0x0029

	createNoWindow        = 0x08000000
	createNewProcessGroup = 0x00000200
	ctrlBreakEvent        = 1

	jobObjectExtendedLimitInformation = 9
	jobObjectLimitKillOnJobClose      = 0x2000
)

type point struct{ X, Y int32 }

type rect struct{ Left, Top, Right, Bottom int32 }

type msg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type paintStruct struct {
	Hdc         uintptr
	Erase       int32
	RcPaint     rect
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}

type jobBasicLimit struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type ioCounters struct{ R, W, O, RT, WT, OT uint64 }

type jobExtendedLimit struct {
	Basic                 jobBasicLimit
	IO                    ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func rgb(r, g, b uint8) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }

func sendMessage(h uintptr, m uint32, w, l uintptr) uintptr {
	r, _, _ := pSendMessageW.Call(h, uintptr(m), w, l)
	return r
}

func setText(h uintptr, s string) { pSetWindowTextW.Call(h, uintptr(unsafe.Pointer(u16(s)))) }

func enable(h uintptr, on bool) {
	v := uintptr(0)
	if on {
		v = 1
	}
	pEnableWindow.Call(h, v)
}

func messageBox(owner uintptr, text, title string, flags uintptr) {
	pMessageBoxW.Call(owner, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(title))), flags)
}

func shellOpen(target string) {
	pShellExecuteW.Call(0, uintptr(unsafe.Pointer(u16("open"))), uintptr(unsafe.Pointer(u16(target))), 0, 0, swShow)
}

// newFont membuat font Segoe UI; pt dalam poin, diskalakan sesuai DPI jendela.
func newFont(pt float64, weight int, dpi int, face string) uintptr {
	h := -int32(pt*float64(dpi)/72 + 0.5)
	f, _, _ := pCreateFontW.Call(uintptr(h), 0, 0, 0, uintptr(weight), 0, 0, 0,
		defCharset, 0, 0, cleartypeQ, 0, uintptr(unsafe.Pointer(u16(face))))
	return f
}
