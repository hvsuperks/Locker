package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Tạo Log File theo ngày
func Create_logger(LogPath string, newFileMode *bool) {
	day := 0
	var ctxPrint context.Context
	var cancelPrint context.CancelFunc
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if time.Now().Day() != day {
				day = time.Now().Day()
				if cancelPrint != nil {
					cancelPrint()
				}
				ctxPrint, cancelPrint = context.WithCancel(ctx)
				defer cancelPrint()
				go println(ctxPrint, OpenDailyLog(LogPath, newFileMode))
			}
		}
	}
}

func println(ctxPrint context.Context, currentFile []*os.File) {
	defer currentFile[0].Close()
	defer currentFile[1].Close()
	defer currentFile[2].Close()
	Log_write_Client := log.New(currentFile[0], "", log.Ldate|log.Ltime)
	Log_write_Aoi := log.New(currentFile[1], "", log.Ldate|log.Ltime)
	Log_write_Master := log.New(currentFile[2], "", log.Ldate|log.Ltime)

	for {
		select {
		case <-ctxPrint.Done():
			return
		case text := <-Log:
			switch text.Type {
			case fmtClient:
				Log_write_Client.Println(text.Data...)
			case fmtAOI:
				Log_write_Aoi.Println(text.Data...)
			case fmtMaster:
				Log_write_Master.Println(text.Data...)
			}

		}
	}
}

func OpenDailyLog(LOG_PATH string, newFileMode *bool) []*os.File {
	today := time.Now().Format("2006-01-02")
	filedir := filepath.Join(LOG_PATH, today)
	if *newFileMode {
		*newFileMode = false
		os.Remove(filedir)
	}
	os.MkdirAll(filedir, 0755)
	var f []*os.File
	var i = []string{"fmtClient.log", "fmtAOI.log", "fmtMaster.log"}
	for _, n := range i {
		filename := filepath.Join(filedir, n)
		file, err := os.OpenFile(
			filename,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_SYNC,
			0644,
		)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		// 2. Kiểm tra nếu file mới tinh (size = 0), hãy ghi 3 byte BOM UTF-8
		info, _ := file.Stat()
		if info.Size() == 0 {
			// 3 byte này (EF BB BF) giúp Windows hiểu đây là file Unicode UTF-8
			file.Write([]byte{0xEF, 0xBB, 0xBF})
		}
		f = append(f, file)
	}
	return f
}
