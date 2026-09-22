package main

import (
	"Locker/config"
	"Locker/script"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

func sendCSV(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			LogInfo(&Logger.merge, "ERROR:", fmt.Sprintf("panic: %v\n%s", err, debug.Stack()))
			msgbox("Send CSV Fail")
			os.Exit(0)
		}
	}()
	isDone := false
	for {
		machine_cd := script.RegRead(`SOFTWARE\CubeX\MES_SET`, `machine_cd`)
		time.Sleep(10 * time.Second)
		if isDone {
			return
		}
		filepath.WalkDir(config.UploadRoot, func(path string, d fs.DirEntry, err error) error {
			select {
			case <-ctx.Done():
				isDone = true
				return filepath.SkipAll
			default:
				time.Sleep(100 * time.Millisecond)
			}
			if err != nil {
				return err
			}
			// Kiểm tra xem là file hay thư mục
			if d.IsDir() || filepath.Ext(path) != ".csv" {
				return nil // Tiếp tục duyệt
			} else {
				filename := d.Name()
				namesplit := strings.Split(strings.Split(filename, ".")[0], "_")
				if len(namesplit) == 2 {
					n := fmt.Sprintf("%s_%s_%s.csv", namesplit[0], namesplit[1][:3], time.Now().Format("20060102150405"))
					os.Rename(path, filepath.Join(filepath.Dir(path), n))
					return nil
				} else if len(namesplit) != 3 {
					return nil
				}
				lotFull := namesplit[0]
				prefix := lotFull[:len(lotFull)-5]
				date := prefix[len(prefix)-6:]
				md := lotFull[:len(lotFull)-11]
				cd := config.TypeMap[ID[6:7]]
				suffix := lotFull[len(lotFull)-2:]
				ver := namesplit[2]
				lot := prefix + "_" + cd + "_" + suffix

				if strings.EqualFold(model, "unknown") {
					md = model
					strings.ReplaceAll(lot, "Unknown", model)
					strings.ReplaceAll(lot, "unknown", model)
				}
				if err := uploadCSV(path, md, cd, lot, date, machine_cd, ver); err == nil {
					os.Remove(path)
					LogInfo(&Logger.merge, "INFO:", "upload OK", path)
				} else {
					LogInfo(&Logger.merge, "ERROR: Upload Fail", err)
				}
			}
			return nil
		})
	}
}

func uploadCSV(path, md, congdoan, lot, date, machine_cd, ver string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	writer.WriteField("model", md)
	writer.WriteField("congdoan", congdoan)
	writer.WriteField("lot", lot)
	writer.WriteField("date", date)
	writer.WriteField("id", machine_cd)
	writer.WriteField("ver", ver)
	writer.Close()
	ip, _ := config.IP.Load().(string)
	req, err := http.NewRequest("POST", "http://"+ip+":"+config.MasterPort+"/api/RawDataPost", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("upload fail: %s", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload fail: %s", string(respBody))
	}

	if string(respBody) != "ok" {
		return fmt.Errorf("upload fail: %s", string(respBody))
	}
	return nil
}
