package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kardianos/service"
)

var ServiceName = "NTanh"                                 //"NTanh"
var AutoUpdatePath = `C:\Windows\System32\AutoUpdate.exe` // `C:\Windows\System32\AutoUpdate.exe`
var WatcherPath = `C:\Windows\System32\Watcher.exe`       //`C:\Windows\System32\AutoUpdate.exe`
var updateServicePath = `C:\programdata\locker\updateService`

func ServiceRegister() bool {
	src, _ := os.Executable()

	// Nếu đang chạy đúng từ file dịch vụ chính -> Thoát hàm để vào luồng main chính
	if ComparePath(src, AutoUpdatePath) {
		return true
	}
	f, err := os.Create(updateServicePath)
	if err != nil {
		return false
	}
	f.Close()
	defer func() {
		os.Remove(updateServicePath)
	}()
	// 1. Lấy thông tin service
	s, err := GetService(ServiceName, AutoUpdatePath)
	if err != nil {
		LogUnique("ERROR: Get service failed:", err)
		ShowErrorPopup("ERROR", "Get service failed: "+err.Error())
		return false
	}
	// 2. Kiểm tra xem service đã cài trên hệ thống chưa
	status, err := s.Status()
	isInstalled := (err == nil && status != service.StatusUnknown)
	if isInstalled {
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
			LogUnique("ERROR: Service Install failed:", err)
			ShowErrorPopup("ERROR", "Get service failed: "+err.Error())
			return false
		}
	}
	// 3. Copy file thực thi mới đè vào System32
	copyErr := CopyFileFull(src, AutoUpdatePath)
	if copyErr != nil {
		LogUnique("ERROR: Copy file failed:", copyErr)
		ShowErrorPopup("ERROR", "Copy File failed: "+copyErr.Error())
		return false
	}
	// 4. Kích hoạt service chạy file từ dst
	if err := s.Start(); err != nil {
		LogUnique("ERROR: Service Start failed:", err)
		ShowErrorPopup("ERROR", "Service Start failed: "+err.Error())
		return false
	}
	// 5. THOÁT tiến trình tạm, nhường quyền cho tiến trình Service vừa start
	LogUnique("INFO: Registered and started service successfully. Exiting setup process.")
	ShowErrorPopup("INFO", "UpDate Pass")
	// Cấu hình SCM tự restart sau 5 giây khi service bị crash/ngắt đột ngột
	AddSystemPath(`C:\ProgramData\Locker`)
	if err := Cmd(run, "sc", "failure", ServiceName, "reset=", "86400", "actions=", "restart/10000"); err != nil {
		LogUnique("WARNING: Failed to set service failure actions:", err)
	}
	RemoveHP()
	return false
}

func ComparePath(scr, dst string) bool {
	// 1. Đưa về đường dẫn tuyệt đối và xử lý Shortcut/Symlink
	path1, err1 := filepath.Abs(scr)
	path2, err2 := filepath.Abs(dst)
	if err1 != nil || err2 != nil {
		return false
	}

	// 2. Làm sạch đường dẫn (xóa các dấu . hoặc .. thừa)
	path1 = filepath.Clean(path1)
	path2 = filepath.Clean(path2)

	// 3. Với Windows: Phải chuyển về cùng chữ thường vì Windows không phân biệt hoa/thường
	// "C:\Users" và "c:\users" là một
	return strings.EqualFold(path1, path2)
}

type program struct{}

func (p *program) Start(s service.Service) error {

	go p.run() // rất quan trọng
	return nil
}

func (p *program) run() {
	WaitUserLogin()
	HttpGetCMD.Store("")
	mutex := checkSignApp()
	_ = mutex
	LogUnique(OpenUDP50000())
	AppUi.Ver.Store(RegRead(regRoot, "verUi"))
	AppLocker.Ver.Store(RegRead(regRoot, "verLocker"))
	AppUi.Last.Store(time.Now())
	AppLocker.Last.Store(time.Now())
	go pipeStart("Service_ui", &AppUi.Ver, &AppUi.Last)
	go pipeStart("Service_locker", &AppLocker.Ver, &AppLocker.Last)
	IP.Store("")
	// Get CMD từ master
	go watcherService()
	go func() {
		httpget := &http.Client{
			Timeout: 2 * time.Second,
		}
		for {
			time.Sleep(3 * time.Second)
			ID := ID_Read()
			if ID != "" && IP.Load().(string) != "" {
				url := HttpGetCMD.Load().(string) + ID
				resp, err := httpget.Get(url)
				if err != nil {
					continue
				}
				if resp.StatusCode != http.StatusOK {
					resp.Body.Close()
					continue
				}
				data, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					continue
				}
				txt := string(data)
				Cmd(start, "cmd", "/c", txt)
			}
		}
	}()
	go SyncTime()
	// Watercher
	go func() {
		for {
			for {
				if _, err := os.Stat("C:\\Programdata\\Locker\\Fail"); err == nil {
					time.Sleep(time.Second)
					continue
				}
				break
			}
			ctxWatcher, cancelWatcher := context.WithCancel(context.Background())
			go Watcher(ctxWatcher)
			for {
				if _, err := os.Stat("C:\\Programdata\\Locker\\Fail"); err == nil {
					cancelWatcher()
					isShutdownLocker.Store(true)
					taskkill("Ui.exe")
					time.Sleep(2 * time.Second)
					taskkill("Locker.exe")
					break
				}
				time.Sleep(time.Second)
			}
		}
	}()
	go ListenMasterUDP()
}

func (p *program) Stop(s service.Service) error {
	return nil
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

func watcherService() {
	for {
		time.Sleep(2 * time.Second)
		s, _ := GetService("Watcher", WatcherPath)
		for {
			if _, err := os.Stat(WatcherPath); err != nil {
				_ = os.WriteFile(WatcherPath, WatcherBin, 0755)
			}
			status, err := s.Status()
			if err != nil {
				// service bị xóa
				_ = s.Install()
				_ = s.Start()
				break
			}

			if status != service.StatusRunning {
				_ = s.Start()
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	}
}
