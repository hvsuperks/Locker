package main

import (
	"Locker/config"
	"Locker/script"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func POST_resetVNC(w http.ResponseWriter, r *http.Request) {
	ID := r.FormValue("id")
	if strings.ToLower(ID) == "master" {
		w.WriteHeader(http.StatusOK)
		Cmd(run, "taskkill", "/F", "/IM", "winvnc.exe")
		time.Sleep(3 * time.Second)
		Cmd(start, "C:\\Program Files\\uvnc bvba\\UltraVNC\\winvnc.exe", "-install")
	} else {
		go MultiClientSend(ID, "resetVNC", "resetVNC", nil)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	LogInfo(&Logger.API, "Reset VNC", ID)
}

func POST_resetMES(w http.ResponseWriter, r *http.Request) {
	ID := r.FormValue("id")
	go MultiClientSend(ID, "cmd", "taskkill|/IM|jahwa_ecm_agent_v2.exe|/F", nil)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	LogInfo(&Logger.API, "Reset MES", ID)
}

func POST_apiCommand(w http.ResponseWriter, r *http.Request) {
	ID := r.FormValue("id")
	Action := r.FormValue("action")
	if strings.ToLower(ID) == "master" {
		Cmd(start, "cmd", "/c", Action)
	} else {
		go MultiClientSend(ID, "cmd", Action, nil)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	LogInfo(&Logger.API, "cmd", Action, ID)
}

func POST_clientRemove(w http.ResponseWriter, r *http.Request) {
	ID := r.FormValue("id")
	go MultiClientSend(ID, "removeApp", "removeApp", nil)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	LogInfo(&Logger.API, "removeClient", ID)
}

func POST_LockerChange(w http.ResponseWriter, r *http.Request) {
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
	LogInfo(&Logger.API, "LockerChange", ID, mode, state)
}

func POST_modelConfig(w http.ResponseWriter, r *http.Request) {
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
	LogInfo(&Logger.API, "ModelConfigChange", ID, mode, value)
}
