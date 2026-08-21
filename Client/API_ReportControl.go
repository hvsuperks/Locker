package main

import (
	"Locker/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func ClientReport(payload clientPost) error {
	URL := fmt.Sprintf("http://%s:%s/api/ClientReport",
		config.IP.Load().(string),
		config.MasterPort,
	)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		LogInfo(&Logger.connect, "ERROR: ClientReport json Marshal", err)
		if payload.Type == "status" {
			return err
		}
		go ClientReport(payload)
		return err
	}
	// Tạo request
	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(jsonData))
	if err != nil {
		LogInfo(&Logger.connect, "ERROR:", "ClientReport Request", err)
		if payload.Type == "status" {
			return err
		}
		go ClientReport(payload)

		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := clientSend.Do(req)
	if err != nil {
		LogInfo(&Logger.connect, "ERROR:", "ClientReport client", err)
		if payload.Type == "status" {
			return err
		}
		go ClientReport(payload)
		return err
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		LogInfo(&Logger.connect, "ERROR:", "ClientReport status :", resp.StatusCode, string(body))
		if payload.Type == "status" {
			return err
		}
		go ClientReport(payload)
	}
	return nil
}
