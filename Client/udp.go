package main

import (
	"Locker/config"
	"Locker/script"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var httpDownload = atomic.Value{}
var VerAutoUpdate = atomic.Value{}
var httpUpload = atomic.Value{}

func ListenMasterUDP(Log chan []any) {
	httpDownload.Store("")
	httpUpload.Store("")
	addr := net.UDPAddr{
		Port: 10001,
		IP:   net.IPv4zero, // nghe toàn bộ
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		var msg struct {
			Locker     string `json:"locker"`
			UiLocker   string `json:"uilocker"`
			Aoi        string `json:"aoi"`
			UiAoi      string `json:"uiaoi"`
			AutoUpdate string `json:"update"`
		}
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}
		ip := remoteAddr.IP.To4().String()
		IP := config.IP.Load().(string)
		if ip != IP {
			config.IP.Store(ip)
			httpDownload.Store(fmt.Sprintf("http://%s:%s/master/", ip, config.MasterPort))
			httpUpload.Store(fmt.Sprintf("http://%s:%s/api/upload", ip, config.MasterPort))
		}
		if msg.AutoUpdate != "" && msg.AutoUpdate != VerAutoUpdate.Load() {
			AutoUpdateChange <- struct{}{}
		}

	}
}

func OpenUDP10001() error {

	// 1. Check rule tồn tại chưa
	checkCmd := exec.Command(
		"netsh",
		"advfirewall",
		"firewall",
		"show",
		"rule",
		"name=Locker_10001",
	)
	checkCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	out, err := checkCmd.Output()
	if err == nil {
		if !strings.Contains(string(out), "No rules match") {
			// Rule đã tồn tại → bỏ qua
			return nil
		}
	}

	// 2. Nếu chưa có → add
	addCmd := exec.Command(
		"netsh",
		"advfirewall",
		"firewall",
		"add",
		"rule",
		"name=Locker_10001",
		"dir=in",
		"action=allow",
		"protocol=UDP",
		"localport=10001",
	)
	addCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	return addCmd.Run()

}

var AutoUpdateChange = make(chan struct{}, 20)

func AutoUpdate(ctx context.Context) {
	ServiceConnectTime.Store(time.Now())
	timer := time.NewTicker(1 * time.Second)
	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-AutoUpdateChange:
			if time.Since(last) > 15*time.Second {
				if script.DownloadFile(httpDownload.Load().(string)+"AutoUpdate.exe", `D:\autoupdate.exe`) == nil {
					Cmd(run, `D:\autoupdate.exe`)
					i := false
					for {
						select {
						case <-AutoUpdateChange:
						default:
							i = true
						}
						if i {
							break
						}
					}
				}
				last = time.Now()
			}
		case <-timer.C:
			if !ServiceStatus.Load() {
				lastoff := ServiceConnectTime.Load().(time.Time)
				if time.Since(lastoff) > 15*time.Second {
					AutoUpdateChange <- struct{}{}
				}
			}
		}
	}
}
