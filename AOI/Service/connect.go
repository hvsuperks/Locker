package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/Microsoft/go-winio"
)

func pipeStart(appname string) {
	LastPing.Store("")
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					LogUnique(fmt.Sprintf("panic: %v", r))
					time.Sleep(time.Second)
				}
			}()
			listener, err := winio.ListenPipe(`\\.\pipe\`+appname, nil)
			if err != nil {
				LogUnique("ERROR: Pipe", `\\.\pipe\`+appname, err)
				time.Sleep(time.Millisecond * 50)
				return
			}
			for {
				conn, err := listener.Accept()
				if err != nil {
					LogUnique("ERROR: Pipe", `\\.\pipe\`+appname, err)
					time.Sleep(time.Millisecond * 50)
					return
				}
				handleConn(conn)

				time.Sleep(time.Second)
			}
		}()
	}
}

var isShutdownLocker = atomic.Bool{}
var LastPing = atomic.Value{}
var sendUI = make(chan sendT, 100)
var IP = atomic.Value{}
var MasterPort = "50001"

type sendT struct {
	Wait  string
	Ver   string
	Room  string
	Model string
	Part  string
}

func handleConn(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			LogUnique(fmt.Sprintf("panic: %v", r))
		}
	}()
	defer conn.Close()
	buf := make([]byte, 4096)
	go func() {
		defer conn.Close()
		defer func() {
			if r := recover(); r != nil {
				LogUnique(fmt.Sprintf("panic: %v", r))
				time.Sleep(time.Second)
			}
		}()
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		for {
			var i []byte
			select {
			case msg := <-sendUI:
				i, _ = json.Marshal(msg)

			case <-timer.C:
				i, _ = json.Marshal(map[string]string{`ver`: Ver})
			}
			_, err := conn.Write(i)
			if err != nil {
				LogUnique(`Pipe Read`, err)
				return
			}
		}
	}()

	for {
		n, err := conn.Read(buf)
		if err != nil {
			LogUnique(`Pipe Write`, err)
			return
		}
		Uiver.Store(string(buf[:n]))
		LastPing.Store(time.Now())
	}
}

func ListenMasterUDP() {
	for {
		defer func() {
			if r := recover(); r != nil {
				LogUnique(fmt.Sprintf("panic: %v", r))
			}
		}()
		HttpGetCMD.Store("")
		addr := net.UDPAddr{
			Port: AutoUpdatePort,
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
			if ip != IP.Load().(string) {
				IP.Store(ip)
				HttpGetCMD.Store(fmt.Sprintf("http://%s:%s/serviceGetCMD?id=", ip, MasterPort))
			}

			if msg.UiAoi != "" && msg.UiAoi != Uiver.Load().(string) {
				go UpdateUi(msg.UiAoi)

			}
			if msg.Aoi != Ver {
				go UpdateService(msg.Aoi)
			}

		}
	}
}

func watcherUi() {
	for {
		t := LastPing.Load().(time.Time)
		if time.Since(t) > 5*time.Second {
			exec.Command(`taskkill`, `/f`, `/im`, `Ui.exe`).Run()
			OpenApp(uiPath)
			time.Sleep(3 * time.Second)
		}
		time.Sleep(time.Second)
	}
}

func uploadImage() {

}
