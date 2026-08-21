package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var aoiChan = make(chan struct{}, 25)

func uploadAOIHandler(w http.ResponseWriter, r *http.Request) {
	select {
	case aoiChan <- struct{}{}:
		defer func() { <-aoiChan }()
		aoiAction(w, r)
	default:
		http.Error(w, "Server đang bận (max 5 request)", http.StatusTooManyRequests)
	}

}
func aoiAction(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("image")
	pp := r.FormValue("path")
	if pp == "" {
		http.Error(w, "path nil", http.StatusBadRequest)
		return
	}
	p := pathConfig(pp)
	if p == "" {
		http.Error(w, "path nil", http.StatusBadRequest)
		return
	}
	path := filepath.Join("Z:\\", p)

	defer file.Close()

	// Tạo file lưu
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		Fmt(fmtAOI, path, err)
		return
	}
	paths := filepath.Join(path, handler.Filename)
	dst, err := os.Create(paths)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy dữ liệu

	_, err = io.Copy(dst, file)
	if err != nil {
		dst.Close()
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
	Fmt(fmtAOI, "aoiAction", "Upload thành công: ", paths)
}

func isValidFolder(name string) bool {

	if strings.Contains(strings.ToLower(name), "master") {
		return true
	}
	re := regexp.MustCompile(`^\d{2}[A-Za-z]\d{2}$`)
	return re.MatchString(name)
}

func pathConfig(rPath string) string {

	part := strings.Split(strings.ToLower(rPath), "\\")
	if len(part) < 8 {
		return ""
	}
	nPath := ""
	for i, folder := range part {
		if i < 6 {
			nPath = filepath.Join(nPath, folder)

		} else if isValidFolder(folder) || strings.Contains(folder, "Tái") {
			nPath = filepath.Join(nPath, folder)

		} else if strings.Contains(folder, "day") {
			nPath = filepath.Join(nPath, "DayShift")

		} else if strings.Contains(folder, "nig") {
			nPath = filepath.Join(nPath, "NightShift")

		}
	}
	return nPath
}
