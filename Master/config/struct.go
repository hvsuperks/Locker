package config

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

type Locker_state struct {
	Value string `json:"value"`
	Color string `json:"color"`
}

type FileMap_struct struct {
	PGM   string `json:"pgm"`
	MD5   string `json:"md5"`
	URL   string `json:"url"`
	Color string `json:"color"`
}

type RegMap_struct struct {
	CRC    string                       `json:"crc"`
	RegMap map[string]map[string]string `json:"regmap"`
	Color  string                       `json:"color"`
}

type MasterMap_struct struct {
	CRC RegMap_struct  `json:"crc"`
	PGM FileMap_struct `json:"pgm"`
}

type STATUS_struct struct {
	MasterMap     MasterMap_struct `json:"mastermap"`
	VerLocker     string           `json:"verlocker"`
	VerUiLocker   string           `json:"veruilocker"`
	VerAutoUpdate string           `json:"verautoupdate"`
	Model         string           `json:"model"`
	ID            string           `json:"id"`
	Locker_PGM    Locker_state     `json:"lockerpgm"` // Lưu trạng thái Unlock / Lock PGM
	Locker_CRC    Locker_state     `json:"lockercrc"` // Lưu trạng thái Unlock / Lock CRC
	Merge         Locker_state     `json:"merge"`     // Trạng thái ghép file
	MesID         string           `json:"mesid"`
	CurrentCRC    string           `json:"currentcrc"`
}

type Message struct {
	ID        string          `json:"id"`
	RequestID string          `json:"requestID"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Text      string          `json:"text"`
}

type RegFail struct {
	RegPath     string
	RegKey      string
	ValueMaster string
	ValueOld    string
}

type Chan_struct struct {
	LockChange_PGM chan struct{}
	LockChange_CRC chan struct{}
	Send_Master    chan Message
	UDP_change     chan string
	FailList_PGM   chan []string
	FailList_CRC   chan []RegFail
	UpdateState    chan map[string]Locker_state
	UpdatePGM      chan map[string]map[string]string
	UpdateCRC      chan map[string]map[string]string
	Reset_CRC      chan struct{}
	Reset_PGM      chan struct{}
	Reset_Merge    chan struct{}
	Reset_Socket   chan struct{}
}

type CsvRow struct {
	File_name string
	Header    []string
	Data      map[string]string
	RawFile   *RawFileStatus
}

type RawFileStatus struct {
	Wg     *sync.WaitGroup
	Result *atomic.Bool
}

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Info struct {
	SharePath string
	Path      string
	User      string
	Pass      string
	FileName  string
}

type ConfigVar struct {
	BackupRoot    *string
	BackupRoot2   *string
	OutputRoot    *string
	MesPath       *string
	UploadRoot    *string
	LotColumn     *string
	User          *string
	Pass          *string
	MasterPath    *string
	XoisPath      *string
	RegPath       *string
	LogPath       *string
	AppRootPath   *string
	AppDogPath    *string
	UdpPort       *string
	MasterPort    *string
	LockPath      *string
	ControlFolder *string
	UpdatePath    *string
	MasterName    *string
}

type SendMasterStruct struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type CsvJsonData struct {
	Header []string            `json:"header"`
	Data   []map[string]string `json:"data"`
}

type ManagedChan struct {
	Ch        chan struct{}
	CreatedAt time.Time
}

type Send_client_struct struct {
	Msg Message
	ID  string
}

type P struct {
	PGM string `json:"pgm"`
	CRC string `json:"crc"`
}
