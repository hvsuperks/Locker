package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

var Loggers *log.Logger

func InitLog() {
	var folder = `D:\log\Service`
	DayNow := time.Now().Day()
	count := 1
	var currentFile *os.File
	const maxSize = 1 * 1024 * 1024 // 5MB
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
				Loggers = l
				if old != nil {
					old.Close()
				}
				DayNow = time.Now().Day()
				keepLastLogs(folder, 5)
			}
		}
		time.Sleep(time.Second * 10)
	}
}

var (
	recentLogs []string
	logMu      sync.Mutex
)

func LogUnique(args ...any) {

	_, file, line, _ := runtime.Caller(1)
	msg := fmt.Sprintf(
		"[%s:%d] %s",
		filepath.Base(file),
		line,
		fmt.Sprint(args...),
	)
	logMu.Lock()
	defer logMu.Unlock()
	for _, v := range recentLogs {
		if v == msg {
			return
		}
	}
	recentLogs = append(recentLogs, msg)
	if len(recentLogs) > 15 {
		recentLogs = recentLogs[1:]
	}
	Loggers.Println(msg)
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

func keepLastLogs(folder string, maxFiles int) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return
	}

	type fileInfo struct {
		Name    string
		ModTime time.Time
	}

	var files []fileInfo

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".log" {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		files = append(files, fileInfo{
			Name:    filepath.Join(folder, e.Name()),
			ModTime: info.ModTime(),
		})
	}

	if len(files) <= maxFiles {
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})

	for _, f := range files[maxFiles:] {
		os.Remove(f.Name)
	}
}
