package main

import (
	"Locker/config"
	"Locker/script"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

func GET_headerEdit(w http.ResponseWriter, r *http.Request) {
	// truyền vào template (⚠ quan trọng: phải là string JSON)
	WebHeaderEdit, _ := template.ParseFiles(filepath.Join(config.WebPathDir, "ApiWeb", "WebHeaderEdit.html"))
	err := WebHeaderEdit.Execute(w, map[string]interface{}{})
	if err != nil {
		LogInfo(&Logger.API, "GET_headerEdit", "WebHeaderEdit Execute", err)
		http.Error(w, "not found", 404)
	}
}

func POST_headerEdit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model string `json:"model"`
		CD    string `json:"cd"`
		Data  string `json:"data"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	if req.Model == "" || req.CD == "" || req.Data == "" {
		w.Write([]byte("Fail"))
		return
	}
	err = CopyFileFull(filepath.Join(config.HttpMasterDir, "header.json"), filepath.Join(config.BackupPGM, fmt.Sprintf("header_%s.json", time.Now().Format("20060102_150405"))))
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	Headers.mu.Lock()
	model := Headers.model[req.Model]
	if model == nil {
		model = &headMolde{mu: sync.RWMutex{}, cd: map[string]*headCD{}}
		Headers.model[req.Model] = model
	}
	Headers.mu.Unlock()

	model.mu.Lock()
	cd := model.cd[req.CD]
	if cd == nil {
		cd = &headCD{mu: sync.RWMutex{}, list: []string{}}
		Headers.model[req.Model].cd[req.CD] = cd
	}
	model.mu.Unlock()

	cd.mu.Lock()
	for i := range cd.list {
		cd.list[i] = strings.NewReplacer("\r", "", "\n", "").Replace(cd.list[i])
	}

	value := strings.NewReplacer("\r", "", "\n", "").Replace(req.Data)
	for _, i := range strings.Split(value, "|") {
		if i != "" && !slices.Contains(cd.list, i) {
			cd.list = append(cd.list, i)
		}
	}
	cd.mu.Unlock()

	newMap := map[string]map[string][]string{}
	Headers.mu.RLock()
	for m, Data1 := range Headers.model {
		newMap[m] = map[string][]string{}
		Data1.mu.RLock()
		for c, Data2 := range Data1.cd {
			Data2.mu.RLock()
			newMap[m][c] = Data2.list
			Data2.mu.RUnlock()
		}
		Data1.mu.RUnlock()
	}
	Headers.mu.RUnlock()

	err = script.SaveJson(filepath.Join(config.HttpMasterDir, "header.json"), newMap)
	if err == nil {
		LogInfo(&Logger.API, "POST_headerEdit", "UPDATED:", req.Model, req.CD)
		w.Write([]byte("ok"))
	} else {
		w.Write([]byte(err.Error()))
	}
}
