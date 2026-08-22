package main

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/Microsoft/go-winio"
)

var lastErr string

func pipeStart(appname string, ver, last *atomic.Value) {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					LogUnique(fmt.Sprintf("panic: %v", r))
					time.Sleep(time.Second)
				}
			}()
			listener, err := winio.ListenPipe(`\\.\pipe\`+appname, nil)
			if err != nil {
				LogUnique("ERROR: Pipe", `\\.\pipe\`+appname, err)
				return
			}
			for {
				conn, err := listener.Accept()
				if err != nil {
					continue
				}
				go handleConn(conn, ver, last, appname)

			}
		}()
		time.Sleep(time.Second)
	}

}

type pipeStruct struct {
	App string `json:"app"`
	Ver string `json:"ver"`
}

type AppStruct struct {
	ExePath    string
	UpdatePath string
	Ver        atomic.Value
	Last       atomic.Value
	UpdateChan chan string
}

var AppUi = &AppStruct{
	ExePath:    `C:\Programdata\Locker\Ui.exe`,
	UpdatePath: `C:\Programdata\Locker\UpdateUi.exe`,
	Ver:        atomic.Value{},
	Last:       atomic.Value{},
	UpdateChan: make(chan string, 30),
}

var AppLocker = &AppStruct{
	ExePath:    `C:\Programdata\Locker\Locker.exe`,
	UpdatePath: `C:\Programdata\Locker\UpdateLocker.exe`,
	Ver:        atomic.Value{},
	Last:       atomic.Value{},
	UpdateChan: make(chan string, 30),
}
var isShutdownLocker = atomic.Bool{}

func handleConn(conn net.Conn, ver, last *atomic.Value, appname string) {
	defer func() {
		if r := recover(); r != nil {
			LogUnique(fmt.Sprintf("panic: %v", r))
		}
	}()
	defer conn.Close()
	buf := make([]byte, 4096)
	msg := []byte(thisVer)
	go func() {
		defer conn.Close()
		defer func() {
			if r := recover(); r != nil {
				LogUnique(fmt.Sprintf("panic: %v", r))
				time.Sleep(time.Second)
			}
		}()
		for {
			conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
			i := msg

			if "Service_locker" == appname && isShutdownLocker.Load() {
				i = []byte("shutdown")
				isShutdownLocker.Store(false)
			}
			_, err := conn.Write(i)
			if err != nil {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	for {
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		ver.Store(string(buf[:n]))
		last.Store(time.Now())
	}
}
