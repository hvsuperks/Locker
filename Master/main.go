package main

import (
	"Locker/config"
	_ "embed"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	_ "net/http/pprof"
)

//go:embed icon.png
var iconBytes []byte

var wg sync.WaitGroup
var Ver = config.Ver

func main() {
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	config.StartTime = time.Now().Format("2006/01/02 15:04:05")
	config.IsDebug = true
	config.IP.Store("")
	// Đường dẫn tiêu chuẩn
	dstAppRootPath := filepath.Join(config.AppRootDir, config.DstAppName)
	if !ServiceRegister(dstAppRootPath) {
		return
	}
	// Khởi tạo khung Service
	s, err := GetService(ServiceName, dstAppRootPath)
	if err != nil {
		ShowErrorPopup("ERROR: Init service framework failed:", err.Error())
		return
	}
	// Bắt đầu vòng đời chính của Windows Service
	if err := s.Run(); err != nil {
		ShowErrorPopup("ERROR: Service run failed:", err.Error())
	}

}
