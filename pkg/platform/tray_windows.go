//go:build windows

package platform

import (
	"errors"
	"syscall"
	"unsafe"
)

// 本文件提供系统托盘（Change 003 Phase 3）所需的 Win32 底层能力封装，
// 全部为无业务依赖的原语：隐藏宿主窗口、Shell_NotifyIcon 增删、消息泵、
// 图标加载、右键弹出菜单。
// 生命周期/回调编排见 pkg/tray（tray.go / tray_windows.go）。

var (
	shell32 = syscall.NewLazyDLL("shell32.dll")

	procRegisterWindowW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW   = user32.NewProc("CreateWindowExW")
	procDestroyWindow     = user32.NewProc("DestroyWindow")
	procDefWindowProcW    = user32.NewProc("DefWindowProcW")
	procGetClassLongPtrW  = user32.NewProc("GetClassLongPtrW")
	procDispatchMessageW  = user32.NewProc("DispatchMessageW")
	procTranslateMessage  = user32.NewProc("TranslateMessage")
	procPostMessageW      = user32.NewProc("PostMessageW")
	procPostQuitMessageW  = user32.NewProc("PostQuitMessage")
	procShellNotifyIconW  = shell32.NewProc("Shell_NotifyIconW")
	procCreatePopupMenu   = user32.NewProc("CreatePopupMenu")
	procAppendMenuW       = user32.NewProc("AppendMenuW")
	procTrackPopupMenu    = user32.NewProc("TrackPopupMenu")
	procDestroyMenu       = user32.NewProc("DestroyMenu")
	procGetCursorPos      = user32.NewProc("GetCursorPos")
	procLoadIconW         = user32.NewProc("LoadIconW")
	procGetWindowLongPtrW = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
)

// 托盘相关常量
const (
	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	WM_USER = 0x0400

	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800

	TPM_RETURNCMD   = 0x00000100
	TPM_RIGHTBUTTON = 0x00000002
	TPM_ALIGNLEFT   = 0x00000000

	GCLP_HICON = -14

	// 宿主隐藏窗口消息：菜单回调
	WM_COMMAND = 0x0111

	IDI_APPLICATION = 32512 // 系统默认应用程序图标

	// GWLP_USERDATA：把托盘控制器实例关联到宿主窗口，供窗口过程读取
	GWLP_USERDATA = -21
)

// TrayMenuItem 托盘右键菜单项。
type TrayMenuItem struct {
	ID        uint32 // 命令 ID（菜单项触发时通过 f命令回调返回）
	Text      string
	Separator bool
}

// WNDCLASSEXW 窗口类扩展结构（用于注册隐藏宿主窗口）。
type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

// NOTIFYICONDATAW 托盘数据（仅用 Shell_NotifyIcon 所需字段，按内存对齐布局）。
type NOTIFYICONDATAW struct {
	CbSize           uint32
	Hwnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
}

// RegisterTrayWindowClass 注册隐藏宿主窗口类（若已注册则静默放行）。
// 注意：该类名在进程内唯一，重复 Start/Stop 不重复报错。
func RegisterTrayWindowClass(className string, wndProc uintptr) error {
	if wndProc == 0 {
		return errors.New("托盘窗口过程不能为空")
	}
	name, _ := syscall.UTF16PtrFromString(className)
	hInst := GetModuleHandle("")
	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         0,
		LpfnWndProc:   wndProc,
		HInstance:     hInst,
		HIcon:         0,
		HCursor:       0,
		HbrBackground: 0,
		LpszClassName: name,
	}
	ret, _, _ := procRegisterWindowW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		// ERROR_CLASS_ALREADY_EXISTS(1410)：同类已注册属于预期（幂等），放行
		if e := syscall.GetLastError(); e != syscall.Errno(1410) {
			return e
		}
	}
	return nil
}

// CreateTrayHostWindow 创建托盘隐藏宿主窗口（HWND_MESSAGE，不进入任务栏）。
// 窗口过程 wndProc 需由 pkg/tray 提供（处理回调消息 + 菜单命令）。
func CreateTrayHostWindow(className string) (uintptr, error) {
	cls, _ := syscall.UTF16PtrFromString(className)
	title, _ := syscall.UTF16PtrFromString("")
	// HWND_MESSAGE：仅消息窗口，不显示、无任务栏项
	hwndMessageParent := ^uintptr(2) // (HWND)-3
	ret, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(title)),
		0x80000000, // WS_POPUP
		uintptr(0),
		uintptr(0),
		uintptr(0),
		uintptr(0),
		hwndMessageParent,
		0,
		0,
		0,
	)
	if ret == 0 {
		return 0, syscall.GetLastError()
	}
	return ret, nil
}

// DestroyTrayHostWindow 销毁托盘宿主窗口。
func DestroyTrayHostWindow(hwnd uintptr) {
	if hwnd != 0 {
		procDestroyWindow.Call(hwnd) //nolint:errcheck // 销毁为尽力而为
	}
}

// GetWindowLongPtr 读取窗口长指针值（配合 GWLP_USERDATA 关联控制器）。
func GetWindowLongPtr(hwnd, index uintptr) uintptr {
	ret, _, _ := procGetWindowLongPtrW.Call(hwnd, index)
	return ret
}

// SetWindowLongPtr 写入窗口长指针值（把托盘控制器实例关联到宿主窗口）。
func SetWindowLongPtr(hwnd, index, value uintptr) uintptr {
	ret, _, _ := procSetWindowLongPtrW.Call(hwnd, index, value)
	return ret
}

// GetClassIcon 读取窗口类图标句柄（复用主窗口，HICON）。
func GetClassIcon(hwnd uintptr) uintptr {
	if hwnd == 0 {
		return 0
	}
	// GCLP_HICON 为负索引（-14），需经有符号 int 传递避免 uintptr 常量溢出
	idx := int(GCLP_HICON)
	ret, _, _ := procGetClassLongPtrW.Call(hwnd, uintptr(idx))
	return ret
}

// LoadAppIcon 获取应用图标句柄（HICON）。
// 优先复用主窗口类图标（与应用一致的既有图标，零额外资源）；
// 主窗口不可用时降级为系统默认应用程序图标，保证功能正常。
func LoadAppIcon() uintptr {
	if hwnd, err := GetWindowHandle(); err == nil && hwnd != 0 {
		if icon := GetClassIcon(uintptr(hwnd)); icon != 0 {
			return icon
		}
	}
	ret, _, _ := procLoadIconW.Call(0, IDI_APPLICATION)
	return ret
}

// AddTrayIcon 添加托盘图标（Shell_NotifyIcon NIM_ADD）。
// hwnd：宿主窗口；icon：HICON；tip：悬停提示；uid 与回调消息。
func AddTrayIcon(hwnd uintptr, icon uintptr, tip string, uid uint32, callbackMsg uint32) error {
	return notifyIconRaw(NIM_ADD, hwnd, icon, tip, uid, callbackMsg)
}

// ModifyTrayIcon 更新托盘图标（NIM_MODIFY；当前主要用于保留期 API 对称）。
func ModifyTrayIcon(hwnd uintptr, icon uintptr, tip string, uid uint32, callbackMsg uint32) error {
	return notifyIconRaw(NIM_MODIFY, hwnd, icon, tip, uid, callbackMsg)
}

// DeleteTrayIcon 删除托盘图标（NIM_DELETE）。
func DeleteTrayIcon(hwnd uintptr, uid uint32) error {
	return notifyIconRaw(NIM_DELETE, hwnd, 0, "", uid, 0)
}

// notifyIconRaw 统一下发 Shell_NotifyIconW 调用。
func notifyIconRaw(nim uint32, hwnd uintptr, icon uintptr, tip string, uid uint32, callbackMsg uint32) error {
	if hwnd == 0 {
		return errors.New("托盘宿主窗口句柄为空")
	}
	data := NOTIFYICONDATAW{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATAW{})),
		Hwnd:             hwnd,
		UID:              uid,
		UFlags:           NIF_MESSAGE | NIF_ICON,
		UCallbackMessage: callbackMsg,
		HIcon:            icon,
	}
	if nim != NIM_DELETE && icon == 0 {
		return errors.New("托盘图标句柄为空")
	}
	if nim == NIM_DELETE {
		data.UFlags = 0
	}
	if tip != "" {
		data.UFlags |= NIF_TIP
		buf := make([]uint16, 128)
		t, _ := syscall.UTF16FromString(tip)
		copy(buf[:], t)
		for i := range buf[:len(buf)] {
			data.SzTip[i] = buf[i]
		}
	}
	ret, _, err := procShellNotifyIconW.Call(uintptr(nim), uintptr(unsafe.Pointer(&data)))
	if ret == 0 {
		return err
	}
	return nil
}

// TranslateMessage 转换键盘消息（消息泵一部分）。
func TranslateMessage(msg *MSG) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
}

// DispatchMessage 分发消息（调用窗口过程）。
func DispatchMessage(msg *MSG) {
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
}

// PostMessage 向窗口投递消息（用于唤醒消息泵停止）。
func PostMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) bool {
	ret, _, _ := procPostMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret != 0
}

// PostQuitMessage 将 WM_QUIT 放入当前线程消息队列，使 GetMessage 返回 0 以退出消息循环。
func PostQuitMessage(exitCode int32) {
	procPostQuitMessageW.Call(uintptr(exitCode))
}

// DefWindowProc 默认窗口过程处理。
func DefWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

// ShowTrayMenu 弹出右键菜单并返回选中的菜单项 ID（TPM_RETURNCMD）。
// 调用发生在宿主窗口所在的消息线程内，封装了菜单创建/展现/销毁全周期。
func ShowTrayMenu(hwnd uintptr, items []TrayMenuItem) (uint32, error) {
	if len(items) == 0 {
		return 0, nil
	}
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return 0, errors.New("创建弹出菜单失败")
	}
	defer procDestroyMenu.Call(menu)

	for _, item := range items {
		if item.Separator {
			procAppendMenuW.Call(menu, MF_SEPARATOR, 0, 0)
			continue
		}
		text, _ := syscall.UTF16PtrFromString(item.Text)
		ret, _, _ := procAppendMenuW.Call(menu, MF_STRING, uintptr(item.ID), uintptr(unsafe.Pointer(text)))
		if ret == 0 {
			return 0, errors.New("添加菜单项失败")
		}
	}

	// TPM_RETURNCMD：函数返回被选中的菜单项 ID；0 表示取消
	ret, _, _ := procTrackPopupMenu.Call(
		menu,
		TPM_RETURNCMD|TPM_RIGHTBUTTON|TPM_ALIGNLEFT,
		uintptr(pt.X),
		uintptr(pt.Y),
		hwnd,
		0,
	)
	return uint32(ret), nil
}
