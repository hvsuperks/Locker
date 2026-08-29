package main

import (
	"Locker/config"
	"encoding/json"
	"net"
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
		LogInfo(&Logger.MAIN, "udpBroadcast", "Không thể lấy danh sách card mạng:", err)
		return
	}
	LogInfo(&Logger.MAIN, "udpBroadcast Start")
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
				if !ok {
					continue
				}
				ipv4 := ipnet.IP.To4()
				if ipv4 == nil {
					LogInfo(&Logger.MAIN,
						iface.Name,
						"No IPv4")
					continue
				}
				LogInfo(&Logger.MAIN,
					"IPv4:",
					iface.Name,
					ipv4.String(),
				)
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
					LogInfo(&Logger.MAIN, msgsend)
				} else {
					LogInfo(&Logger.MAIN, "config.MasterLocker: ", config.MasterLocker, " config.MasterUILocker: ", config.MasterUILocker, " config.MasterService: ", config.MasterService)
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
