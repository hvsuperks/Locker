package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

var Logger *log.Logger

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
		Logger = log.New(
			os.Stdout,
			"",
			log.Ldate|log.Ltime|log.Lshortfile,
		)
		return nil
	}

	Logger = log.New(
		f,
		"",
		log.Ldate|log.Ltime|log.Lshortfile,
	)
	return f
}
