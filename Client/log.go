package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type LogWrap struct {
	connect atomic.Pointer[log.Logger]
	crc     atomic.Pointer[log.Logger]
	pgm     atomic.Pointer[log.Logger]
	merge   atomic.Pointer[log.Logger]
	manager atomic.Pointer[log.Logger]
}

var Logger LogWrap

func InitLog(folder string, p *atomic.Pointer[log.Logger]) {
	DayNow := time.Now().Day()
	go func() {
		const maxSize = 5 * 1024 * 1024 // 5MB
		os.MkdirAll(folder, os.ModePerm)
		logPath := filepath.Join(folder, "app.log")
		openLog := func() (*os.File, *log.Logger, error) {
			f, err := os.OpenFile(
				logPath,
				os.O_CREATE|os.O_APPEND|os.O_WRONLY,
				0666,
			)
			if err != nil {
				return nil, nil, err
			}

			l := log.New(
				f,
				"",
				log.LstdFlags|log.Lshortfile,
			)

			return f, l, nil
		}

		currentFile, logger, err := openLog()
		if err != nil {
			return
		}

		p.Store(logger)

		for {
			info, err := currentFile.Stat()
			if err == nil && info.Size() >= maxSize || DayNow != time.Now().Day() {
				DayNow = time.Now().Day()
				currentFile.Close()

				newName := filepath.Join(
					folder,
					fmt.Sprintf(
						"app_%s.log",
						time.Now().Format("20060102_150405"),
					),
				)

				_ = os.Rename(logPath, newName)

				f, l, err := openLog()
				if err == nil {
					currentFile = f
					p.Store(l)
				}
			}

			time.Sleep(time.Second * 10)
		}
	}()
}

var logMu sync.Mutex
var recentLogs = make(map[*log.Logger][]string)

func LogInfo(p *atomic.Pointer[log.Logger], v ...any) {
	_, file, line, _ := runtime.Caller(1)
	msg := fmt.Sprintf(
		"[%s:%d] %s",
		filepath.Base(file),
		line,
		fmt.Sprint(v...),
	)
	l := p.Load()
	if l == nil {
		return
	}
	logMu.Lock()
	defer logMu.Unlock()
	logs := recentLogs[l]
	if slices.Contains(logs, msg) {
		return
	}
	logs = append(logs, msg)
	if len(logs) > 50 {
		logs = logs[1:]
	}
	recentLogs[l] = logs

	l.Println(msg)
}
