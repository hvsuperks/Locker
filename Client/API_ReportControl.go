package main

import (
	"Locker/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type clientPost struct {
	ID   string          `json:"id"`
	UUID string          `json:"uuid"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

var clientSend = &http.Client{
	Timeout: 5 * time.Second,
}

var muClientReport = sync.Mutex{}

func ClientReport(payload clientPost) error {
	muClientReport.Lock()
	defer muClientReport.Unlock()
	URL := fmt.Sprintf("http://%s:%s/api/clientReport",
		config.IP.Load().(string),
		config.MasterPort,
	)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		LogInfo(&Logger.debug, "ClientReport json Marshal", err)
		if payload.Type == "status" {
			return err
		}
		time.Sleep(time.Second * 2)
		ClientReport(payload)
		return err
	}
	// Tạo request
	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(jsonData))
	if err != nil {
		LogInfo(&Logger.debug, "ClientReport Request", err)
		if payload.Type == "status" {
			return err
		}
		time.Sleep(time.Second * 2)
		ClientReport(payload)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := clientSend.Do(req)
	if err != nil {
		LogInfo(&Logger.debug, "ClientReport client", err)
		if payload.Type == "status" {
			return err
		}
		time.Sleep(time.Second * 2)
		ClientReport(payload)
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// Check status
	if resp.StatusCode != http.StatusOK {
		LogInfo(&Logger.debug, "ClientReport status :", resp.StatusCode, string(body))
		if payload.Type == "status" {
			return err
		}
		time.Sleep(time.Second * 2)
		ClientReport(payload)
	}
	return nil
}
