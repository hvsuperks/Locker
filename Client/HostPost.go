package main

import (
	"Locker/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

var UDPlistenState = atomic.Value{}
var HostpotPath = `C:\programdata\locker\hostpot`

type udpMsgStruct struct {
	IP  string    `json:"ip"`
	Msg udpStruct `json:"msg"`
}

func SendtoHostPost() {
	for {
		time.Sleep(time.Second * 2)
		if time.Since(UDPlistenState.Load().(time.Time)) > 14*time.Second {
			url := "http://192.168.137.1:50001/api/getUDP"
			var httpClient = &http.Client{
				Timeout: 5 * time.Second,
			}
			resp, err := httpClient.Get(url)
			if err != nil {
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
