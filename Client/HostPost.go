package main

import (
	"Locker/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var srv *http.Server
var isHostPot = atomic.Value{}
var mulisten = atomic.Bool{}
var UDPrepeatMSG = udpStruct{}
var muUDPrepeatMSG = sync.RWMutex{}

func StartServer(mux *http.ServeMux) {
	if !mulisten.CompareAndSwap(false, true) {
		return
	}
	defer mulisten.Store(false)
	srv = &http.Server{
		Addr:    ":" + config.MasterPort,
		Handler: mux,
	}

	srv.ListenAndServe()
}

func StopServer() {
	defer RecoverLog()
	if mulisten.Load() {
		srv.Shutdown(context.Background())
	}
}

func SendtoHostPost() {
	time.Sleep(15 * time.Second)
	mux := http.NewServeMux() // Khuyên dùng Mux riêng thay vì http mặc định
	mux.HandleFunc(`GET /api/getUDP`, UDPrepeat)
	OpenUDP10001(`UDP_Hostport`, config.MasterPort, "TCP")
	for {
		t := isHostPot.Load().(time.Time)
		if time.Since(t) < 14*time.Second {
			go StartServer(mux)
			time.Sleep(2 * time.Second)
		} else {
			StopServer()
			url := "http://192.168.137.1:50001/api/getUDP"

			var httpClient = &http.Client{
				Timeout: 5 * time.Second,
			}

			resp, err := httpClient.Get(url)
			if err != nil {
				LogInfo(&Logger.debug, err)
				time.Sleep(time.Second)
				continue
			}
			defer resp.Body.Close()
			var result udpMsgStruct
			b, er := io.ReadAll(resp.Body)
			if er == nil {
				LogInfo(&Logger.debug, string(b))
			}
			err = json.Unmarshal(b, &result)
			if err != nil {
				LogInfo(&Logger.debug, resp.StatusCode, err)
				time.Sleep(time.Second)
				continue
			}
			ip := result.IP
			if config.IP.Load().(string) != ip {
				config.IP.Store(ip)
				LogInfo(&Logger.debug, `UDP IP :`, ip)
				httpDownload.Store(fmt.Sprintf("http://%s:%s/master/", ip, config.MasterPort))
				httpUpload.Store(fmt.Sprintf("http://%s:%s/api/uploadFile", ip, config.MasterPort))
			}
			AutoUpdateChange <- result.Msg.AutoUpdate
		}

	}
}

type udpMsgStruct struct {
	IP  string    `json:"ip"`
	Msg udpStruct `json:"msg"`
}

func UDPrepeat(w http.ResponseWriter, r *http.Request) {
	muUDPrepeatMSG.RLock()
	defer muUDPrepeatMSG.RUnlock()
	msg := udpMsgStruct{
		IP:  config.IP.Load().(string),
		Msg: UDPrepeatMSG,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}
