package script

import (
	"path/filepath"
	"strings"
)

func ComparePath(scr, dst string) bool {
	// 1. Đưa về đường dẫn tuyệt đối và xử lý Shortcut/Symlink
	path1, err1 := filepath.Abs(scr)
	path2, err2 := filepath.Abs(dst)
	if err1 != nil || err2 != nil {
		return false
	}

	// 2. Làm sạch đường dẫn (xóa các dấu . hoặc .. thừa)
	path1 = filepath.Clean(path1)
	path2 = filepath.Clean(path2)

	// 3. Với Windows: Phải chuyển về cùng chữ thường vì Windows không phân biệt hoa/thường
	// "C:\Users" và "c:\users" là một
	return strings.EqualFold(path1, path2)
}
