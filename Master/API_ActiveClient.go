package main

import (
	"Locker/config"
	"Locker/script"
	"encoding/json"
	"net/http"
	"strings"
)

type apiActiveClientType struct {
	Mode  string `json:"mode"`
	Value string `json:"value"`
}

func apiActiveClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var d apiActiveClientType
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), 400)
		Fmt(fmtMaster, "apiActiveClient", err)
		return
	}
	if len(d.Value) != 6 {
		http.Error(w, "Len Value !=6 ", 400)
		Fmt(fmtMaster, "apiActiveClient", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(string("OK")))
	tmp := script.RegRead(config.RegPath, "ActiveClient")
	newTmp := ""
	if d.Mode == "ADD" {
		newTmp = d.Value
	}

	newMap := map[string]struct{}{}
	newMap[d.Value] = struct{}{}
	for _, i := range strings.Split(tmp, "|") {
		if len(i) == 6 {
			if i == d.Value && d.Mode != "ADD" {
				delete(newMap, i)
				continue
			}
			_, ok := newMap[i]
			if ok {
				continue
			}
			newTmp = newTmp + "|" + i
			newMap[i] = struct{}{}
		}
	}
	script.RegWrite(config.RegPath, "ActiveClient", newTmp)
	listActive.mu.Lock()
	listActive.data = newMap
	listActive.mu.Unlock()
	if d.Mode != "ADD" {
		clients.mu.Lock()
		for key := range clients.Client {
			if strings.HasPrefix(key, d.Value) {
				delete(clients.Client, key)
			}
		}
		clients.mu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(string(newTmp)))
}
