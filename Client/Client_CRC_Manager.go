package main

import (
	"Locker/config"
	"Locker/script"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func (a mClient) crcManager(crcRunning *atomic.Bool, LockerChan, Restart_XOIS chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			LogInfo(&Logger.crc, "ERROR:", fmt.Sprintf("panic: %v\n%s", err, debug.Stack()))
			msgbox("CRC Manager Fail")
			os.Exit(0)
		}
	}()
	defer func() {
		crcRunning.Store(false)
	}()
	crcRunning.Store(true)
	STATUS.mu.RLock()
	masterCRC := STATUS.Data.MasterMap.CRC
	RegMap := make(map[string]map[string]string)
	m := masterCRC.RegMap
	crc := masterCRC.CRC
	for k, innerMap := range m {
		// Clone tầng thứ 2
		newInnerMap := make(map[string]string)
		for ik, iv := range innerMap {
			newInnerMap[ik] = iv
		}
		RegMap[k] = newInnerMap
	}
	STATUS.mu.RUnlock()
	crcLock := script.RegRead(config.RegPath, "lockercrc")
	if strings.EqualFold(crcLock, "Locked") {
		setStatus(keyLocker_CRC, config.Locker_state{Value: "Locked", Color: "OK"})
	} else {
		setStatus(keyLocker_CRC, config.Locker_state{Value: "Unlock", Color: "Fail"})
	}
	if crc == "" || crc == "Fail" || len(RegMap) < 2 {
		setStatus(keyCRC, config.Locker_state{Value: "Fail", Color: "Fail"})
		return
	}
	for path := range RegMap {
		if strings.EqualFold(path, "info") || len(path) < 2 || strings.EqualFold(path, "software") {
			continue
		}
		go a.WatchLock_reg(LockerChan, path)
	}
	go a.CRC_check(LockerChan, Restart_XOIS, RegMap)
	LockerChan <- struct{}{}
	<-a.ctx.Done()

}

func (a mClient) WatchLock_reg(LockerChan chan struct{}, path string) {
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			func() {
				defer func() {
					if err := recover(); err != nil {
						LogInfo(&Logger.crc, "ERROR:", fmt.Sprintf("panic: %v\n%s", err, debug.Stack()))
						msgbox("Watcher Lock Fail")
						os.Exit(0)
					}
				}()
				RegKey, err := registry.OpenKey(registry.CURRENT_USER, path, registry.NOTIFY)
				if err != nil {
					LogInfo(&Logger.crc, "ERROR:", "OpenKey "+path+err.Error())
				}
				defer RegKey.Close()
				handle := windows.Handle(RegKey)
				event, err := windows.CreateEvent(nil, 0, 0, nil)
				if err != nil {
					LogInfo(&Logger.crc, "ERROR:", "CreateEvent failed: "+err.Error())
				}
				defer windows.CloseHandle(event)
				for {
					select {
					case <-a.ctx.Done():
						return
					default:
						err = windows.RegNotifyChangeKeyValue(
							handle,
							true,
							windows.REG_NOTIFY_CHANGE_LAST_SET,
							event,
							true, // async
						)
						if err != nil {
							LogInfo(&Logger.crc, "ERROR:", "Notify setup failed: "+err.Error())
							STATUS.mu.RLock()
							crc := STATUS.Data.MasterMap.CRC.CRC
							STATUS.mu.RUnlock()
							setStatus(keyCRC, config.Locker_state{Value: crc, Color: "Fail"})
							return
						}
						status, err := windows.WaitForSingleObject(event, windows.INFINITE)
						if err != nil {
							LogInfo(&Logger.crc, "ERROR:", "Wait failed: "+err.Error())
							STATUS.mu.RLock()
							crc := STATUS.Data.MasterMap.CRC.CRC
							STATUS.mu.RUnlock()
							setStatus(keyCRC, config.Locker_state{Value: crc, Color: "Fail"})
							return
						}
						if status != windows.WAIT_OBJECT_0 {
							LogInfo(&Logger.crc, "ERROR:", "Unexpected wait status")
							STATUS.mu.RLock()
							crc := STATUS.Data.MasterMap.CRC.CRC
							STATUS.mu.RUnlock()
							setStatus(keyCRC, config.Locker_state{Value: crc, Color: "Fail"})
							return
						}
						LogInfo(&Logger.crc, "ERROR:", "Registry changed:"+path)
						LockerChan <- struct{}{}

					}
				}
			}()
		}
	}
}

func (a mClient) CRC_check(LockerChan chan struct{}, Restart_XOIS chan struct{}, RegMon map[string]map[string]string) {
	scan := time.NewTimer(time.Minute * 2)
	defer scan.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-scan.C:
			LockerChan <- struct{}{}
			scan.Reset(time.Minute * 5)

		case <-LockerChan:
			if !scan.Stop() {
				select {
				case <-scan.C:
				default:
				}
			}
			scan.Reset(time.Minute * 2)
			STATUS.mu.RLock()
			crc_locker := STATUS.Data.Locker_CRC.Value
			crc := strings.TrimSpace(strings.ToUpper(STATUS.Data.MasterMap.CRC.CRC))
			isCRC := strings.TrimSpace(strings.ToUpper(isCurrentCRC.Load().(string)))
			STATUS.mu.RUnlock()
			er_list, crc_status := a.CheckRegistry(RegMon)
			if crc_status {
				setStatus(keyCRC, config.Locker_state{Value: crc, Color: "OK"})
				continue
			}
			if crc == isCRC {
				sendData := map[string][]string{}
				for _, er := range er_list {
					sendData[er.RegPath] = append(sendData[er.RegPath+"_"+er.RegPath], er.RegKey)
				}
				STATUS.mu.RLock()
				var payload = struct {
					CRCid string              `json:"crcid"`
					Data  map[string][]string `json:"data"`
				}{
					CRCid: fmt.Sprintf("%s_%s_%s", strings.Split(STATUS.Data.MasterMap.PGM.PGM, "_")[0], strings.Split(STATUS.Data.MasterMap.PGM.PGM, "_")[1], crc),
					Data:  sendData,
				}
				STATUS.mu.RUnlock()
				jsonData, _ := json.Marshal(payload)

				URL := fmt.Sprintf("http://%s:%s/api/RemoveCRC",
					config.IP.Load().(string),
					config.MasterPort,
				)
				req, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(jsonData))
				if err == nil {
					req.Header.Set("Content-Type", "application/json")
					resp, err := clientSend.Do(req)
					if err == nil {
						resp.Body.Close()
					}
				}
			}
			setStatus(keyCRC, config.Locker_state{Value: crc, Color: "Fail"})

			if strings.EqualFold(crc_locker, "unlock") {
				go func(er_list []config.RegFail) {
					defer func() {
						if err := recover(); err != nil {
							LogInfo(&Logger.crc, "ERROR:", fmt.Sprintf("panic: %v\n%s", err, debug.Stack()))
							os.Exit(0)
						}
					}()
					text := ""
					for _, k := range er_list {
						text += k.RegPath + ":\n" + k.RegKey + " : " + k.ValueOld + " → " + k.ValueMaster + "\n"
					}
					er_list = nil
				}(er_list)
				setStatus(keyLocker_CRC, config.Locker_state{Value: "Unlock", Color: "Fail"})
				continue
			} else {
				setStatus(keyLocker_CRC, config.Locker_state{Value: "Locked", Color: "OK"})
			}
			resetXOISChan <- struct{}{}
			time.Sleep(500 * time.Millisecond)
			LogInfo(&Logger.crc, "ERROR:", er_list)
			a.UpdateCRC(er_list)
			LockerChan <- struct{}{}

		}
	}
}

var lastErrorList = ""

func (a mClient) CheckRegistry(r map[string]map[string]string) ([]config.RegFail, bool) {
	er := []config.RegFail{}
	for path, kv := range r {
		if strings.EqualFold(path, "info") {
			continue
		}
		key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
		if err != nil {
			LogInfo(&Logger.crc, "ERROR:", "Không thể mở reg Path : "+path)
			return er, false
		}
		defer key.Close()

		for k, v := range kv {
			var val string
			if err == nil {
				val, _, err = key.GetStringValue(k)
			} else {
				val = ""
			}
			if err != nil || !script.EqualValue(val, v) {
				er = append(er, config.RegFail{
					RegPath:     path,
					RegKey:      k,
					ValueMaster: v,
					ValueOld:    val,
				})
			}
		}
	}
	text := "CRC Err:\n"
	if len(er) > 0 {
		for _, e := range er {
			text = text + e.RegPath + "\\" + e.RegKey + " : " + e.ValueOld + " → " + e.ValueMaster + "\n"
		}
		if text == lastErrorList {
			LogInfo(&Logger.crc, "INFO", text)
		}
		return er, false
	}
	return er, true
}

func (a mClient) UpdateCRC(err_list []config.RegFail) {

	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for _, key := range err_list {
		select {
		case <-a.ctx.Done():
			return
		case <-timer.C:
			resetXOISChan <- struct{}{}
			timer.Reset(time.Second)
		default:
			if script.RegWrite(key.RegPath, key.RegKey, key.ValueMaster) != nil {
				text := key.RegPath + "\\" + key.RegKey + " Set Fail : " + key.ValueOld + "→" + key.ValueMaster
				LogInfo(&Logger.crc, "ERROR:", text)
				return
			}
			text := key.RegPath + "\\" + key.RegKey + " Set PASS : " + key.ValueOld + "→" + key.ValueMaster
			LogInfo(&Logger.crc, "CHANGE:", text)
		}

	}
}
