package main

import (
	"syscall"
	"unsafe"
)

func msgbox(text string) {

	user32 := syscall.NewLazyDLL("user32.dll")
	msgBox := user32.NewProc("MessageBoxW")

	title, _ := syscall.UTF16PtrFromString("Locker Fail")
	message, _ := syscall.UTF16PtrFromString(text)

	msgBox.Call(
		0,
		uintptr(unsafe.Pointer(message)),
		uintptr(unsafe.Pointer(title)),
		0,
	)

}
