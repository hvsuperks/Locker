package script

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func DownloadFile(url, path string) error {
	//fmt.Println("Download :" + url)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := httpClient.Get(url)
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
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
		time.Sleep(50 * time.Millisecond)
		os.Remove(path + ".temp")
	}()

	_, err = io.Copy(file, resp.Body)
	//fmt.Println("Download", err)
	if err != nil {
		return err
	}
	file.Close()
	err = os.Rename(path+".temp", path)
	return err
}
