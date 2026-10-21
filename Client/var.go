package main

import (
	"sync"
	"sync/atomic"
)

var (
	reconnecting  atomic.Bool
	reconnectLock atomic.Bool
	isdownload    atomic.Bool
)

var Log = make(chan []any, 100)
var RequestCount = 0
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
