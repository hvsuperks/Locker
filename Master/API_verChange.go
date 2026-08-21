package main

import (
	"Locker/config"
	"Locker/script"
	"encoding/json"
	"net/http"
	"strings"
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
		Fmt(fmtMaster, "verChange", err)
		return
	}
	Fmt(fmtMaster, d.Mode, d.Value)
	change := func(pl *string, key, value string) bool {
		if script.RegWrite(config.RegPath, key, value) {
			*pl = value
			return true
		} else {
			http.Error(w, "RegWrite locker Fail", 444)
			Fmt(fmtMaster, "verChange", "RegWrite locker Fail")
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
	default:
		http.Error(w, "RegWrite mode Fail"+d.Mode+" "+d.Value, 445)
		Fmt(fmtMaster, "verChange", "RegWrite mode Fail"+d.Mode+" "+d.Value)
		return
	}

	if i {
		// trả về cho web
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
		Fmt(fmtMaster, "verChange", d.Mode, d.Value)
	}
}
