package main

import (
	"Locker/config"
	"Locker/script"
	"context"
	"net/http"
	"os"
	"os/exec"
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

func loadconfig() { // 1. Load Model Config
	cmd := exec.Command(
		"net",
		"use",
		"Z:",
		`\\10.1.1.31\고객불량 대책`,
		"jahwa.ct$",
		`/user:ct4`,
		"/persistent:no",
	)
	out, err := cmd.CombinedOutput()
	LogInfo(&Logger.MAIN, "net use failed:", err)
	LogInfo(&Logger.MAIN, "output:", string(out))

	if err := script.LoadMasterConfig("Model_config.json", &Client_list_config.Data, &Client_list_config.mu); err != nil {
		LogInfo(&Logger.MAIN, "loadconfig", "Model_config.json", err)
		os.Exit(0)
	}

	// 2. Load PGM Master List
	if err := script.LoadMasterConfig("pgm_master_list.json", &pgm_master_list.Data, &pgm_master_list.mu); err != nil {
		LogInfo(&Logger.MAIN, "loadconfig", "pgm_master_list.json", err)
		os.Exit(0)
	}

	// 3. Load CRC Master List
	if err := script.LoadMasterConfig("crc_master_list.json", &crc_master_list.Data, &crc_master_list.mu); err != nil {
		LogInfo(&Logger.MAIN, "loadconfig", "crc_master_list.json", err)
		os.Exit(0)
	}
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

func Start(ctx context.Context, cancel context.CancelFunc, ver string) {
	LogInfo(&Logger.Debug, "Start")
	config.MasterLocker.Store(RegRead(config.RegPath, "lockerVer"))
	config.MasterAOI.Store(RegRead(config.RegPath, "aoiVer"))
	config.MasterUIAoi.Store(RegRead(config.RegPath, "uiaoiVer"))
	config.MasterUILocker.Store(RegRead(config.RegPath, "uilockerVer"))
	config.MasterService.Store(RegRead(config.RegPath, "serviceVer"))
	config.MasterVbatoolVer.Store(RegRead(config.RegPath, "vbatoolVer"))
	var tmp = RegRead(config.RegPath, "ActiveClient")
	LogInfo(&Logger.Debug, config.RegPath)
	LogInfo(&Logger.Debug, tmp, len(tmp))
	listActive.mu.Lock()
	if len(tmp) > 0 {
		for _, i := range strings.Split(tmp, "|") {
			listActive.data[i] = struct{}{}

		}
	}
	LogInfo(&Logger.Debug, listActive.data)
	listActive.mu.Unlock()
	Ver = ver
	if _, err := os.Stat(config.HttpDir); os.IsNotExist(err) {
		os.MkdirAll(config.HttpDir, 0755)
	}

	loadconfig()
	go udpBroadcast()
	//go syscCSV(ctx)
	//go CSVMerge(ctx)
	go manager.CleanUUID()
	initGORM()
	initAdmin()
	apiManager(ctx, cancel)
}
