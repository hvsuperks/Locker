package main

import (
	"Locker/config"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
)

func verChange(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var d apiVerStruct
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), 400)
		LogInfo(&Logger.API, "verChange", err)
		return
	}
	LogInfo(&Logger.API, d.Mode, d.Value)
	change := func(pl *atomic.Value, key, value string) bool {
		if RegWrite(config.RegPath, key, value) {
			pl.Store(value)
			return true
		} else {
			http.Error(w, "RegWrite locker Fail", 444)
			LogInfo(&Logger.API, "verChange", "RegWrite locker Fail")
			return false
		}
	}
	i := false
	m := strings.ToLower(d.Mode)
	switch m {
	case "locker":
		i = change(&config.MasterLocker, "lockerVer", d.Value)
	case "aoi":
		i = change(&config.MasterAOI, "aoiVer", d.Value)
	case "uilocker":
		i = change(&config.MasterUILocker, "uilockerVer", d.Value)
	case "uiaoi":
		i = change(&config.MasterUIAoi, "uiaoiVer", d.Value)
	case "autoupdate":
		i = change(&config.MasterService, "serviceVer", d.Value)
	case "vbatool":
		i = change(&config.MasterVbatoolVer, "vbatoolVer", d.Value)

	default:
		http.Error(w, "RegWrite mode Fail"+d.Mode+" "+d.Value, 445)
		LogInfo(&Logger.API, "verChange", "RegWrite mode Fail"+d.Mode+" "+d.Value)
		return
	}

	if i {
		// trả về cho web
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
		LogInfo(&Logger.API, "verChange", d.Mode, d.Value)
	}
}
