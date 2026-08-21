package main

import (
	"Locker/config"
	"encoding/json"
	"net/http"
)

func clientGetVer(w http.ResponseWriter, r *http.Request) {
	Type := r.URL.Query().Get("type")
	if Type != "locker" {
		http.Error(w, "not found", 404)
		return
	}
	w.Header().Set("Con*ent-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"locker": config.MasterLocker,
		"ui":     config.MasterUILocker,
	})
}
