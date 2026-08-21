package main

import (
	"Locker/config"
	"Locker/config/key"
	"encoding/json"
	"net"
	"strings"
	"time"
)

var msg struct {
	Locker     string `json:"locker"`
	UiLocker   string `json:"uilocker"`
	Aoi        string `json:"aoi"`
	UiAoi      string `json:"uiaoi"`
	AutoUpdate string `json:"update"`
}

func udpBroadcast() {
	// 1. Lấy danh sách tất cả các card mạng
	interfaces, err := net.Interfaces()
	if err != nil {
		Fmt(fmtMaster, "udpBroadcast", "Không thể lấy danh sách card mạng:", err)
		return
	}

	for {
		msg.Locker = config.MasterLocker
		msg.UiLocker = config.MasterUILocker
		msg.Aoi = config.MasterAOI
		msg.UiAoi = config.MasterUIAoi
		msg.AutoUpdate = config.MasterService
		msgsend, _ := json.Marshal(msg)
		//msgLocker := "Master:" + config.MasterLocker
		msgAOI := "Master:" + config.MasterAOI
		for _, iface := range interfaces {
			// Chỉ gửi qua các card đang chạy (Up) và không phải là Loopback (127.0.0.1)
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}

			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if !ok || ipnet.IP.To4() == nil {
					continue
				}

				// 2. Tính toán địa chỉ Broadcast của card mạng này
				// Ví dụ: IP 192.168.1.5/24 => Broadcast 192.168.1.255
				broadcastIP := getBroadcastIP(ipnet)

				targetAddrLocker := &net.UDPAddr{
					IP:   broadcastIP,
					Port: 9999,
				}

				targetAddrAOI := &net.UDPAddr{
					IP:   broadcastIP,
					Port: 9998,
				}

				targetAddrLockerUi := &net.UDPAddr{
					IP:   broadcastIP,
					Port: 50000,
				}

				targetAddrLockerUi_module9 := &net.UDPAddr{
					IP:   broadcastIP,
					Port: 10001,
				}
				targetAddrUpdate_module9 := &net.UDPAddr{
					IP:   broadcastIP,
					Port: 10000,
				}

				// 3. Gửi gói tin

				if config.MasterAOI != "" {
					sendUDP(targetAddrAOI, msgAOI)
				}
				if config.MasterAOI != "" {
					sendUDP(targetAddrLocker, msgAOI)
				}
				if config.MasterLocker != "" && config.MasterUILocker != "" {
					sendUDP(targetAddrLockerUi, config.MasterUILocker+"|"+config.MasterLocker)
				}
				if config.MasterLocker != "" && config.MasterUILocker != "" && config.MasterService != "" {
					sendUDP_module9(targetAddrLockerUi_module9, msgsend)
					sendUDP_module9(targetAddrUpdate_module9, msgsend)
				}
			}

		}
		time.Sleep(3 * time.Second)
	}
}

var buf = make([]byte, 0, 1024)

// Hàm hỗ trợ gửi UDP đơn lẻ
func sendUDP(addr *net.UDPAddr, msg string) {

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()
	b := append(buf[:0], msg...) // reuse buffer
	conn.Write(b)
}

func sendUDP_module9(addr *net.UDPAddr, msg []byte) {

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.Write(msg)
}

// Hàm tính toán Broadcast IP từ IPNet
func getBroadcastIP(ipnet *net.IPNet) net.IP {

	ip := ipnet.IP.To4()
	mask := ipnet.Mask
	broadcast := make(net.IP, len(ip))
	for i := 0; i < len(ip); i++ {
		broadcast[i] = ip[i] | ^mask[i]
	}
	return broadcast
}

func LanCard(bind, color chan config.KV) {

	timer := time.NewTicker(time.Minute)
	defer timer.Stop()
	for {
		<-timer.C
		var A, B string // A cho 172..., B cho 10...

		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return
		}

		for _, addr := range addrs {
			// Ép kiểu về IPNet để lấy IP
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			ipStr := ip.String()

			// Kiểm tra prefix để gán vào biến
			if strings.HasPrefix(ipStr, "172.") {
				A = ipStr
			} else if strings.HasPrefix(ipStr, "10.") {
				B = ipStr
			}
			bind <- config.KV{
				Key:   key.PgmBind,
				Value: A,
			}
			bind <- config.KV{
				Key:   key.CrcBind,
				Value: B,
			}
		}
	}
}
