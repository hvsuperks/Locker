package main

import (
	"Locker/config"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kardianos/service"
)

func (p *program) Stop(s service.Service) error {
	return nil
}

type program struct{}

func (p *program) Start(s service.Service) error {

	go p.run() // rất quan trọng
	return nil
}

var ServiceName = "NTanh" //"NTanh"

func ServiceRegister(dst string) bool {
	src, _ := os.Executable()

	// Nếu đang chạy đúng từ file dịch vụ chính -> Thoát hàm để vào luồng main chính
	if ComparePath(src, dst) {
		return true
	}
	clien := &http.Client{
		Timeout: time.Second,
	}
	clien.Get("http://127.0.0.1:50001/shutdown")
	for range 10 {
		if !IsRunning(dst) {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
	// 1. Lấy thông tin service
	s, err := GetService(ServiceName, dst)
	if err != nil {
		LogInfo(&Logger.MAIN, "ERROR: Get service failed:", err)
		ShowErrorPopup("ERROR", "Get service failed: "+err.Error())
		return false
	}
	// 2. Kiểm tra xem service đã cài trên hệ thống chưa
	status, err := s.Status()
	isInstalled := (err == nil && status != service.StatusUnknown)
	if isInstalled {
		Cmd(run, "taskkill", "/IM", config.DstAppName)
		_ = s.Stop()
		// Chờ service dừng hoàn toàn để nhả khóa file AutoUpdate.exe
		for range 30 {
			st, err := s.Status()
			if err == nil && st == service.StatusStopped {
				break
			}
			time.Sleep(time.Millisecond)
		}
	} else {
		// Chưa cài -> Tiến hành Install
		if err := s.Install(); err != nil {
			ShowErrorPopup("ERROR", "Get service failed: "+err.Error())
			return false
		}
	}
	// 3. Copy file thực thi mới đè vào System32
	copyErr := CopyFileFull(src, dst)
	if copyErr != nil {
		ShowErrorPopup("ERROR", "Copy File failed: "+copyErr.Error())
		return false
	}
	// 4. Kích hoạt service chạy file từ dst
	if err := s.Start(); err != nil {
		ShowErrorPopup("ERROR", "Service Start failed: "+err.Error())
		return false
	}
	// 5. THOÁT tiến trình tạm, nhường quyền cho tiến trình Service vừa start
	ShowErrorPopup("INFO", "UpDate Pass")
	// Cấu hình SCM tự restart sau 5 giây khi service bị crash/ngắt đột ngột
	if err := Cmd(run, "sc", "failure", ServiceName, "reset=", "86400", "actions=", "restart/10000"); err != nil {
	}
	return false
}

func GetService(name, path string) (service.Service, error) {
	cfg := &service.Config{
		Name:        name,
		DisplayName: fmt.Sprintf("%s Service", name),
		Description: fmt.Sprintf("%s Service", name),
		Executable:  path,
	}
	// BẮT BUỘC truyền &program{} thay vì nil
	return service.New(&program{}, cfg)
}

func (p *program) run() {
	lastcheckMap.Store(time.Now())
	var ctx, cancel = context.WithCancel(context.Background())
	go InitLog(config.LogPath_master+"API", &Logger.API)
	go InitLog(config.LogPath_master+"CSV", &Logger.CSV)
	go InitLog(config.LogPath_master+"MAIN", &Logger.MAIN)
	go InitLog(config.LogPath_master+"AOI", &Logger.AOI)
	go InitLog(config.LogPath_master+"Debug", &Logger.Debug)
	time.Sleep(time.Second)
	Start(ctx, cancel, Ver)
}
