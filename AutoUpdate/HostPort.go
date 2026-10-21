package main

import (
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

func SendtoHostPost() {
	time.Sleep(15 * time.Second)
	defer RecoverLog()
	for {
		t := isHostPot.Load().(time.Time)
		if time.Since(t) > 14*time.Second {
			url := "http://192.168.137.1:50001/api/getUDP"
			httpDownload := ""
			var httpClient = &http.Client{
				Timeout: 5 * time.Second,
			}

			resp, err := httpClient.Get(url)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			defer resp.Body.Close()
			var result struct {
				IP  string    `json:"ip"`
				Msg udpStruct `json:"msg"`
			}
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				Loginfo(url, err, resp.StatusCode)
				time.Sleep(time.Second)
				continue
			}
			err = json.Unmarshal(b, &result)
			if err != nil {
				Loginfo(url, resp.StatusCode, "-", string(b), "-", err)
				time.Sleep(time.Second)
				continue
			}
			ip := result.IP
			httpDownload = fmt.Sprintf("http://%s:%s/master", ip, MasterPort)
			HttpGetCMD.Store(fmt.Sprintf("http://%s:%s/serviceGetCMD?id=", ip, MasterPort))
			if ip != IP.Load().(string) {
				IP.Store(ip)
			}
			checkupdate(result.Msg, httpDownload)
		}
		time.Sleep(time.Second * 3)
	}
}
