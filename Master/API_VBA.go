package main

import (
	"Locker/config"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
)

func getListRawDataVBA(w http.ResponseWriter, r *http.Request) {

	// 1. Lấy tham số tầng 1 từ URL
	parent := r.URL.Query().Get("key")

	// 2. Lấy mảng dữ liệu tương ứng (thực tế sẽ gọi DB hoặc Map toàn cục)
	var result = ""
	path := `D:\Master\RawData\` + parent
	if parent == "model" {
		path = `D:\Master\RawData\`
	}

	l, err := os.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	for _, i := range l {
		result = result + "|" + i.Name()
	}
	if len(result) > 0 {
		result = result[1:]
	}
	// 3. Trả về dạng JSON Array thẳng luôn: ["Cầu Giấy", "Ba Đình", ...]
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(result))
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

func GET_VBArawdata(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			http.Error(w, fmt.Sprintf("panic: %v - %s", r, string(debug.Stack())), http.StatusBadRequest)
			LogInfo(&Logger.Debug, fmt.Sprintf("panic: %v", r))
		}
	}()
	model := r.URL.Query().Get("model")
	part := r.URL.Query().Get("cd")
	lotString := r.URL.Query().Get("lot")
	SourcePath := filepath.Join(
		config.BackupRawData,
		model, part,
	)
	lots := strings.Split(lotString, ",")
	date := ""
	paths := []string{}
	for _, d := range lots {
		if len(d) > 6 {
			i := strings.Split(d, `-`)
			if len(i) != 2 || len(i[0]) != 6 {
				continue
			}
			date = i[0]
			d = i[1]
		}
		lot := strings.Split(d, "~")
		if len(lot) > 1 {
			mi, err := strconv.Atoi(lot[0])
			ma, err1 := strconv.Atoi(lot[1])
			if err != nil || err1 != nil {
				continue
			}
			for i := mi; i <= ma; i++ {
				paths = append(paths, filepath.Join(SourcePath, date, fmt.Sprintf("%s%s_%s_%02d", model, date, part, i)))
			}
		} else {
			l, err := strconv.Atoi(lot[0])
			if err != nil {
				continue
			}
			paths = append(paths, filepath.Join(SourcePath, date, fmt.Sprintf("%s%s_%s_%02d", model, date, part, l)))
		}
	}

	if len(paths) == 0 {
		http.Error(w, "Không tìm thấy File nào", http.StatusNotFound)
		return
	}
	csvData := []map[string]string{}
	csvHead := []string{}
	for _, path := range paths {
		LogInfo(&Logger.Debug, path)
		if listcsv, err := os.ReadDir(path); err == nil {
			for _, i := range listcsv {
				if strings.HasSuffix(i.Name(), `.csv`) {
					p := filepath.Join(path, i.Name())
					LogInfo(&Logger.Debug, p)
					readDataCsv(i.Name(), p, &csvHead, &csvData)
				}
			}
		}
	}
	tmpR := []string{`source;` + strings.Join(csvHead, ";")}
	if len(csvHead) == 0 || len(csvData) == 0 {
		http.Error(w, "Không tìm thấy Nội dung File", http.StatusNotFound)
		return
	}

	tmp := make([]string, len(csvHead)+1)
	for _, row := range csvData {
		tmp[0] = row[`source`]
		for i, key := range csvHead {
			value, ok := row[key]
			if part == "AF" && (!ok || value == "-") {
				value, ok = row[key+"_90"]
				if value == "-" || !ok {
					value = row[key+"_180"]
				}
			}
			if value == "" {
				value = "-"
			}
			tmp[i+1] = value
		}
		tmpR = append(tmpR, strings.Join(tmp, ";"))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(strings.Join(tmpR, "|")))
}

func readDataCsv(name, path string, csvHead *[]string, csvData *[]map[string]string) {
	f, err := os.Open(path)
	if err != nil {
		LogInfo(&Logger.Debug, path, err)
		return
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	header, _ := reader.Read()
	LogInfo(&Logger.Debug, err)
	n := true
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		tmp := map[string]string{}
		tmp[`source`] = name
		for i, key := range header {
			if n {
				if !slices.Contains(*csvHead, key) && !strings.Contains(key, "_90") && !strings.Contains(key, "_180") && !strings.Contains(key, "_270") {
					*csvHead = append(*csvHead, key)
				}
			}
			if i > len(row)-1 {
				tmp[key] = "-"
				continue
			}
			tmp[key] = row[i]
			if row[i] == "" {
				tmp[key] = "-"
			}
		}
		n = false
		*csvData = append(*csvData, tmp)
	}
}

func GET_VBAtool(w http.ResponseWriter, r *http.Request) {
	toolver := r.URL.Query().Get("ver")
	currentVer := config.MasterVbatoolVer.Load().(string)
	if currentVer == "" {
		http.Error(w, "Master Vision Not Foud", http.StatusNotFound)
		LogInfo(&Logger.Debug, "Master Vision Not Foud", toolver, currentVer)
		return
	}
	if toolver == currentVer {
		http.Error(w, "skip", http.StatusNoContent)
		return
	}

	path := filepath.Join(config.HttpMasterDir, fmt.Sprintf("Tool_%s.xlsb", currentVer))
	if _, err := os.Stat(path); err != nil {
		http.Error(w, "Master Tool File Not Foud", http.StatusNotFound)
		LogInfo(&Logger.Debug, "Master Tool File Not Foud", path)
		return
	}
	w.Header().Set("ver", currentVer)
	http.ServeFile(w, r, path)
}
