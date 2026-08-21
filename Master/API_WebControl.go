package main

import (
	"Locker/config"
	"Locker/script"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func resetVNC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Fail", http.StatusMethodNotAllowed)
		return
	}
	ID := r.FormValue("id")
	go MultiClientSend(ID, "resetVNC", "resetVNC", nil)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	Fmt(fmtClient, "Reset VNC", ID)
}

func resetMES(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Fail", http.StatusMethodNotAllowed)
		return
	}
	ID := r.FormValue("id")
	go MultiClientSend(ID, "cmd", "taskkill|/IM|jahwa_ecm_agent_v2.exe|/F", nil)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	Fmt(fmtClient, "Reset MES", ID)
}

func apiCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Fail", http.StatusMethodNotAllowed)
		return
	}
	ID := r.FormValue("id")
	Action := r.FormValue("action")
	if strings.ToLower(ID) == "master" {
		script.Cmd("cmd", "/c", Action)
	} else {
		go MultiClientSend(ID, "cmd", Action, nil)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	Fmt(fmtClient, "cmd", Action, ID)
}

func removeClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Fail", http.StatusMethodNotAllowed)
		return
	}
	ID := r.FormValue("id")
	go MultiClientSend(ID, "removeApp", "removeApp", nil)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	Fmt(fmtClient, "removeClient", ID)
}

func LockerChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Fail", http.StatusMethodNotAllowed)
		return
	}
	ID := r.FormValue("id")
	mode := r.FormValue("mode")
	state := r.FormValue("state")
	if ID == "" || mode == "" || state == "" {
		http.Error(w, "FormValue Fail", http.StatusBadRequest)
		return
	}
	if mode != "lockercrc" && mode != "lockerpgm" {
		http.Error(w, "Mode Fail (lockercrc,lockerpgm)", http.StatusBadRequest)
		return
	}
	if len(ID) == 9 {
		ClientWorkerFuncSet(ID, "LockerChange_"+mode, state, nil)
	} else {
		go MultiClientSend(ID, "LockerChange_"+mode, state, nil)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	Fmt(fmtClient, "LockerChange", ID, mode, state)
}

func ModelConfigChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Fail", http.StatusMethodNotAllowed)
		return
	}
	ID := r.FormValue("id")
	mode := r.FormValue("mode")
	value := r.FormValue("value")

	Client_list_config.mu.RLock()
	crc := Client_list_config.Data[ID].CRC
	pgm := Client_list_config.Data[ID].PGM
	Client_list_config.mu.RUnlock()
	var cp config.P
	if mode == "crc" {
		cp = config.P{
			PGM: pgm,
			CRC: value,
		}
	} else if mode == "pgm" {
		cp = config.P{
			PGM: value,
			CRC: crc,
		}
	} else {
		http.Error(w, "Mode Fail", http.StatusBadRequest)
		return
	}
	Client_list_config.mu.Lock()
	Client_list_config.Data[ID] = cp
	err := script.SaveJson(filepath.Join(config.HttpMasterDir, "Model_config.json"), Client_list_config.Data)
	Client_list_config.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		os.Exit(0)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	Fmt(fmtClient, "ModelConfigChange", ID, mode, value)
}
