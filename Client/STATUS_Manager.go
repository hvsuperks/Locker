package main

import (
	"reflect"
	"sync"
)

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

var STATUS = STATUS_struct{
	MasterMap: MasterMap_struct{
		CRC: RegMap_struct{},
		PGM: FileMap_struct{},
	},
	VerLocker: Ver,
}

var muStatus = sync.RWMutex{}

func STATUS_PUT(dst any, val any) {
	//defer Recoverlog()
	muStatus.Lock()
	defer muStatus.Unlock()

	switch p := dst.(type) {
	case *string:
		if v, ok := val.(string); ok {
			*p = v
		}
	case *Locker_state:
		if v, ok := val.(Locker_state); ok {
			*p = v
		}
	case *bool:
		if v, ok := val.(bool); ok {
			*p = v
		}
	case *MasterMap_struct:
		if v, ok := val.(MasterMap_struct); ok {
			*p = v
		}
	case *RegMap_struct:
		if v, ok := val.(RegMap_struct); ok {
			*p = v
		}

	case *FileMap_struct:
		if v, ok := val.(FileMap_struct); ok {
			*p = v
		}
	}
	go func() {
		SendStatus <- struct{}{}
	}()

}

func STATUS_GET(src any) any {
	muStatus.RLock()
	defer muStatus.RUnlock()
	v := reflect.ValueOf(src)

	if v.Kind() == reflect.Ptr {
		return v.Elem().Interface()
	}

	return v.Interface()
}
