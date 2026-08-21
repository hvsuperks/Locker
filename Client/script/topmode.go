package script

import (
	"context"
	"syscall"
	"time"
	"unsafe"
)

var GWL_STYLE int32 = -16
var GWL_EXSTYLE int32 = -20

const (
	// Các chỉ số hồ sơ

	// Các bit style mở rộng (-20)
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_APPWINDOW  = 0x00040000
	WS_EX_TOPMOST    = 0x00000008

	// Các bit style cơ bản (-16)
	WS_CAPTION = 0x00C00000 // Thanh tiêu đề và nút X

	// Vị trí
	HWND_TOPMOST     = ^uintptr(0) // -1
	SWP_NOSIZE       = 0x0001
	SWP_FRAMECHANGED = 0x0020

	SM_CXSCREEN = 0 // Chỉ số lấy chiều rộng màn hình chính
)

func SetTopmost(ctx context.Context, hwnd uintptr, width float32) {
	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()
	var ch = make(chan struct{}, 10)
	ch <- struct{}{}
	user32 := syscall.NewLazyDLL("user32.dll")
	setWindowPos := user32.NewProc("SetWindowPos")
	setWindowLong := user32.NewProc("SetWindowLongW")
	getSystemMetrics := user32.NewProc("GetSystemMetrics")
	// Các hằng số Windows API
	const (
		HWND_TOPMOST   = ^uintptr(0) // -1: Luôn nằm trên cùng
		SWP_NOSIZE     = 0x0001      // Không đổi kích thước
		SWP_NOMOVE     = 0x0002      // Không đổi vị trí
		WS_POPUP       = 0x80000000
		WS_VISIBLE     = 0x10000000
		SWP_SHOWWINDOW = 0x0040
	)
	// Chỉ giữ lại thuộc tính POPUP và VISIBLE, xóa sạch Minimize/Maximize/Close/TitleBar
	setWindowLong.Call(hwnd, uintptr(GWL_STYLE), uintptr(WS_POPUP|WS_VISIBLE))
	screenWidth, _, _ := getSystemMetrics.Call(0)
	x := (float32(screenWidth) - width) / 2
	// Gọi hàm để ghim cửa sổ lên trên cùng
	setWindowPos.Call(
		hwnd,
		HWND_TOPMOST,
		uintptr(x),
		0,
		1400, 30,
		SWP_SHOWWINDOW,
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		case <-ch:
		}

		setWindowPos.Call(
			hwnd,
			HWND_TOPMOST,
			uintptr(x),
			0,
			1400, 30,
			SWP_SHOWWINDOW,
		)
	}
}

func HideFromTaskbar(hwnd uintptr) {
	user32 := syscall.NewLazyDLL("user32.dll")
	setWindowLong := user32.NewProc("SetWindowLongW")
	getWindowLong := user32.NewProc("GetWindowLongW")

	style, _, _ := getWindowLong.Call(hwnd, uintptr(GWL_EXSTYLE))
	// Thêm ToolWindow (ẩn) và loại bỏ AppWindow (hiện)
	newStyle := (style | WS_EX_TOOLWINDOW) &^ WS_EX_APPWINDOW
	setWindowLong.Call(hwnd, uintptr(GWL_EXSTYLE), newStyle)
}

func GetHwndByTitle(title string) uintptr {
	user32 := syscall.NewLazyDLL("user32.dll")
	procFindWindow := user32.NewProc("FindWindowW")

	// Chuyển string của Go sang UTF16 cho Windows hiểu
	tPtr, _ := syscall.UTF16PtrFromString(title)

	// Tham số đầu tiên là ClassName (để nil/0), tham số thứ hai là Title
	hwnd, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(tPtr)))

	return hwnd
}
