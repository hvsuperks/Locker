package main

import (
	"Locker/config"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type PingResponse struct {
	NeedInfo   bool               `json:"needInfo"`
	HasCommand bool               `json:"hasCommand"`
	Type       string             `json:"type"`
	Result     bool               `json:"result"`
	Data       map[string]Command `json:"data"`
}

var WorkerList = struct {
	mu   sync.RWMutex
	List map[string]Command
}{mu: sync.RWMutex{}, List: map[string]Command{}}

type Command struct {
	UUID   string          `json:"uuid"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

var controlListSkip = struct {
	mu   sync.RWMutex
	Data map[string]struct{}
}{mu: sync.RWMutex{}, Data: map[string]struct{}{}}

var isDisconnect = atomic.Bool{}

func PingPong(ctx context.Context, cancel context.CancelFunc) {
	fmt.Println("pingpong Start")
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	isRun := make(chan struct{}, 1)
	for {
		if config.IP.Load() != "" {
			break
		}
		time.Sleep(time.Second)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			go func() {
				select {
				case isRun <- struct{}{}:
					break
				default:
					return
				}
				defer func() {
					<-isRun
				}()

				p, err := GetControl(cancel)
				if err != nil {
					isDisconnect.Store(false)
					fmt.Println("GetControl", err)
					return
				}

				if !isDisconnect.Load() {
					SendStatus <- struct{}{}
				}

				isDisconnect.Store(true)
				if p.NeedInfo {
					SendStatus <- struct{}{}
					return
				}
				if p.HasCommand {
					for T, v := range p.Data {
						fmt.Println("Type", T, v.Action)
						go Dispacht(cancel, T, v)
					}
				}

			}()
		}
	}
}

// Api GetPingPong
func GetControl(cancel context.CancelFunc) (*PingResponse, error) {
	url := fmt.Sprintf("http://%s:%s/api/GetPingPong?ID=%s", config.IP.Load(), config.MasterPort, ID)

	var httpClient = &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not ok: %d", resp.StatusCode)
	}

	var result PingResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Dispacht(cancel context.CancelFunc, ControlType string, msg Command) {
	controlListSkip.mu.RLock()
	_, ok := controlListSkip.Data[msg.UUID]
	controlListSkip.mu.RUnlock()
	payload := &clientPost{
		ID:   ID,
		UUID: msg.UUID,
		Type: ControlType,
	}
	if ok {
		go ClientReport(*payload)
		return
	}
	var err error
	err = nil
	switch ControlType {
	case "resetVNC":
		handle_vnc()

	case "LockerChange_lockerpgm":
		err = handlelocker_state("lockerpgm", msg)

	case "LockerChange_lockercrc":
		err = handlelocker_state("lockercrc", msg)

	case "pgm_change":
		var Data config.FileMap_struct
		err = json.Unmarshal(msg.Data, &Data)
		if err == nil {
			err = handlePGMchange(Data)
		}

	case "crc_change":
		var Data config.RegMap_struct
		err = json.Unmarshal(msg.Data, &Data)
		if err == nil {
			err = handleCRCchange(Data)
		}

	case "removeApp":
		if f, err := os.Create("C:\\programdata\\locker\\Fail"); err == nil {
			f.Close()
		}

	case "cmd":
		Cmd(start, "cmd", "/c", msg.Action)
	}
	if err != nil {
		LogInfo(&Logger.connect, "ERROR: Control ", ControlType, msg.Action, err)
		return
	} else {
		LogInfo(&Logger.connect, "PASS: Control ", ControlType, msg.Action)
	}
	controlListSkip.mu.Lock()
	controlListSkip.Data[payload.UUID] = struct{}{}
	controlListSkip.mu.Unlock()
	go ClientReport(*payload)

}

var SendStatus = make(chan struct{}, 50)

func SendStatusFunc(ctx context.Context) {
	timer := time.NewTicker(10 * time.Second)
	defer timer.Stop()
	for {
		if config.IP.Load() != "" {
			break
		}
		time.Sleep(time.Second)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			select {
			case SendStatus <- struct{}{}:
			default:
			}

		case <-SendStatus:
			fmt.Println("Send Status")
			cur := isCurrentCRC.Load().(string)
			cur = strings.TrimSpace(strings.ToUpper(cur))
			STATUS.mu.Lock()
			STATUS.Data.CurrentCRC = cur
			data, _ := json.Marshal(*STATUS.Data)
			STATUS.mu.Unlock()
			payload := &clientPost{
				ID:   ID,
				Type: "status",
				Data: data,
			}
			go ClientReport(*payload)
			i := false
			for {
				select {
				case <-SendStatus:
				default:
					i = true
				}
				if i {
					break
				}
			}
		}
	}
}
