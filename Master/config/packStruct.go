package config

import (
	"context"
	"sync"
)

type Locker_struct struct {
	Ctx          context.Context
	Get          func(string) any
	Log          chan []any
	Set          chan<- map[string]Locker_state
	Restart_XOIS chan<- struct{}
	Locker_chan  chan struct{}
	MutexStatus  *sync.RWMutex
	LockerRuning *bool
}

type Merge_struct struct {
	Ctx         context.Context
	Get         func(string) any
	Log         chan []any
	Set         chan<- map[string]Locker_state
	ID          string
	MutexStatus *sync.RWMutex
	MergeRuning *bool
}

type Client_struct struct {
	Ctx            context.Context
	Cancel         context.CancelFunc
	Get            func(string) any
	Log            chan []any
	SetState       chan<- map[string]Locker_state
	SetPGM         chan<- KV
	SetCRC         chan<- map[string]map[string]string
	Status         *STATUS_struct
	Reset_CRC_chan chan struct{}
	Reset_PGM_chan chan struct{}
	MutexStatus    *sync.RWMutex
	SocketRuning   *bool
}
