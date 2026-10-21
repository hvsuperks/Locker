package main

import (
	"context"
)

type mClient struct {
	ctx           context.Context
	cancel        context.CancelFunc
	Log           chan<- []any
	resetPackChan chan string
	bind, color   chan KV
}

type DownloadStruct struct {
	url  string
	dst  string
	mode string
}

type setStatusStruct struct {
	Key   int
	Value string
	Color string
}
