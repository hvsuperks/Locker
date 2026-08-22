package main

import (
	ui "Locker/Ui"
	"Locker/config"
	"Locker/script"
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	_ "net/http/pprof"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/systray"
	"golang.org/x/sys/windows"
)

//go:embed icon.png
var iconBytes []byte

var wg sync.WaitGroup
var Ver = config.Ver

const isDebug = true

type LogType struct {
	Type int
	Data []any
}

var Log = make(chan LogType, 100)
var (
	fmtClient = 0
	fmtAOI    = 1
	fmtMaster = 2
)

func Fmt(Type int, v ...any) {
	if config.IsDebug {
		fmt.Println(v...)
	}
	Log <- LogType{Type: Type, Data: v}
}

var ctx, cancel = context.WithCancel(context.Background())

func main() {
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	Ver = Ver + config.MasterLocker
	config.StartTime = time.Now().Format("2006/01/02 15:04:05")
	config.IsDebug = true
	config.IP.Store("")
	// Khởi tạo context Close App
	lastcheckMap.Store(time.Now())
	defer cancel()

	// Kênh nhận tín hiệu close từ Windows
	sigs := make(chan os.Signal, 10)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	// Tên App
	title := "Master"

	// Đường dẫn của app

	CurrentAppRootPath, _ := os.Executable()
	CurrentAppRootDir := filepath.Dir(CurrentAppRootPath)

	// Đường dẫn tiêu chuẩn
	dstAppRootPath := filepath.Join(config.AppRootDir, config.DstAppName)
	// Kiểm tra app đang mở có phải là setup hay ko
	// Nếu đường dẫn app đang ở ko phải ở thư mục quy định sẽ copy vào khu vực quy định và stop
	if !script.ComparePath(CurrentAppRootDir, config.AppRootDir) {
		script.Cmd("taskkill", "/IM", config.DstAppName)
		for _, i := range []string{"PGM_lock.exe", "watcherdog.exe", "app_loop.exe", "setup_locker"} {
			script.Cmd("taskkill", "/IM", i, "/T", "/F")

		}
		time.Sleep(time.Second)
		os.RemoveAll(config.AppRootDir)
		err := script.CopyFileFull(CurrentAppRootPath, dstAppRootPath)
		if err == nil {
			script.LaunchAdminDetached(dstAppRootPath)
			time.Sleep(time.Millisecond)
			os.Exit(0)
		}
		os.Exit(1)
	}

	//Check App có đang mở hay ko, nếu đang mở thì out luôn
	go check_sign_app(ctx)

	// Mở Port UDP

	//Tạo Log

	// Đọc ID và phân loại máy
	var ID, mode string
	ID = ID_Read()
	newFileMode := false
	if script.RegRead(config.RegPath, "log") != "3" {
		script.RegWrite(config.RegPath, "log", "3")
		os.Remove(`D:\master\log\222255555\2026-07-29\fmtMaster.log`)
		newFileMode = true
	}
	if ID[:4] == "2222" || ID == "" {
		if _, err := os.Stat("D:\\Master"); err == nil {
			mode = "master"
			go Create_logger(filepath.Join(config.LogPath_master, ID), &newFileMode)
		} else {
			os.Exit(1)
		}
	} else {
		return
	}

	// ListenMasterUDP
	Fmt(fmtMaster, "Main", mode)
	script.CreateTask(CurrentAppRootPath, config.DstAppName)
	cb := HideApp()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if CheckForbiddenApp(mode, "setup_master", cb) {
				cancel()
				cleanupAndExit()
			}
			time.Sleep(time.Second)
		}
	}()

	// Tạo Ui App
	myApp := app.New()
	// Tạo theme bỏ viền
	myApp.Settings().SetTheme(&TinyTheme{})
	appIcon := fyne.NewStaticResource("icon.png", iconBytes)
	myApp.SetIcon(appIcon)
	window := myApp.NewWindow(title)
	window.SetCloseIntercept(func() {
		window.Hide()
	})
	// Chạy luồng quản lý khay hệ thống riêng biệt, không dùng SystemTray gốc của Fyne
	go func() {
		systray.Run(onReady(window), nil)
	}()
	// Tạo kênh Change
	resetPackChan := make(chan string, 10)
	bind := make(chan config.KV, 100)
	color := make(chan config.KV, 100)
	content := ui.Manager(window, resetPackChan, Ver, ID, mode, bind, color)
	window.SetContent(content)
	var x float32 = 1400
	var y float32 = 30
	window.Resize(fyne.NewSize(x, y))

	go ontop(ctx, window, title, x, mode)
	var reloadChan = make(chan struct{}, 5)
	// gọi app chính theo mode

	go Start(ctx, Ver, reloadChan, bind, color)
	window.ShowAndRun()
}

func onReady(window fyne.Window) func() {
	return func() {
		// Set icon cho khay (Dùng file ico hoặc byte của bạn)
		// systray.SetIcon(yourIconBytes)
		systray.SetTitle("Hệ Thống")
		systray.SetTooltip("Đang chạy ngầm...")

		// Tạo các item menu mong muốn
		mShow := systray.AddMenuItem("Show App", "Hiện ứng dụng")
		mHide := systray.AddMenuItem("Hide App", "Ẩn ứng dụng")

		// Hoàn toàn KHÔNG tạo nút Quit ở đây

		for {
			select {
			case <-mShow.ClickedCh:
				fyne.DoAndWait(func() {
					window.Show() // Hoặc window.Hide(), hoặc label.SetText(...)
					script.RegWrite(config.RegPath, "show", "show")
				})
			case <-mHide.ClickedCh:
				fyne.DoAndWait(func() {
					window.Hide() // Hoặc window.Hide(), hoặc label.SetText(...)
					script.RegWrite(config.RegPath, "show", "hide")
				})
			}
		}
	}
}

func cleanupAndExit() {
	wdChan := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		close(wdChan)
	}()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-wdChan:
	}
	os.Exit(0)
}

func check_sign_app(ctx context.Context) {
	name := windows.StringToUTF16Ptr("Global\\Locker")
	mutex, err := windows.CreateMutex(nil, false, name)
	defer windows.CloseHandle(mutex)
	if err != nil {
		os.Exit(0)
	}
	// ERROR_ALREADY_EXISTS = 183
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		os.Exit(0)
	}
	<-ctx.Done()
}

func ID_Read() string {

	path := "C:\\Program Files\\uvnc bvba\\UltraVNC\\ultravnc.ini"
	data, err := os.ReadFile(path)
	if err != nil {
		panic("Chưa cài VNC à óc chó?")
	}

	text := string(data)

	re := regexp.MustCompile(`ID:(\d{9})(\D|$)`)

	match := re.FindStringSubmatch(text)

	if len(match) != 3 {
		return ""
	}
	ID := match[1]
	return ID

}

func ontop(ctx context.Context, window fyne.Window, title string, x float32, mode string) {

	for {
		hwnd := script.GetHwndByTitle(title)
		if hwnd != 0 {
			if mode == "client" {
				go script.SetTopmost(ctx, hwnd, x)
				go script.HideFromTaskbar(hwnd)
			}
			window.SetCloseIntercept(func() {
				// Để trống: Công nhân nhấn Alt+F4 sẽ không có tác dụng gì
				// Bạn có thể ghi log tại đây nếu muốn theo dõi
				fmt.Println("Cố gắng đóng ứng dụng nhưng đã bị chặn!")
			})
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Cấu trúc dữ liệu để chứa thông tin tiến trình từ Windows
type ProcessEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16 // Tên file tiến trình
}

func CheckForbiddenApp(mode, app string, cb uintptr) bool {

	hideAppStatus = true
	// 1. Chụp ảnh danh sách các tiến trình hiện tại
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(snapshot)

	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	// 2. Duyệt qua tiến trình đầu tiên
	err = syscall.Process32First(snapshot, &entry)
	mes := false
	user32 := syscall.NewLazyDLL("user32.dll")
	// Khai báo các hàm cần thiết từ DLL
	procEnumWindows := user32.NewProc("EnumWindows")
	for err == nil {
		// Chuyển tên tiến trình từ UTF16 sang String
		name := syscall.UTF16ToString(entry.ExeFile[:])
		name = strings.ToLower(name)
		if strings.Contains(name, strings.ToLower(app)) {
			script.Cmd("taskkill", "/IM", "watcherdog.exe", "/F")
			return true // Thấy app cấm phát là báo ngay
		}

		// Duyệt tiến trình tiếp theo
		err = syscall.Process32Next(snapshot, &entry)
	}

	if mode == "client" {
		if !mes {
			script.Cmd("explorer.exe", "C:\\ProgramData\\Jahwa Electronics\\ECM\\ECM Client Agent\\10.1.1.36\\Jahwa ECM Agent V2\\Jahwa_ECM_Agent_V2.exe")
			time.Sleep(3 * time.Second)
		} else {
			procEnumWindows.Call(cb, 0)
		}
	}
	return false
}

var hideAppStatus bool
var ProcessID uint32

func HideApp() uintptr {

	hideAppStatus = false
	user32 := syscall.NewLazyDLL("user32.dll")
	procGetWindowThreadProcessId := user32.NewProc("GetWindowThreadProcessId") // Thêm dòng này
	procIsWindowVisible := user32.NewProc("IsWindowVisible")
	const SW_MINIMIZE = 6
	procShowWindow := user32.NewProc("ShowWindow")
	cb := syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))

		if pid == ProcessID {
			// Kiểm tra xem cửa sổ có đang hiển thị không (tránh thu nhỏ mấy cái ẩn)
			visible, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
			if visible != 0 {
				procShowWindow.Call(uintptr(hwnd), SW_MINIMIZE)
			}
		}
		return 1 // Tiếp tục duyệt các cửa sổ khác
	})
	return cb
}
