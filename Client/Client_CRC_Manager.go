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
	defer RecoverLog()
	defer func() {
		crcRunning.Store(false)
	}()
	crcRunning.Store(true)
	masterCRC := STATUS_GET(&STATUS.MasterMap.CRC).(RegMap_struct)
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
	crcLock := script.RegRead(config.RegPath, "lockercrc")
	if strings.EqualFold(crcLock, "Locked") {
		STATUS_PUT(&STATUS.Locker_CRC, Locker_state{Value: "Locked", Color: "OK"})
	} else {
		STATUS_PUT(&STATUS.Locker_CRC, Locker_state{Value: "Unlock", Color: "Fail"})
	}
	if crc == "" || crc == "Fail" || len(RegMap) < 2 {
		STATUS_PUT(&STATUS.MasterMap.CRC.CRC, "Fail")
		STATUS_PUT(&STATUS.MasterMap.CRC.Color, "Fail")
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
				defer RecoverLog()
				RegKey, err := registry.OpenKey(registry.CURRENT_USER, path, registry.NOTIFY)
				if err != nil {
					LogInfo(&Logger.debug, "OpenKey "+path+err.Error())
				}
				defer RegKey.Close()
				handle := windows.Handle(RegKey)
				event, err := windows.CreateEvent(nil, 0, 0, nil)
				if err != nil {
					LogInfo(&Logger.debug, "CreateEvent failed: "+err.Error())
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
							LogInfo(&Logger.debug, "Notify setup failed: "+err.Error())
							STATUS_PUT(&STATUS.MasterMap.CRC.Color, "Fail")
							return
						}
						status, err := windows.WaitForSingleObject(event, windows.INFINITE)
						if err != nil {
							LogInfo(&Logger.debug, "Wait failed: "+err.Error())
							STATUS_PUT(&STATUS.MasterMap.CRC.Color, "Fail")
							return
						}
						if status != windows.WAIT_OBJECT_0 {
							LogInfo(&Logger.debug, "Unexpected wait status")
							STATUS_PUT(&STATUS.MasterMap.CRC.Color, "Fail")
							return
						}
						LogInfo(&Logger.debug, "Registry changed:"+path)
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

			crc := strings.TrimSpace(strings.ToUpper(STATUS_GET(&STATUS.MasterMap.CRC.CRC).(string)))
			isCRC := strings.TrimSpace(strings.ToUpper(isCurrentCRC.Load().(string)))
			er_list, crc_status := a.CheckRegistry(RegMon)
			if crc_status {
				STATUS_PUT(&STATUS.MasterMap.CRC.Color, "OK")
				continue
			}
			if crc == isCRC {
				sendData := map[string][]string{}
				for _, er := range er_list {
					sendData[er.RegPath] = append(sendData[er.RegPath+"_"+er.RegPath], er.RegKey)
				}
				tmppgm := STATUS_GET(&STATUS.MasterMap.PGM.PGM).(string)
				var payload = struct {
					CRCid string              `json:"crcid"`
					Data  map[string][]string `json:"data"`
				}{
					CRCid: fmt.Sprintf("%s_%s_%s", strings.Split(tmppgm, "_")[0], strings.Split(tmppgm, "_")[1], crc),
					Data:  sendData,
				}
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
			STATUS_PUT(&STATUS.MasterMap.CRC.Color, "Fail")
			crc_locker := strings.ToLower(STATUS_GET(&STATUS.Locker_CRC.Value).(string))
			if strings.EqualFold(crc_locker, "unlock") {
				go func(er_list []RegFail) {
					defer func() {
						if err := recover(); err != nil {
							LogInfo(&Logger.debug, fmt.Sprintf("panic: %v\n%s", err, debug.Stack()))
							os.Exit(0)
						}
					}()
					text := ""
					for _, k := range er_list {
						text += k.RegPath + ":\n" + k.RegKey + " : " + k.ValueOld + " → " + k.ValueMaster + "\n"
					}
					er_list = nil
				}(er_list)
				STATUS_PUT(&STATUS.Locker_CRC, Locker_state{Value: "Unlock", Color: "Fail"})
				continue
			} else {
				STATUS_PUT(&STATUS.Locker_CRC, Locker_state{Value: "Locked", Color: "OK"})
			}
			resetXOISChan <- struct{}{}
			time.Sleep(500 * time.Millisecond)
			LogInfo(&Logger.crc, er_list)
			a.UpdateCRC(er_list)
			LockerChan <- struct{}{}

		}
	}
}

var lastErrorList = ""

func (a mClient) CheckRegistry(r map[string]map[string]string) ([]RegFail, bool) {
	er := []RegFail{}
	for path, kv := range r {
		if strings.EqualFold(path, "info") {
			continue
		}
		key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
		if err != nil {
			LogInfo(&Logger.debug, "Không thể mở reg Path : "+path)
			script.DeleteTree(registry.CURRENT_USER, config.RegCRCMasterPath)
			a.cancel()
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
				er = append(er, RegFail{
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

func (a mClient) UpdateCRC(err_list []RegFail) {

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
				LogInfo(&Logger.debug, text)
				return
			}
			text := key.RegPath + "\\" + key.RegKey + " Set PASS : " + key.ValueOld + "→" + key.ValueMaster
			LogInfo(&Logger.crc, "CHANGE:", text)
		}

	}
}
