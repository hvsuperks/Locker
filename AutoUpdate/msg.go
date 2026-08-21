package main

import (
	"syscall"
	"unsafe"
)

var (
	modwtsapi32         = syscall.NewLazyDLL("wtsapi32.dll")
	procWTSSendMessageW = modwtsapi32.NewProc("WTSSendMessageW")
)

const (
	WTS_CURRENT_SERVER_HANDLE = 0
	WTS_CURRENT_SESSION       = 0xFFFFFFFF // Hoặc lấy ID Session active
	MB_OK                     = 0x00000000
	MB_ICONERROR              = 0x00000010
)

// ShowErrorPopup hiện popup lỗi trực tiếp lên Desktop của user từ Service
func ShowErrorPopup(title, message string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(message)

	var response uint32

	// WTSSendMessageW gửi Message Box hiển thị ở Session của user đang active
	procWTSSendMessageW.Call(
		uintptr(WTS_CURRENT_SERVER_HANDLE),
		uintptr(WTS_CURRENT_SESSION),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(len(title)*2),
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(len(message)*2),
		uintptr(MB_OK|MB_ICONERROR),
		uintptr(5), // Timeout (0 = chờ user bấm OK)
		uintptr(unsafe.Pointer(&response)),
		uintptr(1), // Wait for response
	)
}
