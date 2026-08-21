package main

import (
	"Locker/config"
	"Locker/script"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

const (
	AppRootPath = "C:\\ProgramData\\Locker"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	mutexSendAdmin sync.Map
	send_admin     = SvelteST{}
	clients        = GlobalClientStruct{
		Client: map[string]*ClientStruct{},
		mu:     sync.RWMutex{},
	}

	Client_list_config struct {
		mu   sync.RWMutex
		Data map[string]config.P
	}

	pgm_master_list = struct {
		mu   sync.RWMutex
		Data map[string]map[string]string
	}{mu: sync.RWMutex{}, Data: map[string]map[string]string{}}

	crc_master_list = struct {
		mu   sync.RWMutex
		Data map[string]map[string]map[string]string
	}{mu: sync.RWMutex{}, Data: map[string]map[string]map[string]string{}}
)

func GetVar() (map[string]config.P, map[string]map[string]string, map[string]map[string]map[string]string) {
	return Client_list_config.Data, pgm_master_list.Data, crc_master_list.Data
}

var waitHeadNew sync.WaitGroup
var mutextHeadNew sync.Mutex

type headStruct struct {
	mu    sync.RWMutex
	model map[string]*headMolde
}

type headMolde struct {
	mu sync.RWMutex
	cd map[string]*headCD
}

type headCD struct {
	mu   sync.RWMutex
	list []string
}

var Headers = &headStruct{
	mu:    sync.RWMutex{},
	model: map[string]*headMolde{},
}

func loadconfig() { // 1. Load Model Config
	if err := script.LoadMasterConfig("Model_config.json", &Client_list_config.Data, &Client_list_config.mu); err != nil {
		Fmt(fmtMaster, "loadconfig", "Model_config.json", err)
		os.Exit(0)
	}

	// 2. Load PGM Master List
	if err := script.LoadMasterConfig("pgm_master_list.json", &pgm_master_list.Data, &pgm_master_list.mu); err != nil {
		Fmt(fmtMaster, "loadconfig", "pgm_master_list.json", err)
		os.Exit(0)
	}

	// 3. Load CRC Master List
	if err := script.LoadMasterConfig("crc_master_list.json", &crc_master_list.Data, &crc_master_list.mu); err != nil {
		Fmt(fmtMaster, "loadconfig", "crc_master_list.json", err)
		os.Exit(0)
	}

	// load heart
	j, er := script.LoadJson(filepath.Join(config.HttpMasterDir, "header.json"))
	if er != nil {
		script.SaveJson(filepath.Join(config.HttpMasterDir, "header.json"), HeadersRaw)
	}
	var header map[string]map[string][]string
	er = json.Unmarshal(j.([]byte), &header)
	if er != nil {
		Fmt(fmtMaster, "loadconfig", er)
		return
	}
	script.MergeHeaders(HeadersRaw, header)
	setHeader(header)
}

func setHeader(header map[string]map[string][]string) {

	Headers.mu.Lock()
	for model, cds := range header {
		if _, ok := Headers.model[model]; !ok {
			Headers.model[model] = &headMolde{
				mu: sync.RWMutex{},
				cd: map[string]*headCD{},
			}
		}
		Headers.model[model].mu.Lock()
		for cd, list := range cds {
			if _, ok := Headers.model[model].cd[cd]; !ok {
				Headers.model[model].cd[cd] = &headCD{
					mu:   sync.RWMutex{},
					list: []string{},
				}
			}
			Headers.model[model].cd[cd].mu.Lock()
			Headers.model[model].cd[cd].list = list
			Headers.model[model].cd[cd].mu.Unlock()
		}
		Headers.model[model].mu.Unlock()
	}
	Headers.mu.Unlock()
}

func ClientWorkerFuncSet(id, Type, action string, data any) {
	ClientWorkerFunc.mu.Lock()
	ClientID := ClientWorkerFunc.ID[id]
	if ClientID == nil {
		ClientID = &TypeListWorker{
			mu:   sync.RWMutex{},
			Type: map[string]Command{},
		}
		ClientWorkerFunc.ID[id] = ClientID
	}
	ClientWorkerFunc.mu.Unlock()
	u := newUUID()
	ClientID.mu.Lock()
	ClientID.Type[Type] = Command{
		UUID:   u,
		Action: action,
		Data:   data,
	}
	ClientID.mu.Unlock()
	Fmt(fmtClient, "ClientWorkerFuncSet", id, Type, action, u)
}

func MultiClientSend(ID, Type, Action string, Data any) {
	var li []string
	ClientOnlineMap.mu.RLock()
	for id := range ClientOnlineMap.Data {
		if strings.HasPrefix(id, ID) {
			li = append(li, id)
		}
	}
	ClientOnlineMap.mu.RUnlock()

	for _, id := range li {
		ClientWorkerFuncSet(id, Type, Action, Data)
	}
}

func Start(c context.Context, ver string, reloadChan chan struct{}, bind, color chan config.KV) {
	//script.CopyFileFull(filepath.Join(config.AppRootDir, "Locker.exe"), filepath.Join(config.HttpDir, "setup_locker.exe"))
	config.MasterLocker = script.RegRead(config.RegPath, "lockerVer")
	config.MasterAOI = script.RegRead(config.RegPath, "aoiVer")
	config.MasterUIAoi = script.RegRead(config.RegPath, "uiaoiVer")
	config.MasterUILocker = script.RegRead(config.RegPath, "uilockerVer")
	config.MasterService = script.RegRead(config.RegPath, "serviceVer")
	var tmp = script.RegRead(config.RegPath, "ActiveClient")
	listActive.mu.Lock()
	if len(tmp) > 0 {
		for _, i := range strings.Split(tmp, "|") {
			listActive.data[i] = struct{}{}
		}
	}
	listActive.mu.Unlock()
	go LanCard(bind, color)
	Ver = ver
	if _, err := os.Stat(config.HttpDir); os.IsNotExist(err) {
		os.MkdirAll(config.HttpDir, 0755)
	}

	loadconfig()
	go udpBroadcast()
	for range 20 {
		go syscCSV()
	}
	go CSVMerge()
	apiManager()
}
