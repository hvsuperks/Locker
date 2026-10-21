package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

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

func checkSignApp() windows.Handle {
	name := windows.StringToUTF16Ptr("Global\\AOI")
	mutex, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		os.Exit(0)
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		os.Exit(0)
	}
	return mutex
}

func OpenUDP() error {
	// 1. Check rule tồn tại chưa
	checkCmd := exec.Command(
		"netsh",
		"advfirewall",
		"firewall",
		"show",
		"rule",
		"name=AOI",
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
		"name=AOI",
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

func OpenApp(ExePath string) error {
	defer func() {
		if r := recover(); r != nil {
			LogUnique(fmt.Sprintf("panic: %v", r))
			time.Sleep(time.Second)
		}
	}()
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

func RemoveEmptyDirs(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Chỉ xử lý folder
		if !info.IsDir() {
			return nil
		}

		// Không xóa root
		if path == root {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		if len(entries) == 0 {
			_ = os.Remove(path)
		}

		return nil
	})
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

func RegWrite(path string, name string, value string) bool {

	k, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		path,
		registry.SET_VALUE,
	)

	if err != nil {
		fmt.Println(err)
		return false
	}
	defer k.Close()

	err = k.SetStringValue(name, value)
	fmt.Println(err)
	return err == nil
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
