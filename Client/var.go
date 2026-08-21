package main

import (
	"Locker/config"
	"sync"
	"sync/atomic"
)

var (
	reconnecting  atomic.Bool
	reconnectLock atomic.Bool
	isdownload    atomic.Bool
)

var (
	keyMasterMap  = 1
	keyLocker_PGM = 2
	keyLocker_CRC = 3
	keyMerge      = 4
	keyConnect    = 5
	keyPGM        = 6
	keyCRC        = 7
)

// Client
var STATUS = struct {
	mu   sync.RWMutex
	Data *config.STATUS_struct
}{mu: sync.RWMutex{}, Data: &config.STATUS_struct{MasterMap: &config.MasterMap_struct{
	CRC: config.RegMap_struct{},
	PGM: config.FileMap_struct{},
}}}

var Log = make(chan []any, 100)
var RequestCount = 0
var Bind chan config.KV
var Color chan config.KV
var resetPackChan = make(chan string, 10)
var (
	muFmt sync.RWMutex
	wg    sync.WaitGroup
)

var (
	resetXOISChan = make(chan struct{}, 10)
)

// Main

// CSV
var lotCount = make(chan struct{}, 200)
var lotWriterChan sync.Map
var Lot_chan sync.Map
var ID, model string
