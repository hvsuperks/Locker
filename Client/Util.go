package main

import (
	"Locker/config"
	"Locker/script"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

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

func ClearOldApp() {
	// Xóa Task
	config.IsDog.Store(true)
	Cmd(run, "taskkill", "/IM", "watcherdog.exe", "/F")
	if _, err := os.Stat(`C:\programdata\locker\Autoupdate.exe`); err == nil {
		Cmd(run, "taskkill", "/IM", "Autoupdate.exe", "/F")
		os.Remove(`C:\programdata\locker\Autoupdate.exe`)
	}
	time.Sleep(time.Second)
	Cmd(start, "schtasks", "/delete", "/tn", config.DstAppName, "/f")
	Cmd(start, "schtasks", "/delete", "/tn", config.DstAppName+"_", "/f")
	// Xóa Start Up
	homeDir, _ := os.UserHomeDir()
	startupDir := filepath.Join(homeDir, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	// Tên file shortcut (ví dụ: MyProgram.lnk)
	shortcutPath := filepath.Join(startupDir, "Locker.lnk")
	files, _ := os.ReadDir(startupDir)
	os.Remove(shortcutPath)
	for _, file := range files {
		if file.IsDir() {
			os.RemoveAll(file.Name())
		} else if !strings.EqualFold(filepath.Ext(file.Name()), ".lnk") || filepath.Ext(file.Name()) == ".vbs" {
			os.Remove(file.Name())
		}
	}
}

func CheckModeApp(CurrentAppRootDir, CurrentAppRootPath string) {
	if !script.ComparePath(CurrentAppRootDir, config.AppRootDir) {
		for _, i := range []string{"PGM_lock.exe", "watcherdog.exe", "app_loop.exe"} {
			Cmd(start, "taskkill", "/IM", i, "/T", "/F")
		}
		src := filepath.Join(config.AppRootDir, "UpdateLocker.exe")
		dst := filepath.Join(config.AppRootDir, "Locker.exe")
		err := script.CopyFileFull(CurrentAppRootPath, src)
		if err != nil {
			msgbox("Update Locker Fail " + err.Error())
			os.Exit(1)
		}
		if ServiceUpdateGet() {
			os.Exit(0)
		}
		script.Cmd(run, "taskkill", "/F", "/IM", "Locker.exe")
		var er error
		for range 5 {
			os.Remove(dst)
			er = os.Rename(src, dst)
			if er == nil {
				script.Cmd(run, dst)
				msgbox("Update Locker Pass")
				os.Exit(0)
			}
			fmt.Println("rename", src, dst, er)
			time.Sleep(500 * time.Millisecond)
		}
		msgbox("Update Locker Fail " + er.Error())
		os.Exit(1)
	}
}
