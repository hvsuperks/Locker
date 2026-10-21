package main

import (
	"Locker/config"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func POST_uploadFile(w http.ResponseWriter, r *http.Request) {

	// 1. Giới hạn dung lượng file upload (ví dụ: 10MB)
	r.ParseMultipartForm(500 << 20)
	folderName := r.FormValue("folder")
	mode := r.FormValue(`mode`)
	// 2. Lấy file từ form-data
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		http.Error(w, "Lỗi lấy file", http.StatusBadRequest)
		LogInfo(&Logger.API, "POST_uploadFile", "Lỗi lấy file ", folderName, err)
		return
	}
	defer file.Close()
	dstPath := filepath.Join(config.HttpUpload, folderName, handler.Filename)
	if mode == `free` {
		dstPath = filepath.Join(config.HttpDir, folderName, handler.Filename)
	} else {
		if pa := strings.Split(handler.Filename, "_"); len(pa) > 3 {
			if len(pa[0]) == 9 && len(pa[1]) == 8 && len(pa[2]) == 6 {
				dstPath = filepath.Join(config.HttpUpload, folderName, fmt.Sprintf("%s_%s_%s_%s", pa[1], pa[2], pa[0], strings.Join(pa[3:], "_")))
			}
		}
	}

	os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
	if i, err := os.Stat(dstPath); err == nil {
		if i.IsDir() {
			os.RemoveAll(dstPath)
		}
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Lỗi tạo file trên server "+dstPath+": "+err.Error(), http.StatusInternalServerError)
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
