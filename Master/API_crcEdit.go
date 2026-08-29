package main

import (
	"Locker/config"
	"Locker/script"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"time"
)

func POST_crcEdit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string                       `json:"name"`
		Mode string                       `json:"mode"`
		Data map[string]map[string]string `json:"data"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	if req.Name == "" || req.Data == nil {
		w.Write([]byte("Fail"))
		return
	}
	err = CopyFileFull(filepath.Join(config.HttpMasterDir, "crc_master_list.json"), filepath.Join(config.BackupPGM, fmt.Sprintf("crc_master_list_%s.json", time.Now().Format("20060102_150405"))))
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	crc_master_list.mu.Lock()
	if req.Mode == "remove" {
		delete(crc_master_list.Data, req.Name)
	} else {
		crc_master_list.Data[req.Name] = req.Data
	}
	err = script.SaveJson(filepath.Join(config.HttpMasterDir, "crc_master_list.json"), crc_master_list.Data)
	crc_master_list.mu.Unlock()
	if err == nil {
		LogInfo(&Logger.API, "POST_crcEdit", req.Mode, req.Name)
		w.Write([]byte("ok"))
	} else {
		w.Write([]byte(err.Error()))
	}
}

func GET_crcEdit(w http.ResponseWriter, r *http.Request) {
	Name := r.URL.Query().Get("name")
	// convert map → JSON string
	crc_master_list.mu.RLock()
	defer crc_master_list.mu.RUnlock()
	data, ok := crc_master_list.Data[Name]
	WebCRCEdit, _ := template.ParseFiles(filepath.Join(config.WebPathDir, "ApiWeb", "WebCRCEdit.html"))
	if !ok {
		err := WebCRCEdit.Execute(w, map[string]any{})
		if err != nil {
			LogInfo(&Logger.API, "GET_crcEdit", "WebCRCEdit Execute", err)
			http.Error(w, "not found", 404)
		}
		return
	}
	j, _ := json.Marshal(data)
	// truyền vào template (⚠ quan trọng: phải là string JSON)
	c := GetRemoveMap(Name)
	LogInfo(&Logger.API, "GET_crcEdit", "GetRemoveMap", Name, c)
	err := WebCRCEdit.Execute(w, map[string]any{
		"name":      Name,
		"data":      string(j), // ❗ đổi ở đây (không dùng template.JS)
		"removemap": c,
	})
	if err != nil {
		LogInfo(&Logger.API, "GET_crcEdit", "WebCRCEdit Execute", err)
		http.Error(w, "not found", 404)
	}
}

func GetRemoveMap(name string) string {

	RemoveMap.mu.RLock()
	model, ok := RemoveMap.Data[name]
	RemoveMap.mu.RUnlock()
	if !ok {
		return "{}"
	}
	model.mu.RLock()
	defer model.mu.RUnlock()
	section := model.Data
	if section == nil {
		return "{}"
	}
	j, err := json.Marshal(section)
	if err != nil {
		return "{}"
	}
	return string(j)
}
