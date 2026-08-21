package main

import (
	"Locker/config"
	"Locker/script"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func updateHandler(w http.ResponseWriter, r *http.Request) {

	// 1. Giới hạn dung lượng file upload (ví dụ: 10MB)
	r.ParseMultipartForm(500 << 20)
	// 2. Lấy file từ form-data
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		http.Error(w, "Lỗi lấy file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	dstPath := ""
	isMaster := false
	// 3. Tạo đường dẫn lưu file (Tránh ghi đè hoặc thoát khỏi thư mục chỉ định)
	if strings.Contains(strings.ToLower(handler.Filename), "master") {
		dstPath = filepath.Join("D:\\", handler.Filename)
		isMaster = true
	} else if strings.HasSuffix(handler.Filename, "html") {
		dstPath = filepath.Join(config.WebPathDir, "ApiWeb", handler.Filename)
	} else {
		dstPath = filepath.Join(config.HttpDir, handler.Filename)
	}
	dst, err := os.Create(dstPath + ".tmp")
	if err != nil {
		http.Error(w, "Lỗi tạo file trên server", http.StatusInternalServerError)
		return
	}

	// 4. Copy dữ liệu từ request vào file (Streaming - không tốn RAM)
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		dst.Close()
		return
	}
	dst.Close()
	os.Remove(dstPath)
	err = os.Rename(dstPath+".tmp", dstPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isMaster {
		script.LaunchAdminDetached(dstPath)
	}
}
