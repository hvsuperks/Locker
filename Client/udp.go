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
	LogInfo(&Logger.debug, `UDP Start`)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			LogInfo(&Logger.debug, `UDP Read Fail`, err)
			time.Sleep(time.Second)
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
			LogInfo(&Logger.debug, `UDP Unmarshal Fail`, err)
			continue
		}
		ip := remoteAddr.IP.To4().String()
		IP := config.IP.Load().(string)
		LogInfo(&Logger.debug, `UDP IP :`, ip)
		if ip != IP {
			config.IP.Store(ip)
			LogInfo(&Logger.debug, `UDP IP :`, ip)
			httpDownload.Store(fmt.Sprintf("http://%s:%s/master/", ip, config.MasterPort))
			httpUpload.Store(fmt.Sprintf("http://%s:%s/api/uploadFile", ip, config.MasterPort))
		}
		AutoUpdateChange <- msg.AutoUpdate

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

var AutoUpdateChange = make(chan string, 100)

func AutoUpdate(ctx context.Context) {
	ServiceConnectTime.Store(time.Now())

	for {
		select {
		case <-ctx.Done():
			return
		case v := <-AutoUpdateChange:
			if v != VerAutoUpdate.Load().(string) {
				if script.DownloadFile(fmt.Sprintf(`%sLockerService_%s.exe`, httpDownload.Load().(string), v), `D:\LockerService.exe`) == nil {
					VerAutoUpdate.Store("")
					ServiceStatus.Store(false)
					Cmd(run, `D:\LockerService.exe`)
					for i := range 32 {
						if ServiceStatus.Load() {
							break
						}
						if i >= 30 {
							VerAutoUpdate.Store("")
							break
						}
						time.Sleep(100 * time.Millisecond)
					}
					for {
						select {
						case <-AutoUpdateChange:
						default:
							goto done
						}
					}
				done:
				}
			}
		}
	}

}
