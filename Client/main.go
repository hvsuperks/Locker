package main

import (
	"Locker/config"
	"Locker/script"
	"context"
	_ "embed"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	_ "net/http/pprof"

	"github.com/kardianos/service"
	"golang.org/x/sys/windows"
)

func main() {
	if _, err := os.Stat("C:\\programdata\\locker\\Fail"); err == nil {
		return
	}
	isdownload.Store(false)
	isCurrentCRC.Store("None")
	VerAutoUpdate.Store("")
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	config.IsDebug = true
	config.IP.Store("")
	// Khởi tạo context Close App
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go AutoUpdate(ctx)
	CurrentAppRootPath, _ := os.Executable()
	CurrentAppRootDir := filepath.Dir(CurrentAppRootPath)
	// Đường dẫn tiêu chuẩn
	// Nếu đường dẫn app đang ở ko phải ở thư mục quy định sẽ copy vào khu vực quy định và stopsc
	Cmd(start, "taskkill", "/IM", "PGM_lock.exe", "/F")
	Cmd(start, "taskkill", "/IM", "app_loop.exe", "/F")
	Cmd(start, "taskkill", "/IM", "watcherdog.exe", "/F")
	time.Sleep(time.Second)
	Cmd(start, "schtasks", "/delete", "/tn", config.DstAppName, "/f")
	Cmd(start, "schtasks", "/delete", "/tn", config.DstAppName+"_", "/f")
	CheckModeApp(CurrentAppRootDir, CurrentAppRootPath)
	//Check App có đang mở hay ko, nếu đang mở thì out luôn
	mutex := check_sign_app()
	_ = mutex
	ClearOldApp()
	InitLog(`D:\log\Locker\connect`, &Logger.connect)
	InitLog(`D:\log\Locker\CRC`, &Logger.crc)
	InitLog(`D:\log\Locker\PGM`, &Logger.pgm)
	InitLog(`D:\log\Locker\Merge`, &Logger.merge)
	InitLog(`D:\log\Locker\Manager`, &Logger.manager)
	InitLog(`D:\log\Locker\debug`, &Logger.debug)
	go PipeConnect()
	go StartPipeServer()
	// Mở Port UDP
	go OpenUDP10001()

	// Đọc ID và phân loại máy
	ID, model = ID_Read()
	if len(ID) != 9 {
		LogInfo(&Logger.manager, "INFO:", "ID Fail", ID)
	}
	STATUS.Data.VerLocker = Ver
	STATUS.Data.ID = ID
	STATUS.Data.Model = model
	STATUS.Data.MesID = script.RegRead("SOFTWARE\\CubeX\\MES_SET", "machine_cd")

	// ListenMasterUDP
	go ListenMasterUDP(Log)
	go Start(ctx, cancel)
	go sendCSV(ctx)
	go PingPong(ctx, cancel)
	go SendStatusFunc(ctx)
	go copyTMPtoMes(ctx)
	select {}
}

func cleanupAndExit() {
	wdChan := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		close(wdChan)
	}()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-wdChan:
	}
	os.Exit(0)
}

func check_sign_app() windows.Handle {
	name := windows.StringToUTF16Ptr("Global\\Locker")
	mutex, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		os.Exit(0)
	}
	// ERROR_ALREADY_EXISTS = 183
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		os.Exit(0)
	}
	return mutex
}

func ID_Read() (string, string) {
	path := "C:\\Program Files\\uvnc bvba\\UltraVNC\\ultravnc.ini"
	data, err := os.ReadFile(path)
	if err != nil {
		panic("Chưa cài VNC à óc chó?")
	}
	text := string(data)
	re := regexp.MustCompile(`ID:(\d{9})(\D|$)`)
	match := re.FindStringSubmatch(text)
	if len(match) != 3 {
		return "", ""
	}
	ID := match[1]
	Model := ID[:6]
	return ID, Model

}

func ServiceUpdateGet() bool {
	cfg := &service.Config{
		Name: "NTanh",
	}
	s, err := service.New(nil, cfg)
	if err != nil {
		return false
	}
	st, err := s.Status()
	if err != nil {
		return false
	}
	return st == service.StatusRunning
}
