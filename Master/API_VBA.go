package main

import (
	"encoding/json"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func getListRawDataVBA(w http.ResponseWriter, r *http.Request) {

	// 1. Lấy tham số tầng 1 từ URL
	parent := r.URL.Query().Get("key")

	// 2. Lấy mảng dữ liệu tương ứng (thực tế sẽ gọi DB hoặc Map toàn cục)
	var result []string
	if parent == "model" {
		k := maps.Keys(HeadersRaw)
		result = slices.Collect(k)

	} else {
		k := maps.Keys(HeadersRaw[parent])
		result = slices.Collect(k)
	}
	chuoiTraVe := strings.Join(result, "|")
	// 3. Trả về dạng JSON Array thẳng luôn: ["Cầu Giấy", "Ba Đình", ...]
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(chuoiTraVe))
}

func getListChetao(w http.ResponseWriter, r *http.Request) {

	room := []string{"AB", "CD", "X3T1", "X3T2", "X3T3"}
	m := make(map[string]map[string][]string)
	for _, xuong := range room {
		m[xuong] = make(map[string][]string)
		models, err := os.ReadDir(filepath.Join("Z:\\", xuong))
		if err != nil {
			continue
		}
		for _, model := range models {
			cd := []string{}
			cds, err := os.ReadDir(filepath.Join("Z:\\", xuong, model.Name()))
			if err != nil {

				continue
			}
			for _, i := range cds {
				cd = append(cd, i.Name())
			}
			m[xuong][model.Name()] = cd
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}
