package main

import (
	"Locker/config"
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type message struct {
	Type      string   `json:"type"`
	Data      SvelteST `json:"data"`
	RequestID string   `json:"requestid"`
	Report    string   `json:"report"`
}
type May struct {
	Locker_PGM config.Locker_state `json:"lockerpgm"` // Lưu trạng thái Unlock / Lock PGM
	Locker_CRC config.Locker_state `json:"lockercrc"` // Lưu trạng thái Unlock / Lock CRC
	PGM        config.Locker_state `json:"pgm"`       // Lưu PGM hiện tại (Model,PID,Ver..v.v..)
	CRC        config.Locker_state `json:"crc"`       // Lưu mã CRC
	Merge      config.Locker_state `json:"merge"`     // Trạng thái ghép file
	Connect    config.Locker_state `json:"connect"`
	Ver        string              `json:"ver"`
	MesID      string              `json:"mesid"`
	CurrentCRC string              `json:"currentcrc"`
}

type Model struct {
	ModelName string                 `json:"model_name"`
	CongDoan  map[string]map[int]May `json:"congdoan"`
}

type MaxIDMap struct {
	Marking int `json:"marking"`
	AF      int `json:"af"`
	OIS     int `json:"ois"`
	Tilt    int `json:"tilt"`
	Prism   int `json:"prism"`
	Fra_AF  int `json:"fra_af"`
	Fra_OIS int `json:"fra_ois"`
	AOI     int `json:"aoi"`
	Flag    int `json:"flag"`
}

type SvelteST struct {
	Model        []Model                        `json:"model"`
	Masterconfig SendSvelteConfig               `json:"masterconfig"`
	Connect      bool                           `json:"connect"`
	Ver          string                         `json:"ver"`
	StartTime    string                         `json:"starttime"`
	NatLogin     bool                           `json:"natlogin"`
	LockerVer    string                         `json:"lockerver"`
	UiLockerVer  string                         `json:"uilockerver"`
	AoiVer       string                         `json:"aoiver"`
	UiAoiVer     string                         `json:"uiaoiver"`
	ServiceVer   string                         `json:"servicever"`
	FileList     map[string]map[string]struct{} `json:"filelist"`
}

type SendSvelteConfig struct {
	Md5Map      map[string]map[string]string            `json:"md5map"`
	RegMap      map[string]map[string]map[string]string `json:"regmap"`
	Modelconfig map[string]config.P                     `json:"modelconfig"`
}

var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		// Tạo sẵn một buffer với dung lượng mồi (ví dụ 1MB) để đỡ phải nới rộng mảng
		return bytes.NewBuffer(make([]byte, 0, 1024*1024))
	},
}
var lastcheckMap = atomic.Value{}
var natonline = atomic.Bool{}
var ListFile = struct {
	mu   sync.RWMutex
	Data map[string]map[string]struct{}
}{
	mu:   sync.RWMutex{},
	Data: make(map[string]map[string]struct{}),
}

func AddFile(group, filename string) {
	if ListFile.Data[group] == nil {
		ListFile.Data[group] = make(map[string]struct{})
	}
	ListFile.Data[group][filename] = struct{}{}
}

func GetMachineList() SvelteST {
	if time.Since(lastcheckMap.Load().(time.Time)) > 10*time.Second {
		if _, err := os.Stat("Z:\\X3T1"); err == nil {
			natonline.Store(true)
		} else {
			natonline.Store(false)
		}
		lastcheckMap.Store(time.Now())
	}
	var sendWebControl = make(map[string]map[string]map[int]May)
	tempClients := make(map[string]*ClientStruct, len(clients.Client))
	clients.mu.RLock()
	maps.Copy(tempClients, clients.Client)
	clients.mu.RUnlock()

	onlineTmp := map[string]config.Locker_state{}
	ClientOnlineMap.mu.RLock()
	for i, t := range ClientOnlineMap.Data {
		if time.Since(t) > 5*time.Second {
			onlineTmp[i] = config.Locker_state{Value: "Disconnected", Color: "Fail"}
		} else {
			onlineTmp[i] = config.Locker_state{Value: "Connecting", Color: "OK"}
		}
	}
	ClientOnlineMap.mu.RUnlock()

	for idClient, client := range tempClients {
		client.mu.RLock()
		model_name := client.Status.Model
		if model_name == "" {
			client.mu.RUnlock()
			continue
		}
		congdoan_name := config.TypeMap[idClient[6:7]]
		may_name, _ := strconv.Atoi(idClient[7:])
		if _, ok := sendWebControl[model_name]; !ok {
			sendWebControl[model_name] = map[string]map[int]May{}
		}
		if _, ok := sendWebControl[model_name][congdoan_name]; !ok {
			sendWebControl[model_name][congdoan_name] = map[int]May{}
		}
		sendWebControl[model_name][congdoan_name][may_name] = May{
			Locker_PGM: client.Status.Locker_PGM,
			Locker_CRC: client.Status.Locker_CRC,
			PGM:        config.Locker_state{Value: client.Status.MasterMap.PGM.PGM, Color: client.Status.MasterMap.PGM.Color},
			CRC:        config.Locker_state{Value: client.Status.MasterMap.CRC.CRC, Color: client.Status.MasterMap.CRC.Color},
			Merge:      client.Status.Merge,
			Connect:    onlineTmp[idClient],
			Ver:        fmt.Sprintf("%s|%s|%s", client.Status.VerAutoUpdate, client.Status.VerLocker, client.Status.VerUiLocker),
			MesID:      client.Status.MesID,
			CurrentCRC: client.Status.CurrentCRC,
		}

		client.mu.RUnlock()
	}

	models := []Model{}
	for model, congdoans := range sendWebControl {
		cd := map[string]map[int]May{}
		maps.Copy(cd, congdoans)
		models = append(models, Model{
			ModelName: model,
			CongDoan:  cd,
		})
	}

	//fmt.Println(list)
	m := SendSvelteConfig{
		Modelconfig: Client_list_config.Data,
		Md5Map:      pgm_master_list.Data,
		RegMap:      crc_master_list.Data,
	}

	listfile, err := os.ReadDir(config.HttpDir)
	if err == nil {
		ListFile.mu.Lock()
		for _, file := range listfile {
			if file.IsDir() {
				continue
			}
			filename := strings.ToLower(file.Name())
			if strings.HasSuffix(filename, ".exe") {
				if strings.Contains(filename, "setup_locker") {
					AddFile("lockerver", filename)
				} else if strings.Contains(filename, "ui_ver") {
					AddFile("uilockerver", filename)
				} else if strings.Contains(filename, "setup_aoi") {
					AddFile("aoiver", filename)
				} else if strings.Contains(filename, "uiaoi_ver") {
					AddFile("uiaoiver", filename)
				}
			}
		}
		ListFile.mu.Unlock()
	}
	ListFile.mu.RLock()
	tmpmap := map[string]map[string]struct{}{}
	for k, v := range ListFile.Data {
		tmpmap[k] = map[string]struct{}{}
		for i := range v {
			tmpmap[k][i] = struct{}{}
		}
	}
	ListFile.mu.RUnlock()
	return SvelteST{
		Model:        models,
		Connect:      true,
		Masterconfig: m,
		Ver:          Ver,
		StartTime:    config.StartTime,
		NatLogin:     natonline.Load(),
		LockerVer:    config.MasterLocker,
		UiLockerVer:  config.MasterUILocker,
		AoiVer:       config.MasterAOI,
		UiAoiVer:     config.MasterUIAoi,
		ServiceVer:   config.MasterService,
		FileList:     tmpmap,
	}
}

func APIStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Type string   `json:"type"`
		Data SvelteST `json:"data"`
	}{
		Type: "status",
		Data: GetMachineList(),
	})
}
