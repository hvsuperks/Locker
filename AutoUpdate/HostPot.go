package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var srv *http.Server
var UDPlistenState = atomic.Value{}
var mulisten = atomic.Bool{}
var UDPrepeatMSG = udpStruct{}
var muUDPrepeatMSG = sync.RWMutex{}
var isHostPotState = atomic.Bool{}
var WifiHostPot = `WifiHostPot`
var WifiHostPotState = atomic.Bool{}

type udpMsgStruct struct {
	IP  string    `json:"ip"`
	Msg udpStruct `json:"msg"`
}

func StartServer(mux *http.ServeMux) {
	if !mulisten.CompareAndSwap(false, true) {
		return
	}
	defer mulisten.Store(false)
	srv = &http.Server{
		Addr:    ":" + MasterPort,
		Handler: mux,
	}
	srv.ListenAndServe()
}

func StopServer() {
	defer RecoverLog()
	ps := `$connectionProfile = [Windows.Networking.Connectivity.NetworkInformation,Windows,ContentType=WindowsRuntime]::GetInternetConnectionProfile();
$tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager,Windows,ContentType=WindowsRuntime]::CreateFromConnectionProfile($connectionProfile);
$tetheringManager.StopTetheringAsync()`
	Cmd(Cmd_run,
		"powershell",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-Command", ps,
	)
	if mulisten.Load() {
		srv.Shutdown(context.Background())
	}
}

func RenameLanCard() {
	ps := `
Get-NetAdapter |
Where-Object {$_.ifIndex -eq ((Get-NetIPAddress -AddressFamily IPv4 |
Where-Object {$_.IPAddress -like "10.1.*"} |
Select-Object -First 1).InterfaceIndex)} |
Rename-NetAdapter -NewName "LAN"
`
	Cmd(Cmd_run,
		"powershell",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-Command", ps,
	)
}

func restart_network() {
	RenameLanCard()
	ifaces, _ := net.Interfaces()

	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				ip := ipnet.IP.To4()
				if ip != nil && ip[0] == 10 && ip[1] == 1 {
					out, _ := Cmd(Cmd_out, `netsh`, `interface`, `set`, `interface`, fmt.Sprintf(`name="%s"`, iface.Name), "admin=disable")
					LogInfo(Cmd_out, fmt.Sprintf(`netsh interface set interface  name="%s" admin=disable → %s`, iface.Name, out))
					out, _ = Cmd(Cmd_run, `netsh`, `interface`, `set`, `interface`, fmt.Sprintf(`name="%s"`, iface.Name), "admin=enable")
				}
			}
		}
	}
}

func HostpotManager() {
	mux := http.NewServeMux() // Khuyên dùng Mux riêng thay vì http mặc định
	mux.HandleFunc(`GET /api/getUDP`, UDPrepeat)
	OpenPort(`Hostpot`, MasterPort, "TCP")
	connect := false
	for {
		time.Sleep(15 * time.Second)
		func() {
			defer RecoverLog()
			for {
				if time.Since(UDPlistenState.Load().(time.Time)) < 14*time.Second {
					connect = false
					if f, err := os.Stat(HostpotPath); err == nil && !f.IsDir() {
						go StartServer(mux)
						s, _ := HotspotState()
						if strings.ToLower(strings.TrimSpace(s)) == `on` {
							WifiHostPotState.Store(true)
							time.Sleep(time.Second * 5)
							continue
						} else {
							WifiHostPotState.Store(false)
						}

						err, ok := HostUpdate(IP.Load().(string))
						if err != nil {
							LogInfo(err)
							time.Sleep(time.Second * 5)
							continue
						}
						if ok {
							restart_network()
						}
						Cmd(Cmd_run, `ipconfig`, ` /flushdns`)
						time.Sleep(15 * time.Second)
						UDPlistenState.Store(time.Now())
						MobileHotspot(IDcurrent.Load().(string), "12345678")
						ps := `$connectionProfile = [Windows.Networking.Connectivity.NetworkInformation,Windows,ContentType=WindowsRuntime]::GetInternetConnectionProfile();
$tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager,Windows,ContentType=WindowsRuntime]::CreateFromConnectionProfile($connectionProfile);
$tetheringManager.StartTetheringAsync()`

						out, _ := Cmd(Cmd_out,
							"powershell",
							"-NoProfile",
							"-ExecutionPolicy", "Bypass",
							"-Command", ps,
						)
						status := strings.TrimSpace(out)
						LogInfo(`Bật Wifi`, status)
						time.Sleep(time.Second)

					} else {
						StopServer()
						if WifiHostPotState.Load() {
							_, err := Cmd(Cmd_start, `netsh`, `interface`, `set`, `interface`, `name="Wi-Fi"`, `admin=disabled`)
							if err == nil {
								WifiHostPotState.Store(false)
							}
						}
					}
					time.Sleep(2 * time.Second)
				} else {
					StopServer()
					if WifiHostPotState.Load() {
						_, err := Cmd(Cmd_start, `netsh`, `interface`, `set`, `interface`, `name="Wi-Fi"`, `admin=disabled`)
						if err == nil {
							WifiHostPotState.Store(false)
						}
					}
					if !connect {
						if err := ConnectWifi(); err != nil {
							time.Sleep(time.Second)
							LogInfo(err)
							continue
						}
						connect = true
					}
					os.Remove(HostpotPath)
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
						LogInfo(url, err, resp.StatusCode)
						time.Sleep(time.Second)
						continue
					}
					err = json.Unmarshal(b, &result)
					if err != nil {
						LogInfo(url, resp.StatusCode, "-", string(b), "-", err)
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
					UDPlistenState.Store(time.Now())
					if UDPrepeatMSG != result.Msg {
						muUDPrepeatMSG.Lock()
						UDPrepeatMSG = result.Msg
						muUDPrepeatMSG.Unlock()
					}
				}
				time.Sleep(time.Second * 3)
			}
		}()
	}
}

func UDPrepeat(w http.ResponseWriter, r *http.Request) {
	muUDPrepeatMSG.RLock()
	defer muUDPrepeatMSG.RUnlock()
	msg := udpMsgStruct{
		IP:  IP.Load().(string),
		Msg: UDPrepeatMSG,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	return s
}

func HostUpdate(id string) (error, bool) {
	hostText := fmt.Sprintf(`# Copyright (c) 1993-2009 Microsoft Corp.
#
# This is a sample HOSTS file used by Microsoft TCP/IP for Windows.
#
# This file contains the mappings of IP addresses to host names. Each
# entry should be kept on an individual line. The IP address should
# be placed in the first column followed by the corresponding host name.
# The IP address and the host name should be separated by at least one
# space.
#
# Additionally, comments (such as these) may be inserted on individual
# lines or following the machine name denoted by a '#' symbol.
#
# For example:
#
#      102.54.94.97     rhino.acme.com          # source server
#       38.25.63.10     x.acme.com              # x client host

# localhost name resolution is handled within DNS itself.
#	127.0.0.1       localhost
#	::1             localhost
%s www.msftconnecttest.com
%s ipv6.msftconnecttest.com
%s www.msftncsi.com
131.107.255.255 dns.msftncsi.com
`, id, id, id)

	path := `C:\Windows\System32\drivers\etc\hosts`

	oldData, err := os.ReadFile(path)
	if err == nil && normalize(string(oldData)) == normalize(hostText) {
		// Giống hệt rồi thì khỏi ghi
		return nil, false
	}
	return os.WriteFile(path, []byte(hostText), 0644), true
}

type Wifi struct {
	SSID   string
	Signal int
}

var (
	ssidRegex   = regexp.MustCompile(`^\d{9}$`)
	signalRegex = regexp.MustCompile(`(\d+)%`)
)

func ScanWifi() ([]Wifi, error) {
	Cmd(Cmd_run,
		"netsh",
		"interface",
		"set",
		"interface",
		"Wi-Fi",
		"enable",
	)
	out, err := Cmd(Cmd_out,
		"netsh",
		"wlan",
		"show",
		"networks",
		"mode=bssid",
	)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	var result []Wifi
	var currentSSID string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// SSID
		if strings.HasPrefix(line, "SSID ") {
			if idx := strings.Index(line, ":"); idx != -1 {
				currentSSID = strings.TrimSpace(line[idx+1:])
				if !ssidRegex.MatchString(currentSSID) {
					currentSSID = ""
				}
			}
			continue
		}
		if currentSSID == "" {
			continue
		}
		// Signal
		if strings.Contains(line, "Signal") {
			match := signalRegex.FindStringSubmatch(line)
			if len(match) == 2 {
				signal, _ := strconv.Atoi(match[1])
				result = append(result, Wifi{
					SSID:   currentSSID,
					Signal: signal,
				})
				currentSSID = ""
			}
		}
	}
	return result, nil
}

func ConnectWifi() error {
	out, _ := Cmd(
		Cmd_out,
		"netsh", "wlan", "show", "interfaces",
	)
	s := strings.ToLower(out)
	if strings.Contains(s, "connected") || strings.Contains(s, "연결됨") {
		return nil
	}
	LogInfo(out)
	Cmd(Cmd_start,
		"netsh", "interface", "set", "interface",
		`name="Wi-Fi"`,
		"admin=enable",
	)
	wifis, err := ScanWifi()
	if err != nil {
		return err
	}
	if len(wifis) == 0 {
		return fmt.Errorf("không tìm thấy wifi ID 9 số")
	}
	sort.Slice(wifis, func(i, j int) bool {
		return wifis[i].Signal > wifis[j].Signal
	})
	for _, wifi := range wifis {
		xml := fmt.Sprintf(`<?xml version="1.0"?>
<WLANProfile xmlns="http://www.microsoft.com/networking/WLAN/profile/v1">
	<name>%s</name>
	<SSIDConfig>
		<SSID>
			<name>%s</name>
		</SSID>
	</SSIDConfig>
	<connectionType>ESS</connectionType>
	<connectionMode>auto</connectionMode>
	<MSM>
		<security>
			<authEncryption>
				<authentication>WPA2PSK</authentication>
				<encryption>AES</encryption>
				<useOneX>false</useOneX>
			</authEncryption>
			<sharedKey>
				<keyType>passPhrase</keyType>
				<protected>false</protected>
				<keyMaterial>12345678</keyMaterial>
			</sharedKey>
		</security>
	</MSM>
</WLANProfile>`, wifi.SSID, wifi.SSID)
		xmlPath := filepath.Join(os.TempDir(), wifi.SSID+".xml")
		if err := os.WriteFile(xmlPath, []byte(xml), 0644); err != nil {
			continue
		}
		Cmd(Cmd_run,
			"netsh", "wlan", "add",
			"profile",
			fmt.Sprintf(`filename=%s`, xmlPath),
		)
		out, err := Cmd(
			Cmd_out,
			"netsh", "wlan", "connect",
			fmt.Sprintf("name=%s", wifi.SSID),
		)
		if err != nil {
			LogInfo(wifi.SSID, out)
			continue
		}
		time.Sleep(5 * time.Second)
		out, _ = Cmd(
			Cmd_out,
			"netsh", "wlan", "show", "interfaces",
		)
		s := strings.ToLower(out)
		if strings.Contains(s, strings.ToLower(wifi.SSID)) &&
			(strings.Contains(s, "connected") || strings.Contains(s, "연결됨")) {
			LogInfo(wifi.SSID, wifi.Signal, "Connect OK")
			return nil
		}
	}

	return fmt.Errorf("không kết nối được wifi nào")
}

func CurrentSSID() (string, bool) {
	s, err := Cmd(Cmd_out,
		"netsh",
		"wlan",
		"show",
		"interfaces",
	)
	if err != nil {
		return "", false
	}
	if !strings.Contains(strings.ToLower(s), "connected") {
		return "", false
	}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SSID") &&
			!strings.HasPrefix(line, "SSID name") {
			idx := strings.Index(line, ":")
			if idx > 0 {
				return strings.TrimSpace(line[idx+1:]), true
			}
		}
	}
	return "", false
}

func MobileHotspot(ssid, pass string) error {
	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Runtime.WindowsRuntime

function Await {
    param($AsyncOperation)

    $asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() |
        Where-Object {
            $_.Name -eq "AsTask" -and
            $_.IsGenericMethod -and
            $_.GetParameters().Count -eq 1
        } |
        Select-Object -First 1)

    $resultType = $AsyncOperation.GetType().GenericTypeArguments[0]

    $method = $asTaskGeneric.MakeGenericMethod($resultType)

    $task = $method.Invoke($null, @($AsyncOperation))

    $task.Wait()

    return $task.Result
}

try {

    $SSID = %s
    $PASS = %s

    $profile = [Windows.Networking.Connectivity.NetworkInformation,Windows,ContentType=WindowsRuntime]::GetInternetConnectionProfile()

    if ($null -eq $profile) {
        Write-Error "No internet connection profile found."
        exit 1
    }

    $tm = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager,Windows,ContentType=WindowsRuntime]::CreateFromConnectionProfile($profile)

    if ($null -eq $tm) {
        Write-Error "Cannot create tethering manager."
        exit 1
    }

    Write-Host "Current state:" $tm.TetheringOperationalState

    $config = $tm.GetCurrentAccessPointConfiguration()

    $config.Ssid = $SSID
    $config.Passphrase = $PASS

    Write-Host "Applying config..."

    Await ($tm.ConfigureAccessPointAsync($config)) | Out-Null

    Write-Host "SSID:" $SSID
    Write-Host "PASS:" $PASS

    Write-Host "Starting hotspot..."

    $result = Await ($tm.StartTetheringAsync())

    Write-Host "Status:" $result.Status

    if ($result.AdditionalErrorMessage) {
        Write-Host "Message:" $result.AdditionalErrorMessage
    }

    if ($result.Status -ne "Success") {
        exit 1
    }

    exit 0
}
catch {
    Write-Error $_.Exception.Message
    exit 1
}

`, ssid, pass)

	out, err := exec.Command(
		"powershell",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-Command", ps,
	).CombinedOutput()

	LogInfo(fmt.Sprintf("err=%v", err))
	LogInfo(fmt.Sprintf("out=%s", out))

	return err
}

func HotspotState() (string, error) {
	ps := `
Add-Type -AssemblyName System.Runtime.WindowsRuntime

$cp = [Windows.Networking.Connectivity.NetworkInformation,Windows,ContentType=WindowsRuntime]::GetInternetConnectionProfile()
$tm = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager,Windows,ContentType=WindowsRuntime]::CreateFromConnectionProfile($cp)

Write-Output $tm.TetheringOperationalState
`

	out, err := Cmd(Cmd_out,
		"powershell",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-Command", ps,
	)
	return strings.TrimSpace(out), err
}
