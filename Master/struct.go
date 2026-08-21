package main

import (
	"Locker/config"
	"sync"
)

type GlobalClientStruct struct {
	Client map[string]*ClientStruct
	mu     sync.RWMutex
}

type ClientStruct struct {
	mu     sync.RWMutex
	Status *config.STATUS_struct
}

type SendMasterConfig struct {
	Md5Map      map[string]map[string]string            `json:"md5map"`
	RegMap      map[string]map[string]map[string]string `json:"regmap"`
	Modelconfig map[string]config.P                     `json:"modelconfig"`
}

type apiVerStruct struct {
	Mode  string `json:"mode"`
	Value string `json:"value"`
}
