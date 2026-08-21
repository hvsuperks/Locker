package script

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

var run = true
var start = false

func Cmd(mode bool, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	if mode == run {
		return cmd.Run()
	} else {
		return cmd.Start()
	}
}

func EqualValue(a, b string) bool {

	// thử parse thành số
	f1, err1 := strconv.ParseFloat(a, 64)
	f2, err2 := strconv.ParseFloat(b, 64)

	if err1 == nil && err2 == nil {
		return f1 == f2
	}

	// nếu không phải số → so sánh string
	return strings.EqualFold(a, b)
}
