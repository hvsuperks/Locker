package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func uploadAOI() {
	for {
		chetaoPath := chetao.Load().(string)
		ip := IP.Load().(string)
		if len(listUploadWait) == 0 {
			sendUI <- sendT{Wait: `0`}
			return
		} else {
			val := strconv.Itoa(len(listUploadWait))
			LogUnique("listUploadWait", val)
			sendUI <- sendT{Wait: val}
		}
		if chetaoPath == "" {
			sendUI <- sendT{Room: `Fail`, Model: `Fail`, Part: `Fail`}
		}
		buff := 0
		for path := range listUploadWait {
			if uploadWriter(ip, path) {
				LogUnique("listUploadWait delete", path)
				delete(listUploadWait, path)
				listUploadDone[path] = true
				buff += 1
			}

			if buff >= 10 {
				SaveJson(uploadJsonPath, listUploadDone)
				val := strconv.Itoa(len(listUploadWait))
				sendUI <- sendT{Wait: val}
				buff = 0
			}
		}
		val := strconv.Itoa(len(listUploadWait))
		sendUI <- sendT{Wait: val}
		buff = 0
		SaveJson("upload.json", listUploadDone)
		return
	}
}

func uploadWriter(ip, paths string) bool {
	rel, _ := filepath.Rel(syncFolder, filepath.Dir(paths))
	ct := chetao.Load().(string)
	if ct == "" {
		return false
	}
	path := filepath.Join(ct, rel)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, err := os.Open(paths)
	if err != nil {
		LogUnique("Open Fail", rel, err)
		return false
	}
	defer file.Close()

	part, err := writer.CreateFormFile("image", filepath.Base(paths))
	if err != nil {
		LogUnique("Writer Fail", rel, err)
		LogUnique(err)
		return false
	}

	io.Copy(part, file)
	writer.WriteField("path", path)
	writer.Close()

	req, _ := http.NewRequest("POST", "http://"+ip+":"+MasterPort+"/uploadAOI", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		LogUnique("Upload Fail", rel, err)
		LogUnique(err)
		return false
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		LogUnique("Upload OK", respBody)
		return true
	}
	LogUnique("Fail:", string(respBody))
	return false

}
