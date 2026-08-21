package main

import (
	"Locker/script"
	"net/http"
	"time"
)

func resVNC(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	script.Cmd("taskkill", "/F", "/IM", "winvnc.exe")
	time.Sleep(4 * time.Second)
	script.Cmd("C:\\Program Files\\uvnc bvba\\UltraVNC\\winvnc.exe", "-install")
	Fmt(fmtMaster, "Master Restart VNC")
}
