package main

import (
	"encoding/json"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Microsoft/go-winio"
)

var isUiOnline = atomic.Bool{}

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
type UiReciver struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

func StartPipeServer() {
	for {
		pipe, err := winio.ListenPipe(`\\.\pipe\Locker`, nil)
		if err != nil {
			LogInfo(&Logger.debug, `Pipe \\.\pipe\Locker ListenPipe`, err)
			time.Sleep(time.Second)
			continue
		}
		for {
			conn, err := pipe.Accept()
			if err != nil {
				LogInfo(&Logger.debug, `Pipe \\.\pipe\Locker Accept`, err)
				break
			}
			go pipeHandler(conn)
		}
	}
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
func pipeHandler(pipe net.Conn) {
	defer pipe.Close()
	isUiOnline.Store(true)
	isVer := ""
	go func() {
		defer isUiOnline.Store(false)
		enc := json.NewEncoder(pipe)
		for {
			connect := "Disconnected"
			if isDisconnect.Load() {
				connect = "Connecting"
			}
			tmpstatus := STATUS_GET(&STATUS).(STATUS_struct)
			urlHome := httpDownload.Load().(string)
			urlUpload := httpUpload.Load().(string)
			con := "OK"
			if !isDisconnect.Load() {
				con = "FAIL"
			}
			data := map[string][]uiChanStruct{
				"uichange": {
					{Key: "CRC", Value: "CRC: " + tmpstatus.MasterMap.CRC.CRC, State: color_style(tmpstatus.MasterMap.CRC.Color)},
					{Key: "PGM", Value: "PGM: " + tmpstatus.MasterMap.PGM.PGM, State: color_style(tmpstatus.MasterMap.PGM.Color)},
					{Key: "CRC_LOCKER", Value: "CRC Locker: " + STATUS.Locker_CRC.Value, State: color_style(STATUS.Locker_CRC.Color)},
					{Key: "PGM_LOCKER", Value: "PGM Locker: " + STATUS.Locker_PGM.Value, State: color_style(STATUS.Locker_PGM.Color)},
					{Key: "CONNECT", Value: connect, State: color_style(con)},
					{Key: "MERGE", Value: STATUS.Merge.Value, State: color_style(STATUS.Merge.Color)},
				},
				"url": {
					{Key: "home", Value: urlHome},
					{Key: "upload", Value: urlUpload},
				},
			}
			pipe.SetWriteDeadline(time.Now().Add(3 * time.Second))
			if err := enc.Encode(data); err != nil {
				LogInfo(&Logger.debug, "Locker send Fail", err)
				pipe.Close()
				return
			}
			time.Sleep(time.Millisecond * 500)
		}
	}()
	dec := json.NewDecoder(pipe)
	for {
		var msg UiReciver
		pipe.SetReadDeadline(time.Now().Add(3 * time.Second))
		if err := dec.Decode(&msg); err != nil {
			LogInfo(&Logger.debug, "Locker Decode Fail", err)
			return
		}
		dispacht(msg, &isVer)
	}
}

var isCurrentCRC = atomic.Value{}

func dispacht(msg UiReciver, isVer *string) {
	switch msg.Type {
	case "click":
		switch msg.Key {
		case "id":
			if f, err := os.Stat(HostpotPath); err == nil && !f.IsDir() {
				os.Remove(HostpotPath)
			} else {
				os.WriteFile(HostpotPath, []byte{}, 0644)
			}
		case "pgm":
		case "crc":
		case "pgm_locker":
		case "crc_locker":
			i := STATUS_GET(&STATUS.Locker_CRC.Value).(string)
			if i == "Unlock" {
				handlelocker_state("lockercrc", Command{Action: "Locked"})
			} else {
				handlelocker_state("lockercrc", Command{Action: "Unlock"})
			}

		case "merge":
		case "connect":
		case "msg":
		}

	case "crc":
		if msg.Key != isCurrentCRC.Load().(string) {
			isCurrentCRC.Store(msg.Key)
			STATUS_PUT(&STATUS.CurrentCRC, strings.ToUpper(msg.Key))
			SendStatus <- struct{}{}
		}
	case "ver":
		if isVer != &msg.Key {
			*isVer = msg.Key
			STATUS_PUT(&STATUS.VerUiLocker, *isVer)
		}
	}
}

func PipeConnect() error {
	timeout := 3 * time.Second
	for {
		conn, err := winio.DialPipe(`\\.\pipe\Service_locker`, &timeout)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			LogInfo(&Logger.debug, err)
			continue
		}
		AutoUpdateHandler(conn)
		ServiceStatus.Store(false)
		ServiceConnectTime.Store(time.Now())
	}
}

var ServiceStatus = atomic.Bool{}
var ServiceConnectTime = atomic.Value{}

func AutoUpdateHandler(conn net.Conn) {
	defer conn.Close()
	ServiceStatus.Store(true)
	go func() {
		i := []byte(Ver)
		for {
			_, err := conn.Write(i)
			if err != nil {
				LogInfo(&Logger.debug, err)
				conn.Close()
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			LogInfo(&Logger.debug, err)
			return
		}
		i := string(buf[:n])
		LogInfo(&Logger.debug, i)
		if i == "shutdown" {
			LogInfo(&Logger.manager, "shutdown")
			go cleanupAndExit()
		} else if VerAutoUpdate.Load() != i {
			VerAutoUpdate.Store(i)

			STATUS_PUT(&STATUS.VerAutoUpdate, i)
		}
	}
}
