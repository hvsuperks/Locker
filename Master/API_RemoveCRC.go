package main

import (
	"Locker/config"
	"encoding/json"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type removeMap struct {
	mu   sync.RWMutex
	Data map[string]*removeCRCID
}
type removeCRCID struct {
	mu   sync.RWMutex
	Data map[string]map[string]struct{}
}

var RemoveMap = &removeMap{
	mu:   sync.RWMutex{},
	Data: map[string]*removeCRCID{},
}

func RemoveCRC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CRCid string              `json:"crcid"`
		Data  map[string][]string `json:"data"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	RemoveMap.mu.Lock()
	model, ok := RemoveMap.Data[req.CRCid]
	Fmt(fmtClient, req.CRCid)
	if !ok {
		model = &removeCRCID{
			mu:   sync.RWMutex{},
			Data: map[string]map[string]struct{}{},
		}
		RemoveMap.Data[req.CRCid] = model
	}
	RemoveMap.mu.Unlock()

	for k, v := range req.Data {
		model.mu.Lock()
		section, ok := model.Data[k]
		if !ok {
			section = map[string]struct{}{}
		}
		model.Data[k] = section
		for _, i := range v {
			section[i] = struct{}{}
		}
		model.mu.Unlock()
	}
}

func BuildResponse() map[string]map[string][]string {
	RemoveMap.mu.RLock()
	models := make(map[string]*removeCRCID, len(RemoveMap.Data))
	maps.Copy(models, RemoveMap.Data)
	RemoveMap.mu.RUnlock()
	result := make(map[string]map[string][]string)
	for crcid, model := range models {
		model.mu.RLock()
		sectionMap := make(map[string][]string, len(model.Data))
		for section, items := range model.Data {
			arr := make([]string, 0, len(items))
			for item := range items {
				arr = append(arr, item)
			}
			sectionMap[section] = arr
		}
		model.mu.RUnlock()
		result[crcid] = sectionMap
	}
	return result
}

func WebRemoveCRC(w http.ResponseWriter, r *http.Request) {
	webpath := filepath.Join(config.WebPathDir, "ApiWeb", "WebRemoveCRC.html")
	data, err := os.ReadFile(webpath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(data)
}
