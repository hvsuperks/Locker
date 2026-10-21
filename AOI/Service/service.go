package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kardianos/service"
)

var serviceName = "NTanh"                                  // NTanh
var updateFlag = `C:\programdata\AOI\updateService`        // C:\programdata\AOI\updateService
var servicePath = `C:\programdata\AOI\aoi.exe`             // C:\programdata\AOI\aoi.exe
var serviceUpdatePath = `C:\programdata\AOI\updateAOI.exe` // C:\programdata\AOI\updateAOI.exe
var uiPath = `C:\programdata\AOI\updateAOI.exe`            // C:\programdata\AOI\Ui.exe
var uiUpdatePath = `C:\programdata\AOI\updateAOI.exe`      // C:\programdata\AOI\updateUi.exe
var uploadJsonPath = `C:\programdata\AOI\upload.json`      // C:\programdata\AOI\upload.json
var AutoUpdatePort = 10000                                 // 10000

type program struct{}

func (p *program) Stop(s service.Service) error {
	return nil
}

func (p *program) Start(s service.Service) error {
	go p.run() // rất quan trọng
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

func (p *program) run() {
	WaitUserLogin()
	driverL := []string{"D:\\", "E:\\", "F:\\", "G:\\", "H:\\"}
	for _, i := range driverL {
		if _, err := os.ReadDir(i); err == nil {
			uploadPath = i + "upload"
			syncFolder = i + "Backup_AOI"
			driveLetter = i
			break
		}
	}
	InitLog()
	if driveLetter == "" {
		LogUnique("Không tìm thấy ổ đĩa")
		os.Exit(1)
	}
	go findScreenCapture()
	i := RegRead(RegRoot, "chetao")
	if i != "" {
		chetao.Store(strings.ReplaceAll(i, "|", "\\"))
	}
	var err error
	err, listUploadDone = LoadJson(uploadJsonPath)
	if err != nil {
		LogUnique(err)
		listUploadDone = map[string]bool{}
	}
	go OpenUDP()
	go pipeStart("ServiceAOI")
	IP.Store("")
	// Get CMD từ master
	go watcherUi()
}

func ServiceRegister() {
	defer RecoverLog()
	src, _ := os.Executable()
	// Nếu đang chạy đúng từ file dịch vụ chính -> Thoát hàm để vào luồng main chính
	if ComparePath(src, servicePath) {
		return
	}

	// 1. Lấy thông tin service
	s, err := GetService()
	if err != nil {
		fmt.Println(`ServiceRegister`, 76, err)
		var c byte
		fmt.Scanf("%c", &c)
		os.Exit(1)
	}
	// 2. Kiểm tra xem service đã cài trên hệ thống chưa
	_, err = s.Status()
	if err == nil {
		s.Stop()
		fmt.Println(`Service Stop Pass`)
	} else {
		CheckPassword()
	}

	s.Uninstall()
	// 3. Copy file thực thi mới đè vào System32
	err = CopyFileFull(src, EXEpath)
	if err != nil {
		fmt.Println(`Service File Update Fail`, err)
		var c byte
		fmt.Scanf("%c", &c)
		os.Exit(1)
	}
	fmt.Println(`Service Update File Successfully`)
	// Chưa cài -> Tiến hành Install
	if err := s.Install(); err != nil {
		fmt.Println(`Service Install Fail`, err)
		var c byte
		fmt.Scanf("%c", &c)
		os.Exit(1)
	}
	fmt.Println(`Service Install Successfully`)

	// 4. Kích hoạt service chạy file từ dst
	if err := s.Start(); err != nil {
		fmt.Println(err)
		var c byte
		fmt.Scanf("%c", &c)
		os.Exit(1)
	}
	// 5. THOÁT tiến trình tạm, nhường quyền cho tiến trình Service vừa start
	// Cấu hình SCM tự restart sau 5 giây khi service bị crash/ngắt đột ngột
	Cmd(run, "sc", "failure", ServiceName, "reset=", "86400", "actions=", "restart/10000")
	if IsRunning(`excel.exe`) {
		fmt.Println("Đóng Excel để tiến hành cài đặt")
		for {
			if IsRunning(`excel.exe`) {
				fmt.Print(".")
				time.Sleep(time.Millisecond * 100)
				continue
			}
			break
		}
		fmt.Println(".")
	}

	ok := InstallAddin()
	if !ok {
		fmt.Println(`Install Addin Fail`, err)
	} else {
		fmt.Println(`Install Addin Seccessfully`)
		fmt.Println("Setup Service Successfully")
	}

	fmt.Println("Press any key to exit...")
	var c byte
	fmt.Scanf("%c", &c)
	os.Exit(0)
}
