package main

//go:generate go-winres make --in winres.json

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func init() {
	// Lock the OS thread immediately so Win32 message pumps stay bound to thread 0
	runtime.LockOSThread()
}

//go:embed assets/eject.ico
var embeddedIcon embed.FS

var (
	user32                          = syscall.NewLazyDLL("user32.dll")
	procRegisterDevices             = user32.NewProc("RegisterRawInputDevices")
	procGetRawInputData             = user32.NewProc("GetRawInputData")
	procGetMessage                  = user32.NewProc("GetMessageW")
	procTranslateMessage            = user32.NewProc("TranslateMessage")
	procDispatchMessage             = user32.NewProc("DispatchMessageW")
	procCreateWindowEx              = user32.NewProc("CreateWindowExW")
	procRegisterClassEx             = user32.NewProc("RegisterClassExW")
	procDefWindowProc               = user32.NewProc("DefWindowProcW")
	procSendInput                   = user32.NewProc("SendInput")
	procSendMessage                 = user32.NewProc("SendMessageW")
	procCreatePopupMenu             = user32.NewProc("CreatePopupMenu")
	procAppendMenu                  = user32.NewProc("AppendMenuW")
	procTrackPopupMenu              = user32.NewProc("TrackPopupMenu")
	procSetForegroundWin            = user32.NewProc("SetForegroundWindow")
	procPostQuitMessage             = user32.NewProc("PostQuitMessage")
	procLoadImage                   = user32.NewProc("LoadImageW")
	procGetSystemMetrics            = user32.NewProc("GetSystemMetrics")
	procGetCursorPos                = user32.NewProc("GetCursorPos")
	procLockWorkStation             = user32.NewProc("LockWorkStation")
	procChangeWindowMessageFilterEx = user32.NewProc("ChangeWindowMessageFilterEx")

	powrprof            = syscall.NewLazyDLL("powrprof.dll")
	procSetSuspendState = powrprof.NewProc("SetSuspendState")

	shell32             = syscall.NewLazyDLL("shell32.dll")
	procShellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")

	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procAllocConsole             = kernel32.NewProc("AllocConsole")
	procSetConsoleTitle          = kernel32.NewProc("SetConsoleTitleW")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procTerminateProcess         = kernel32.NewProc("TerminateProcess")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procGetCurrentProcessId      = kernel32.NewProc("GetCurrentProcessId")
	procGetModuleHandle          = kernel32.NewProc("GetModuleHandleW")

	versionDLL                 = syscall.NewLazyDLL("version.dll")
	procGetFileVersionInfoSize = versionDLL.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfo     = versionDLL.NewProc("GetFileVersionInfoW")
	procVerQueryValue          = versionDLL.NewProc("VerQueryValueW")
)

type ActionType int

const (
	ActionCombo ActionType = iota
	ActionLock
	ActionSleep
	ActionHibernate
	ActionMonitorsOff
	ActionResetGFX
	ActionRun
)

type POINT struct{ X, Y int32 }

type RAWINPUTDEVICE struct {
	UsUsagePage uint16
	UsUsage     uint16
	DwFlags     uint32
	HWndTarget  uintptr
}

type RAWINPUTHEADER struct {
	DwType  uint32
	DwSize  uint32
	HDevice uintptr
	WParam  uintptr
}

type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type INPUT struct {
	Type uint32
	Ki   KEYBDINPUT
	_pad [8]byte
}

type WNDCLASSEX struct {
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

type PROCESSENTRY32 struct {
	DwSize              uint32
	CntUsage            uint32
	Th32ProcessID       uint32
	Th32DefaultHeapID   uintptr
	Th32ModuleID        uint32
	CntThreads          uint32
	Th32ParentProcessID uint32
	PcPriClassBase      int32
	DwFlags             uint32
	SzExeFile           [260]uint16
}

type NOTIFYICONDATA struct {
	CbSize            uint32
	HWnd              uintptr
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	HIcon             uintptr
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
}

const (
	RIDEV_INPUTSINK = 0x00000100
	RIDEV_DEVNOTIFY = 0x00002000
	RID_INPUT       = 0x10000003

	WM_INPUT        = 0x00FF
	WM_USER         = 0x0400
	WM_TRAYICON     = WM_USER + 1
	WM_RBUTTONUP    = 0x0205
	WM_LBUTTONUP    = 0x0202
	WM_COMMAND      = 0x0111
	WM_DESTROY      = 0x0002
	WM_SYSCOMMAND   = 0x0112
	SC_MONITORPOWER = 0xF170

	MSGFLT_ALLOW = 1

	HWND_BROADCAST = 0xFFFF

	NIM_ADD     = 0x00000000
	NIM_DELETE  = 0x00000002
	NIF_ICON    = 0x00000002
	NIF_MESSAGE = 0x00000001
	NIF_TIP     = 0x00000004

	MF_STRING       = 0x00000000
	TPM_BOTTOMALIGN = 0x0020
	TPM_RIGHTBUTTON = 0x0002

	IMAGE_ICON      = 1
	LR_LOADFROMFILE = 0x0010
	LR_DEFAULTCOLOR = 0x0000

	SM_CXSMICON = 49
	SM_CYSMICON = 50

	INPUT_KEYBOARD  = 1
	KEYEVENTF_KEYUP = 0x0002

	RIM_TYPEHID = 2

	TH32CS_SNAPPROCESS = 0x00000002
	PROCESS_TERMINATE  = 0x0001

	IDM_QUIT           = 1001
	INJECTED_SIGNATURE = 0xA1243
)

var vkMap = map[string]uint16{
	"CTRL": 0x11, "CONTROL": 0x11, "LCTRL": 0xA2, "RCTRL": 0xA3,
	"ALT": 0x12, "LALT": 0xA4, "RALT": 0xA5,
	"SHIFT": 0x10, "LSHIFT": 0xA0, "RSHIFT": 0xA1,
	"WIN": 0x5B, "WINDOWS": 0x5B, "LWIN": 0x5B, "RWIN": 0x5C,
	"ESC": 0x1B, "ESCAPE": 0x1B, "SPACE": 0x20, "TAB": 0x09, "ENTER": 0x0D,
	"BACKSPACE": 0x08, "DELETE": 0x2E, "DEL": 0x2E, "INSERT": 0x2D,
	"PRINTSCREEN": 0x2C, "PRTSC": 0x2C, "SNAPSHOT": 0x2C,
	"HOME": 0x24, "END": 0x23, "PAGEUP": 0x21, "PAGEDOWN": 0x22,
	"UP": 0x26, "DOWN": 0x28, "LEFT": 0x25, "RIGHT": 0x27,
	"MUTE": 0xAD, "VOLUP": 0xAF, "VOLDOWN": 0xAE,
}

func initVkMap() {
	for i := byte('A'); i <= byte('Z'); i++ {
		vkMap[string(i)] = uint16(i)
	}
	for i := byte('0'); i <= byte('9'); i++ {
		vkMap[string(i)] = uint16(i)
	}
	for i := 1; i <= 24; i++ {
		vkMap[fmt.Sprintf("F%d", i)] = uint16(0x70 + i - 1)
	}
}

var (
	globalDebug      bool
	globalAction     ActionType
	globalVkSequence []uint16
	globalExecCmd    string
	globalHIcon      uintptr
	nid              NOTIFYICONDATA
	isProcessing     bool
	targetActionStr  string
)

func killPreviousInstances() {
	currentPID, _, _ := procGetCurrentProcessId.Call()
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	exeName := filepath.Base(exePath)

	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == uintptr(syscall.InvalidHandle) {
		return
	}
	defer procCloseHandle.Call(snapshot)

	var entry PROCESSENTRY32
	entry.DwSize = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for ret != 0 {
		if entry.Th32ProcessID != uint32(currentPID) {
			name := syscall.UTF16ToString(entry.SzExeFile[:])
			if strings.EqualFold(name, exeName) {
				hProcess, _, _ := procOpenProcess.Call(PROCESS_TERMINATE, 0, uintptr(entry.Th32ProcessID))
				if hProcess != 0 {
					procTerminateProcess.Call(hProcess, 0)
					procCloseHandle.Call(hProcess)
				}
			}
		}
		ret, _, _ = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
}

func parseKeyCombo(input string) ([]uint16, bool) {
	parts := strings.Split(input, "+")
	var sequence []uint16

	for _, p := range parts {
		token := strings.ToUpper(strings.TrimSpace(p))
		if strings.HasPrefix(token, "0X") {
			parsed, err := strconv.ParseUint(token[2:], 16, 16)
			if err == nil && parsed >= 0x01 && parsed <= 0xFE {
				sequence = append(sequence, uint16(parsed))
				continue
			}
		}
		if vk, exists := vkMap[token]; exists {
			sequence = append(sequence, vk)
		} else {
			return nil, false
		}
	}
	return sequence, len(sequence) > 0
}

func triggerAction() {
	if isProcessing {
		return
	}
	isProcessing = true
	defer func() { isProcessing = false }()

	switch globalAction {
	case ActionCombo:
		n := len(globalVkSequence)
		inputs := make([]INPUT, n*2)

		for i, vk := range globalVkSequence {
			inputs[i].Type = INPUT_KEYBOARD
			inputs[i].Ki.WVk = vk
			inputs[i].Ki.DwExtraInfo = INJECTED_SIGNATURE
		}

		for i, vk := range globalVkSequence {
			idx := n + (n - 1 - i)
			inputs[idx].Type = INPUT_KEYBOARD
			inputs[idx].Ki.WVk = vk
			inputs[idx].Ki.DwFlags = KEYEVENTF_KEYUP
			inputs[idx].Ki.DwExtraInfo = INJECTED_SIGNATURE
		}

		procSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), uintptr(unsafe.Sizeof(inputs[0])))

	case ActionLock:
		procLockWorkStation.Call()

	case ActionSleep:
		procSetSuspendState.Call(0, 0, 0)

	case ActionHibernate:
		procSetSuspendState.Call(1, 0, 0)

	case ActionMonitorsOff:
		procSendMessage.Call(HWND_BROADCAST, WM_SYSCOMMAND, SC_MONITORPOWER, 2)

	case ActionResetGFX:
		seq := []uint16{0x5B, 0x11, 0x10, 0x42}
		n := len(seq)
		inputs := make([]INPUT, n*2)
		for i, vk := range seq {
			inputs[i].Type = INPUT_KEYBOARD
			inputs[i].Ki.WVk = vk
			inputs[i].Ki.DwExtraInfo = INJECTED_SIGNATURE
		}
		for i, vk := range seq {
			idx := n + (n - 1 - i)
			inputs[idx].Type = INPUT_KEYBOARD
			inputs[idx].Ki.WVk = vk
			inputs[idx].Ki.DwFlags = KEYEVENTF_KEYUP
			inputs[idx].Ki.DwExtraInfo = INJECTED_SIGNATURE
		}
		procSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), uintptr(unsafe.Sizeof(inputs[0])))

	case ActionRun:
		go func() {
			args := strings.Fields(globalExecCmd)
			if len(args) == 0 {
				return
			}
			cmd := exec.Command(args[0], args[1:]...)
			_ = cmd.Start()
		}()
	}
}

func loadIconAsset() uintptr {
	exePath, err := os.Executable()
	if err != nil {
		return 0
	}

	iconPath := filepath.Join(filepath.Dir(exePath), "assets", "eject.ico")

	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		iconData, err := embeddedIcon.ReadFile("assets/eject.ico")
		if err == nil {
			_ = os.MkdirAll(filepath.Dir(iconPath), 0755)
			_ = os.WriteFile(iconPath, iconData, 0644)
		}
	}

	cx, _, _ := procGetSystemMetrics.Call(SM_CXSMICON)
	cy, _, _ := procGetSystemMetrics.Call(SM_CYSMICON)

	if cx < 24 {
		cx = 24
		cy = 24
	}

	pathPtr, _ := syscall.UTF16PtrFromString(iconPath)
	hIcon, _, _ := procLoadImage.Call(
		0,
		uintptr(unsafe.Pointer(pathPtr)),
		IMAGE_ICON,
		cx, cy,
		LR_LOADFROMFILE|LR_DEFAULTCOLOR,
	)
	return hIcon
}

func setupTrayIcon(hwnd uintptr) {
	globalHIcon = loadIconAsset()

	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwnd
	nid.UID = 1
	nid.UFlags = NIF_ICON | NIF_MESSAGE | NIF_TIP
	nid.UCallbackMessage = WM_TRAYICON
	nid.HIcon = globalHIcon

	tipText := fmt.Sprintf("Apple A1243 Eject Map (%s)", targetActionStr)
	if len(tipText) > 127 {
		tipText = tipText[:127]
	}
	copy(nid.SzTip[:], syscall.StringToUTF16(tipText))

	procShellNotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
}

func removeTrayIcon() {
	procShellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}

func showContextMenu(hwnd uintptr) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	hMenu, _, _ := procCreatePopupMenu.Call()
	quitLabel, _ := syscall.UTF16PtrFromString("Quit AppleA1243EjectMap")
	procAppendMenu.Call(hMenu, MF_STRING, IDM_QUIT, uintptr(unsafe.Pointer(quitLabel)))

	procSetForegroundWin.Call(hwnd)
	procTrackPopupMenu.Call(hMenu, TPM_BOTTOMALIGN|TPM_RIGHTBUTTON, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
}

func spawnDedicatedConsole() {
	res, _, _ := procAllocConsole.Call()
	if res == 0 {
		return
	}

	title, _ := syscall.UTF16PtrFromString("A1243 Eject Map Console")
	procSetConsoleTitle.Call(uintptr(unsafe.Pointer(title)))

	fOut, errOut := os.OpenFile("CONOUT$", os.O_WRONLY, 0666)
	if errOut == nil {
		os.Stdout = fOut
		os.Stderr = fOut
		syscall.Stdout = syscall.Handle(fOut.Fd())
		syscall.Stderr = syscall.Handle(fOut.Fd())
	}

	fIn, errIn := os.OpenFile("CONIN$", os.O_RDONLY, 0666)
	if errIn == nil {
		os.Stdin = fIn
		syscall.Stdin = syscall.Handle(fIn.Fd())
	}
}

func printUsage() {
	version, copyright := runningExeVersionInfo()
	fmt.Println("==================================================")
	fmt.Printf("   Apple A1243 Eject Key Remapper v%s\n", version)
	fmt.Printf("   %s\n", copyright)
	fmt.Println("==================================================")
	fmt.Println("[ERROR] Invalid shortcut sequence or command.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  AppleA1243EjectMap.exe <ACTION|SHORTCUT> [FLAGS]")
	fmt.Println()
	fmt.Println("Built-in System Actions:")
	fmt.Println("  lock          - Lock Workstation")
	fmt.Println("  sleep         - Put PC to sleep")
	fmt.Println("  hibernate     - Put PC into hibernation")
	fmt.Println("  monitor-off   - Turn off displays instantly")
	fmt.Println("  reset-gfx     - Reset graphics driver (Win+Ctrl+Shift+B)")
	fmt.Println("  taskmgr       - Launch Task Manager")
	fmt.Println("  clipboard     - Toggle Clipboard History (Win+V)")
	fmt.Println("  gamebar       - Toggle Game Bar (Win+G)")
	fmt.Println("--------------------------------------------------")
}

func runningExeVersionInfo() (string, string) {
	exePath, err := os.Executable()
	if err != nil {
		return "error", ""
	}
	fileName, err := syscall.UTF16FromString(exePath)
	if err != nil {
		return "error", ""
	}

	size, _, _ := procGetFileVersionInfoSize.Call(uintptr(unsafe.Pointer(&fileName[0])), 0)
	if size == 0 || size > uintptr(^uint32(0)) {
		return "error", ""
	}
	data := make([]byte, size)
	if result, _, _ := procGetFileVersionInfo.Call(
		uintptr(unsafe.Pointer(&fileName[0])),
		0,
		size,
		uintptr(unsafe.Pointer(&data[0])),
	); result == 0 {
		return "error", ""
	}

	version := queryVersionString(data, `\StringFileInfo\040904b0\FileVersion`)
	copyright := queryVersionString(data, `\StringFileInfo\040904b0\LegalCopyright`)
	return version, copyright
}

func queryVersionString(data []byte, subBlock string) string {
	path, err := syscall.UTF16FromString(subBlock)
	if err != nil {
		return ""
	}

	var value uintptr
	var length uint32
	result, _, _ := procVerQueryValue.Call(
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(unsafe.Pointer(&path[0])),
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Pointer(&length)),
	)
	if result == 0 || value == 0 || length == 0 {
		return ""
	}

	return string(utf16.Decode(unsafe.Slice((*uint16)(unsafe.Pointer(value)), length-1)))
}

func pauseAndExit(code int) {
	if globalDebug {
		fmt.Println("\n--------------------------------------------------")
		fmt.Println("Press ENTER to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
	os.Exit(code)
}

func wndProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	switch msg {
	case WM_INPUT:
		handleRawInput(lParam)
		return 0
	case WM_TRAYICON:
		if lParam == WM_RBUTTONUP || lParam == WM_LBUTTONUP {
			showContextMenu(hwnd)
		}
		return 0
	case WM_COMMAND:
		if wParam&0xFFFF == IDM_QUIT {
			removeTrayIcon()
			procPostQuitMessage.Call(0)
		}
		return 0
	case WM_DESTROY:
		removeTrayIcon()
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func main() {
	runtime.LockOSThread()

	initVkMap()

	// Parse flags for -debug before any standard output or command handling
	for _, arg := range os.Args[1:] {
		lower := strings.ToLower(arg)
		if lower == "-debug" || lower == "--debug" {
			globalDebug = true
			spawnDedicatedConsole()
			break
		}
	}
	killPreviousInstances()

	hasAction := false

	for _, arg := range os.Args[1:] {
		lowerArg := strings.ToLower(arg)

		if lowerArg == "-debug" || lowerArg == "--debug" {
			continue
		}

		switch lowerArg {
		case "lock":
			globalAction = ActionLock
			targetActionStr = "Lock Workstation"
			hasAction = true
		case "sleep":
			globalAction = ActionSleep
			targetActionStr = "System Sleep"
			hasAction = true
		case "hibernate":
			globalAction = ActionHibernate
			targetActionStr = "System Hibernate"
			hasAction = true
		case "monitors-off":
			globalAction = ActionMonitorsOff
			targetActionStr = "Monitors Off"
			hasAction = true
		case "reset-gfx":
			globalAction = ActionResetGFX
			targetActionStr = "Reset Graphics Driver"
			hasAction = true
		case "taskmgr":
			seq, _ := parseKeyCombo("Ctrl+Shift+Esc")
			globalAction = ActionCombo
			globalVkSequence = seq
			targetActionStr = "Task Manager"
			hasAction = true
		case "clipboard":
			seq, _ := parseKeyCombo("Win+V")
			globalAction = ActionCombo
			globalVkSequence = seq
			targetActionStr = "Clipboard History"
			hasAction = true
		case "gamebar":
			seq, _ := parseKeyCombo("Win+G")
			globalAction = ActionCombo
			globalVkSequence = seq
			targetActionStr = "Xbox Game Bar"
			hasAction = true
		default:
			if strings.HasPrefix(lowerArg, "run:") {
				globalAction = ActionRun
				globalExecCmd = arg[4:]
				targetActionStr = "Run: " + globalExecCmd
				hasAction = true
			} else if seq, ok := parseKeyCombo(arg); ok {
				globalAction = ActionCombo
				globalVkSequence = seq
				targetActionStr = strings.ToUpper(arg)
				hasAction = true
			}
		}
	}

	if !hasAction {
		if globalDebug {
			printUsage()
		}
		pauseAndExit(1)
	}

	if globalDebug {
		printUsage()
		fmt.Printf("Target Action: %s\n", targetActionStr)
		fmt.Println("Registering Kernel Raw Input Hook...")
	}

	hInstance, _, _ := procGetModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("A1243EjectClass")

	wc := WNDCLASSEX{
		Size:      uint32(unsafe.Sizeof(WNDCLASSEX{})),
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  hInstance,
		ClassName: className,
	}

	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0, 0, 0, 0, 0, 0,
		0,
		0, hInstance, 0,
	)

	procChangeWindowMessageFilterEx.Call(hwnd, WM_INPUT, MSGFLT_ALLOW, 0)

	devices := []RAWINPUTDEVICE{
		{UsUsagePage: 0x01, UsUsage: 0x06, DwFlags: RIDEV_INPUTSINK | RIDEV_DEVNOTIFY, HWndTarget: hwnd},
		{UsUsagePage: 0x0C, UsUsage: 0x01, DwFlags: RIDEV_INPUTSINK | RIDEV_DEVNOTIFY, HWndTarget: hwnd},
		{UsUsagePage: 0xFF01, UsUsage: 0x01, DwFlags: RIDEV_INPUTSINK | RIDEV_DEVNOTIFY, HWndTarget: hwnd},
	}

	res, _, _ := procRegisterDevices.Call(
		uintptr(unsafe.Pointer(&devices[0])),
		uintptr(len(devices)),
		unsafe.Sizeof(devices[0]),
	)

	if globalDebug {
		if res == 0 {
			fmt.Println("[ERROR] Failed to register raw input devices!")
		} else {
			fmt.Println("[SUCCESS] Kernel Raw Input Hook registered globally!")
		}
		fmt.Println("System Tray Icon created. Ready to trigger action.")
		fmt.Println("--------------------------------------------------")
	}

	setupTrayIcon(hwnd)

	var msg struct {
		HWnd    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      struct{ X, Y int32 }
	}

	for {
		res, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if res == 0 || res == ^uintptr(0) {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}

	removeTrayIcon()
	pauseAndExit(0)
}

func handleRawInput(hRawInput uintptr) {
	var dwSize uint32
	procGetRawInputData.Call(hRawInput, RID_INPUT, 0, uintptr(unsafe.Pointer(&dwSize)), unsafe.Sizeof(RAWINPUTHEADER{}))
	if dwSize == 0 {
		return
	}

	buffer := make([]byte, dwSize)
	ret, _, _ := procGetRawInputData.Call(
		hRawInput,
		RID_INPUT,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&dwSize)),
		unsafe.Sizeof(RAWINPUTHEADER{}),
	)

	if ret == ^uintptr(0) || ret == 0 {
		return
	}

	dwType := *(*uint32)(unsafe.Pointer(&buffer[0]))

	if globalAction == ActionCombo && dwSize == 40 {
		injectedVK := uint16(buffer[30])
		for _, vk := range globalVkSequence {
			if injectedVK == vk {
				return
			}
		}
	}

	if globalDebug {
		fmt.Printf("[RAW PACKET (%d bytes, type %d)]: ", len(buffer), dwType)
		for _, b := range buffer {
			fmt.Printf("%02X ", b)
		}
		fmt.Println()
	}

	if dwType == RIM_TYPEHID {
		for _, b := range buffer {
			if b == 0xB8 || b == 0x08 {
				if globalDebug {
					fmt.Printf("  └─>>> [EJECT DETECTED] Executing: %s\n", targetActionStr)
				}
				go triggerAction()
				break
			}
		}
	}
}
