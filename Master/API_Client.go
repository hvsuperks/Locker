package main

import (
	"Locker/config"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TypeListWorker struct {
	mu   sync.RWMutex
	Type map[string]Command
}
type Command struct {
	UUID   string `json:"uuid"`
	Action string `json:"action"`
	Data   any    `json:"data"`
}
type IDListWorker struct {
	mu sync.RWMutex
	ID map[string]*TypeListWorker
}

type clientPost struct {
	ID   string          `json:"id"`
	UUID string          `json:"uuid"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

var ClientWorkerFunc = &IDListWorker{mu: sync.RWMutex{}, ID: make(map[string]*TypeListWorker)}
var ClientOnlineMap = struct {
	mu   sync.RWMutex
	Data map[string]time.Time
}{mu: sync.RWMutex{}, Data: map[string]time.Time{}}

type listActiveType struct {
	mu   sync.RWMutex
	data map[string]struct{}
}

var listActive = listActiveType{
	mu:   sync.RWMutex{},
	data: map[string]struct{}{},
}

func newUUID() string {
	return uuid.NewString()
}

func Get_clientPingPong(w http.ResponseWriter, r *http.Request) {
	ID := r.URL.Query().Get("ID")
	if len(ID) != 9 {
		http.Error(w, "ID Fail", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	clients.mu.RLock()
	client := clients.Client[ID]
	clients.mu.RUnlock()

	// ✅ client chưa tồn tại → yêu cầu gửi info
	if client == nil {
		json.NewEncoder(w).Encode(map[string]any{
			"needInfo": true,
		})
		return
	}
	ClientOnlineMap.mu.Lock()
	last := ClientOnlineMap.Data[ID]
	ClientOnlineMap.Data[ID] = time.Now()
	ClientOnlineMap.mu.Unlock()
	ClientWorkerFunc.mu.RLock()
	Worker := ClientWorkerFunc.ID[ID]
	ClientWorkerFunc.mu.RUnlock()
	if time.Since(last) > 5*time.Second {
		json.NewEncoder(w).Encode(map[string]any{
			"needInfo": true,
		})
		return
	}
	// ✅ không có command
	if Worker == nil {
		json.NewEncoder(w).Encode(map[string]any{
			"hasCommand": false,
		})
		return
	}

	Worker.mu.RLock()
	defer Worker.mu.RUnlock()

	// ✅ có command
	json.NewEncoder(w).Encode(map[string]any{
		"hasCommand": true,
		"data":       Worker.Type,
	})
}

func POST_clientReport(w http.ResponseWriter, r *http.Request) {
	var j clientPost
	// ✅ decode JSON từ body
	err := json.NewDecoder(r.Body).Decode(&j)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch j.Type {
	case "status":
		var data config.STATUS_struct

		if err := json.Unmarshal(j.Data, &data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(j.ID) < 8 {
			http.Error(w, "ID Fail", http.StatusBadRequest)
			return
		}
		clients.mu.RLock()
		client := clients.Client[j.ID]
		clients.mu.RUnlock()
		if client == nil {
			clients.mu.Lock()
			client = &ClientStruct{
				mu:     sync.RWMutex{},
				Status: &config.STATUS_struct{},
			}
			clients.Client[j.ID] = client
			clients.mu.Unlock()
		}
		Client_list_config.mu.RLock()
		master, ok := Client_list_config.Data[j.ID[:7]]
		Client_list_config.mu.RUnlock()
		if !ok {
			LogInfo(&Logger.API, "POST_clientReport", j.ID[:6], " Client_list_config NoData")
			break
		}
		listActive.mu.RLock()
		_, ok = listActive.data[j.ID[:6]]
		listActive.mu.RUnlock()
		if !ok {
			LogInfo(&Logger.API, "POST_clientReport", j.ID[:6], " notActive")
			break
		}
		crc := data.MasterMap.CRC.Color
		pgm := data.MasterMap.PGM.Color
		m := strings.Split(master.PGM, "_")
		f := fmt.Sprintf("%s_%s_%s", m[0], m[1], master.CRC)
		crc_master_list.mu.RLock()
		c, ok := crc_master_list.Data[f]
		crcMd5 := ""
		if ok {
			if info, o := c["info"]; o {
				if crcMd5, o = info["md5"]; !ok {
					crcMd5 = ""
				}
			}
		}
		crc_master_list.mu.RUnlock()
		if ok && master.CRC != "" && master.PGM != "" {

			if data.MasterMap.CRC.CRC != master.CRC || (len(crcMd5) > 5 && data.MasterMap.CRC.MD5 != crcMd5) || (len(data.CurrentCRC) == 4 && strings.ToLower(data.CurrentCRC) != "none" && master.CRC != data.CurrentCRC && data.Locker_CRC.Value == "Locked") {
				id := ""
				if len(j.ID) > 7 {
					id = j.ID[:6]
				}
				LogInfo(&Logger.API, "POST_clientReport", fmt.Sprintf("ID: %s - data.MasterMap.CRC.CRC: %s - Master: %s - Client: %s", id, data.MasterMap.CRC.CRC, master.CRC, data.CurrentCRC))
				crc = "Fail"

				ClientWorkerFuncSet(j.ID, "crc_change", master.CRC, config.RegMap_struct{CRC: master.CRC, RegMap: c})
			}
			m := strings.Split(master.PGM, "_")
			f := fmt.Sprintf("%s_%s", m[0], m[1])
			pgm_master_list.mu.RLock()
			model_json := pgm_master_list.Data[f]
			pgm_master_list.mu.RUnlock()
			if data.MasterMap.PGM.PGM != master.PGM || data.MasterMap.PGM.MD5 != model_json[master.PGM] {
				pgm = "Fail"
				ClientWorkerFuncSet(j.ID, "pgm_change", master.PGM, config.FileMap_struct{PGM: master.PGM, MD5: model_json[master.PGM], URL: "PGM/" + m[0] + "/" + m[1] + "/" + master.PGM + ".zip"})

			}
		}
		tmpClient := config.STATUS_struct{
			MasterMap: config.MasterMap_struct{
				CRC: config.RegMap_struct{
					CRC:    data.MasterMap.CRC.CRC,
					RegMap: data.MasterMap.CRC.RegMap,
					Color:  crc,
					MD5:    data.MasterMap.CRC.MD5,
				},
				PGM: config.FileMap_struct{
					PGM:   data.MasterMap.PGM.PGM,
					MD5:   data.MasterMap.PGM.MD5,
					URL:   data.MasterMap.PGM.URL,
					Color: pgm,
				},
			},
			VerLocker:     data.VerLocker,
			VerUiLocker:   data.VerUiLocker,
			VerAutoUpdate: data.VerAutoUpdate,
			Locker_PGM:    data.Locker_PGM,
			Locker_CRC:    data.Locker_CRC,
			Merge:         data.Merge,
			Model:         data.Model,
			MesID:         data.MesID,
			CurrentCRC:    data.CurrentCRC,
		}

		client.mu.Lock()
		client.Status = &tmpClient
		client.mu.Unlock()

	default:
		ClientWorkerFunc.mu.RLock()
		worker := ClientWorkerFunc.ID[j.ID]
		ClientWorkerFunc.mu.RUnlock()

		if worker != nil {
			worker.mu.Lock()
			if worker.Type != nil {
				command, ok := worker.Type[j.Type]
				if ok && command.UUID == j.UUID {
					delete(worker.Type, j.Type)
				}
			}
			worker.mu.Unlock()
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
