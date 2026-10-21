package main

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var HttpGetCMD = atomic.Value{}
var Uiver = atomic.Value{}
var chetao = atomic.Value{}
var CurrentUser string
var uploadPath = ""
var syncFolder = ""
var driveLetter = ""
var RootFolder = ""
var wm = WatcherManager{
	watched: map[string]bool{},
	mu:      sync.Mutex{},
}

var RegRoot = "SOFTWARE\\Produc\\AOI" // SOFTWARE\Produc\AOI

func main() {
	HttpGetCMD.Store("")
	Uiver.Store("")
	chetao.Store("")
	if !ServiceRegister() {
		return
	}
	// Khởi tạo khung Service
	s, err := GetService(serviceName, servicePath)
	if err != nil {
		LogUnique("ERROR: Init service framework failed:", err)
		return
	}
	// Bắt đầu vòng đời chính của Windows Service
	if err := s.Run(); err != nil {
		LogUnique("ERROR: Service run failed:", err)
	}
}

func findScreenCapture() {
	defer func() {
		if err := recover(); err != nil {
			LogUnique("bắt được panic:" + err.(string))
		}
	}()
	r, err := GetLatestNGImagePath(driveLetter)
	if err != nil || r == "" {
		filepath.WalkDir(driveLetter, func(path string, d fs.DirEntry, err error) error {
			filename := strings.ToLower(d.Name())
			if d.IsDir() {
				if strings.Contains(filename, "aoi") || strings.Contains(filename, "upload") || strings.Contains(filename, "del") ||
					strings.Contains(filename, "auto") || strings.Contains(filename, "pgm") {
					return filepath.SkipDir
				}
				return nil
			} else {
				ext := filepath.Ext(path)
				if ext == ".jpg" || ext == ".bmp" || ext == ".png" {
					matched, _ := regexp.MatchString(`(?i)screencapture[\\/]\d{4}[\\/]\d{1,2}[\\/]\d{1,2}`, path)
					if matched {
						idx := strings.Index(strings.ToLower(path), "screencapture")
						r = filepath.Join(path[:idx], "screencapture")
						return filepath.SkipAll
					}
				}
			}
			return nil
		})
		if r == "" {
			LogUnique("Get Path Errp :" + err.Error())
			return
		}
	}
	RootFolder = r
	go wm.SyncRealtimeFile(RootFolder)
	go wm.StartMonitoring(RootFolder)
	var d map[string]bool
	err, d = LoadJson("upload.json")
	if err == nil && d != nil {
		listUploadDone = d
	}
	wm.syncUpload()
	ticker := time.NewTicker(60 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		wm.syncUpload()
	}
}
