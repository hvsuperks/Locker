package main

import (
	"encoding/json"
	"net"
	"sync/atomic"
	"time"

	"github.com/Microsoft/go-winio"
)

var isUiOnline = atomic.Bool{}

type uiChanStruct struct {
	Value string `json:"value"`
	State bool   `json:"state"`
	Key   string `json:"key"`
}
type UiReciver struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

func StartPipeServer() {
	for {
		pipe, err := winio.ListenPipe(`\\.\pipe\Locker`, nil)
		if err != nil {
			LogInfo(&Logger.connect, "ERRR:", `Pipe \\.\pipe\Locker ListenPipe`, err)
			time.Sleep(time.Second)
			continue
		}
		for {
			conn, err := pipe.Accept()
			if err != nil {
				LogInfo(&Logger.connect, "ERRR:", `Pipe \\.\pipe\Locker Accept`, err)
				break
			}
			go pipeHandler(conn)
		}
	}
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
			STATUS.mu.RLock()
			urlHome := httpDownload.Load().(string)
			urlUpload := httpUpload.Load().(string)
			data := map[string][]uiChanStruct{
				"uichange": {
					{Key: "ID", Value: "ID: " + ID, State: true},
					{Key: "CRC", Value: "CRC: " + STATUS.Data.MasterMap.CRC.CRC, State: STATUS.Data.MasterMap.CRC.Color == "OK"},
					{Key: "PGM", Value: "PGM: " + STATUS.Data.MasterMap.PGM.PGM, State: STATUS.Data.MasterMap.PGM.Color == "OK"},
					{Key: "CRC_LOCKER", Value: "CRC Locker: " + STATUS.Data.Locker_CRC.Value, State: STATUS.Data.Locker_CRC.Color == "OK"},
					{Key: "PGM_LOCKER", Value: "PGM Locker: " + STATUS.Data.Locker_PGM.Value, State: STATUS.Data.Locker_PGM.Color == "OK"},
					{Key: "CONNECT", Value: connect, State: isDisconnect.Load()},
					{Key: "VERSION", Value: "Go:" + Ver, State: true},
					{Key: "MERGE", Value: STATUS.Data.Merge.Value, State: STATUS.Data.Merge.Color == "OK"},
				},
				"url": {
					{Key: "home", Value: urlHome},
					{Key: "upload", Value: urlUpload},
				},
			}
			STATUS.mu.RUnlock()
			pipe.SetWriteDeadline(time.Now().Add(3 * time.Second))
			if err := enc.Encode(data); err != nil {
				LogInfo(&Logger.connect, "ERROR:", "Locker send Fail", err)
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
			LogInfo(&Logger.connect, "ERROR:", "Locker Decode Fail", err)
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
		case "ID":
		case "pgm":
		case "crc":
		case "pgm_locker":
		case "crc_locker":
			STATUS.mu.RLock()
			i := STATUS.Data.Locker_CRC.Value
			STATUS.mu.RUnlock()
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
			SendStatus <- struct{}{}
		}
	case "ver":
		if isVer != &msg.Key {
			*isVer = msg.Key
			LockMap(&STATUS.mu)
			STATUS.Data.VerUiLocker = *isVer
			UnlockMap(&STATUS.mu)
		}
	}
}

func PipeConnect() error {
	timeout := 3 * time.Second
	LogInfo(&Logger.debug, `Start`)
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
			LockMap(&STATUS.mu)
			STATUS.Data.VerAutoUpdate = i
			UnlockMap(&STATUS.mu)
		}
	}
}
