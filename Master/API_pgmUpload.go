package main

import (
	"Locker/config"
	"Locker/script"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"time"
)

// get model,cd
func Get_pgmUpload(w http.ResponseWriter, r *http.Request) {
	model := r.URL.Query().Get("model")
	cd := r.URL.Query().Get("cd")
	modellist := map[string]struct{}{}
	cdlist := []string{"Marking", "AF", "OIS", "Tilt", "Prism", "Fra_AF", "Fra_OIS", "Flag"}
	pgm_master_list.mu.RLock()
	for i := range pgm_master_list.Data {
		m := strings.Split(i, "_")[0]
		_, ok := modellist[m]
		if !ok {
			modellist[m] = struct{}{}
		}
	}
	pgm_master_list.mu.RUnlock()
	webPGMZipUpload, _ := template.ParseFiles(filepath.Join(config.WebPathDir, "ApiWeb", "webPGMZipUpload.html"))
	err := webPGMZipUpload.Execute(w, map[string]interface{}{
		"model":     model,
		"cd":        cd,
		"modellist": slices.Collect(maps.Keys(modellist)),
		"cdlist":    cdlist,
	})
	if err != nil {
		LogInfo(&Logger.API, "PGMZipUpload", err)
	}
}

// Post model, cd, version, pid, mode, file
func POST_pgmUpload(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			LogInfo(&Logger.Debug, fmt.Sprintf("panic: %v\n%s", err, debug.Stack()))
		}
	}()
	err := r.ParseMultipartForm(100 << 20) // 100MB
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	model := r.FormValue("model")
	cd := r.FormValue("cd")
	version := r.FormValue("version")
	pid := r.FormValue("pid")
	mode := r.FormValue("mode")
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	saveDir := filepath.Join(config.HttpMasterDir, model, cd)
	os.MkdirAll(saveDir, 0755)
	filenameFull := fmt.Sprintf("%s_%s_%s_%s.zip", model, cd, version, pid)
	filename := fmt.Sprintf("%s_%s_%s_%s", model, cd, version, pid)
	savePath := filepath.Join(saveDir, filenameFull)
	oldPath := fmt.Sprintf("%s_%s", savePath, time.Now().Format("20060102_150405"))
	if _, err = os.Stat(savePath); err == nil {
		err = os.Rename(savePath, oldPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if mode == "remove" {
		pgm_master_list.mu.Lock()
		md := pgm_master_list.Data[model+"_"+cd]
		if md != nil {
			delete(pgm_master_list.Data, model+"_"+cd)
		}
		pgm_master_list.mu.Unlock()
	} else {
		dst, err := os.Create(savePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			dst.Close()
			return
		}
		dst.Close()

		md5, err := GetMD5(savePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			dst.Close()
			return
		}

		pgm_master_list.mu.Lock()
		md := pgm_master_list.Data[model+"_"+cd]
		if md == nil {
			md = map[string]string{}
			pgm_master_list.Data[model+"_"+cd] = md
		}
		md[filename] = md5
		pgm_master_list.mu.Unlock()
	}
	jsonFileTmp := filepath.Join(config.HttpMasterDir, "pgm_master_list.tmp")
	jsonFile := filepath.Join(config.HttpMasterDir, "pgm_master_list.json")
	pgm_master_list.mu.RLock()
	err = script.SaveJson(jsonFileTmp, pgm_master_list.Data)
	pgm_master_list.mu.RUnlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = os.Rename(jsonFile, filepath.Join(config.BackupPGM, "pgm_master_list_"+time.Now().Format("20060102_150405")+".json"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = os.Rename(jsonFileTmp, jsonFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"msg":     "Upload success",
	})

}
