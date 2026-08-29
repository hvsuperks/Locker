package main

import (
	"Locker/config"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func POST_uploadFile(w http.ResponseWriter, r *http.Request) {

	// 1. Giới hạn dung lượng file upload (ví dụ: 10MB)
	r.ParseMultipartForm(500 << 20)
	folderName := r.FormValue("folder")

	// 2. Lấy file từ form-data
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		http.Error(w, "Lỗi lấy file", http.StatusBadRequest)
		LogInfo(&Logger.API, "POST_uploadFile", "Lỗi lấy file ", folderName, err)
		return
	}
	defer file.Close()
	dstPath := filepath.Join(config.HttpUpload, folderName, handler.Filename)
	os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Lỗi tạo file trên server", http.StatusInternalServerError)
		LogInfo(&Logger.API, "POST_uploadFile", "Lỗi tạo file trên server ", folderName, err)
		return
	}
	defer dst.Close()

	// 4. Copy dữ liệu từ request vào file (Streaming - không tốn RAM)
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		LogInfo(&Logger.API, "POST_uploadFile", "Lỗi Copy ", folderName, err)
		return
	}
}
