package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var Loggers *log.Logger

func InitLog() *os.File {
	os.MkdirAll(`D:\log\Service`, os.ModePerm)

	today := time.Now().Format("2006-01-02")
	logpath := fmt.Sprintf(`D:\log\Service\%s.log`, today)

	f, err := os.OpenFile(
		logpath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)

	if err != nil {
		Loggers = log.New(
			os.Stdout,
			"",
			log.Ldate|log.Ltime,
		)
		return nil
	}

	Loggers = log.New(
		f,
		"",
		log.Ldate|log.Ltime,
	)
	return f
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
