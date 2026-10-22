package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
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
					LogInfo(fmt.Sprintf("panic: %v", r))
					time.Sleep(time.Second)
				}
			}()
			listener, err := winio.ListenPipe(`\\.\pipe\`+appname, nil)
			if err != nil {
				LogInfo("ERROR: Pipe", `\\.\pipe\`+appname, err)
				return
			}
			LogInfo("pipe Start")
			for {
				conn, err := listener.Accept()
				if err != nil {
					LogInfo("ERROR: Pipe", `\\.\pipe\`+appname, err)
					return
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

type colorStruct struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}
type uiChanStruct struct {
	Value string      `json:"value"`
	State colorStruct `json:"state"`
	Key   string      `json:"key"`
}

func color_style(state string) colorStruct {
	switch strings.ToUpper(state) {
	case "OK":
		return colorStruct{R: 0, G: 0, B: 40}
	case "FAIL":
		return colorStruct{R: 150, G: 0, B: 0}
	case "WIFI":
		return colorStruct{R: 0, G: 90, B: 0}
	}
	return colorStruct{R: 50, G: 50, B: 50}
}

var isShutdownLocker = atomic.Bool{}
var IDcurrent = atomic.Value{}

func handleConn(conn net.Conn, ver, last *atomic.Value, appname string) {
	defer func() {
		if r := recover(); r != nil {
			LogInfo(fmt.Sprintf("panic: %v", r))
		}
	}()
	defer conn.Close()
	buf := make([]byte, 4096)
	msg := []byte(thisVer)
	go func() {
		defer conn.Close()
		defer RecoverLog()
		for {
			i := msg
			if "Service_locker" == appname && isShutdownLocker.Load() {
				i = []byte("shutdown")
				isShutdownLocker.Store(false)
			}
			if appname == "Service_ui" {
				idcor := color_style("OK")
				if f, err := os.Stat(HostpotPath); err == nil && !f.IsDir() {
					if WifiHostPotState.Load() {
						idcor = color_style("WIFI")
					} else {
						idcor = color_style("FAIL")
					}
				}
				locor := color_style("OK")
				if lockerUpdateState.Load() {
					locor = color_style("FAIL")
				}

				uicor := color_style("OK")
				if uiUpdateState.Load() {
					uicor = color_style(`FAIL`)
				}
				secor := color_style("OK")
				muUDPrepeatMSG.RLock()
				if thisVer != UDPrepeatMSG.AutoUpdate {
					secor = color_style("FAIL")
				}
				muUDPrepeatMSG.RUnlock()
				m := []uiChanStruct{
					{Key: "ID", Value: "ID: " + IDcurrent.Load().(string), State: idcor},
					{Key: "VERSERVICE", Value: "SC: " + thisVer, State: secor},
					{Key: "VERSION", Value: "GO: " + AppLocker.Ver.Load().(string), State: locor},
					{Key: "UIVERSION", Value: "Ui: " + AppUi.Ver.Load().(string), State: uicor},
				}
				i, _ = json.Marshal(m)
			}
			_, err := conn.Write(i)
			if err != nil {
				LogInfo(appname, `conn.write`, err)
				return
			}
			time.Sleep(time.Second)
		}
	}()
	for {
		n, err := conn.Read(buf)
		if err != nil {
			LogInfo(appname, `conn.SetReadDeadline`, err)
			return
		}
		ver.Store(string(buf[:n]))
		last.Store(time.Now())
	}
}
