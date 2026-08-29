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
	count := 1
	var currentFile *os.File
	go func() {
		const maxSize = 5 * 1024 * 1024 // 5MB
		os.MkdirAll(folder, os.ModePerm)
		logpath := filepath.Join(folder, fmt.Sprintf("%s_%d.log", time.Now().Format("2006-01-02"), count))
		for {
			info, err := os.Stat(logpath)
			if currentFile == nil || (err == nil && info.Size() >= maxSize) || DayNow != time.Now().Day() {
				if DayNow != time.Now().Day() {
					count = 1
				}
				for {
					logpath = filepath.Join(folder,
						fmt.Sprintf("%s_%d.log", time.Now().Format("2006-01-02"), count))
					info, err := os.Stat(logpath)
					if os.IsNotExist(err) {
						break
					}
					if info.Size() < maxSize {
						break
					}
					count++
				}
				f, l, err := openLog(logpath)
				if err == nil {
					old := currentFile
					currentFile = f
					p.Store(l)
					if old != nil {
						old.Close()
					}
					DayNow = time.Now().Day()
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
		fmt.Sprintln(v...),
	)
	l := p.Load()
	if l == nil {
		return
	}
	logMu.Lock()
	defer logMu.Unlock()
	logs := recentLogs[l]
	if p == &Logger.manager {
		l.Println(msg)
		return
	}
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

func openLog(logPath string) (*os.File, *log.Logger, error) {
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
