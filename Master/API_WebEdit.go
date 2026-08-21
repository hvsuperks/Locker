package main

import (
	"Locker/config"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Req struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func WebeditPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Fail", http.StatusMethodNotAllowed)
		return
	}
	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Content == "" {
		http.Error(w, "Data Fail", http.StatusBadRequest)
		return
	}
	dst := filepath.Join(config.WebPathDir, "ApiWeb", req.Name)
	err := os.WriteFile(dst, []byte(req.Content), 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func Webedit(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	webpath := filepath.Join(config.WebPathDir, "ApiWeb", name)
	if !strings.HasSuffix(webpath, ".html") {
		webpath = webpath + ".html"
	}
	data, err := os.ReadFile(webpath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(data)
}

func WebapiList(w http.ResponseWriter, r *http.Request) {
	webpath := filepath.Join(config.WebPathDir, "ApiWeb", "*.html")
	files, _ := filepath.Glob(webpath)

	var names []string

	for _, f := range files {
		names = append(names, filepath.Base(f))
	}
	json.NewEncoder(w).Encode(names)
}
