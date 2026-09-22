package config

import (
	"sync"
	"sync/atomic"
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
	MD5    string                       `json:"md5"`
}

type MasterMap_struct struct {
	CRC RegMap_struct  `json:"crc"`
	PGM FileMap_struct `json:"pgm"`
}

type STATUS_struct struct {
	MasterMap     *MasterMap_struct `json:"mastermap"`
	VerLocker     string            `json:"verlocker"`
	VerUiLocker   string            `json:"veruilocker"`
	VerAutoUpdate string            `json:"verautoupdate"`
	Model         string            `json:"model"`
	ID            string            `json:"id"`
	Locker_PGM    Locker_state      `json:"lockerpgm"` // Lưu trạng thái Unlock / Lock PGM
	Locker_CRC    Locker_state      `json:"lockercrc"` // Lưu trạng thái Unlock / Lock CRC
	Merge         Locker_state      `json:"merge"`     // Trạng thái ghép file
	MesID         string            `json:"mesid"`
	CurrentCRC    string            `json:"currentcrc"`
}

type RegFail struct {
	RegPath     string
	RegKey      string
	ValueMaster string
	ValueOld    string
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
