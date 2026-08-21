package main

import (
	"Locker/config"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func API_uploadHandler(w http.ResponseWriter, r *http.Request) {

	// 1. Giới hạn dung lượng file upload (ví dụ: 10MB)
	r.ParseMultipartForm(500 << 20)
	folderName := r.FormValue("folder")

	// 2. Lấy file từ form-data
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		http.Error(w, "Lỗi lấy file", http.StatusBadRequest)
		Fmt(fmtClient, "Lỗi lấy file ", folderName, err)
		return
	}
	defer file.Close()
	dstPath := filepath.Join(config.HttpUpload, folderName, handler.Filename)
	Fmt(fmtClient, dstPath)
	if strings.HasPrefix(handler.Filename, "log_") {
		dstPath = filepath.Join(config.LogPath_master, folderName, handler.Filename)
	}
	os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Lỗi tạo file trên server", http.StatusInternalServerError)
		Fmt(fmtClient, "Lỗi tạo file trên server ", folderName, err)
		return
	}
	defer dst.Close()

	// 4. Copy dữ liệu từ request vào file (Streaming - không tốn RAM)
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		Fmt(fmtClient, "Lỗi Copy ", folderName, err)
		return
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(config.WebPathDir, "ApiWeb", "webPGMZipUpload.html"))
}
