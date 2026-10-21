package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func UpdateService(newVer string) {
	url := fmt.Sprintf(`http://%s:%s/AOI_%s`, IP.Load().(string), MasterPort, newVer)
	if err := DownloadFile(url, serviceUpdatePath); err != nil {
		LogUnique(`Download`, url, err)
		return
	}
	exec.Command(serviceUpdatePath).Start()
	os.Exit(0)
}

func UpdateUi(newVer string) {
	url := fmt.Sprintf(`http://%s:%s/UiAoi_%s`, IP.Load().(string), MasterPort, newVer)
	if err := DownloadFile(url, serviceUpdatePath); err != nil {
		LogUnique(`Download`, url, err)
		return
	}
	exec.Command(`taskkill`, `/f`, `/im`, `Ui.exe`).Start()
	time.Sleep(time.Second * 3)
	os.Rename(uiUpdatePath, uiPath)
	exec.Command(uiPath).Start()
	LogUnique(`Update Pass: UiAoi_` + newVer)
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
