package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type WatcherManager struct {
	watcher *fsnotify.Watcher
	watched map[string]bool
	mu      sync.Mutex
}

var lastRenameOldPath string
var listUploadDone = map[string]bool{}
var listUploadWait = map[string]bool{}
var chetaoMap map[string]map[string][]string
var mutexDone, mutexWait sync.RWMutex
var movelist = map[string]string{}
var mu sync.RWMutex

func (wm *WatcherManager) StartMonitoring(rootFolder string) {
	defer func() {
		if err := recover(); err != nil {
			LogUnique("bắt được panic:" + err.(string))
		}
	}()
	var err error
	for {
		wm.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			LogUnique("Không thể khởi tạo Watcher:", err)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}
	defer wm.watcher.Close()
	defer LogUnique("StartMonitoring Stop")
	LogUnique("Đang khởi tạo danh sách thư mục...")
	go wm.watchRecursive(rootFolder)
	done := make(chan bool)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				LogUnique("bắt được panic:" + err.(string))
			}
		}()
		defer LogUnique("wm.watcher.Events Stop")
		movewait := time.NewTimer(2 * time.Second)
		movewait.Stop()
		for {
			select {
			case event, ok := <-wm.watcher.Events:
				if !ok {

					return
				}

				// ==========================================
				// KHỐI LỆNH XỬ LÝ TẠO MỚI (CREATE)
				// ==========================================

				if event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
					createChan <- event.Name
				} else if event.Has(fsnotify.Remove) {
					deleteChan <- event.Name
				}
			case err, ok := <-wm.watcher.Errors:
				if !ok {
					return
				}
				LogUnique("Lỗi giám sát:", err)
			}
		}
	}()

	LogUnique("=== HỆ THỐNG ĐỒNG BỘ AUTO-CONVERT CHẠY TẠI: ", rootFolder, `===`)
	<-done
}

// watcher các folder
func (wm *WatcherManager) watchRecursive(root string) error {
	defer func() {
		if err := recover(); err != nil {
			LogUnique("bắt được panic:" + err.(string))
		}
	}()
	wm.mu.Lock()
	defer wm.mu.Unlock()
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(strings.ToLower(info.Name()), "ok") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			if wm.watched[path] {
				return nil
			}
			if wm.watcher.Add(path) == nil {
				wm.watched[path] = true
			}
			LogUnique("-> Đang giám sát thư mục:", path)
		}
		return nil
	})
}

func GetLatestNGImagePath(searchFolder string) (string, error) {
	// Đoạn script PowerShell thần thánh: Quét file -> Lọc tên & ngày tháng -> Sắp xếp -> Lấy thằng đầu tiên
	psCommand := `Get-ChildItem -Path "` + searchFolder + `" -Recurse -File | Where-Object {
			$_.Extension -match '\.(jpg|png|bmp)$' -and
			$_.FullName -match '\d{4}\\\d{1,2}\\\d{1,2}' -and
			$_.FullName -match 'ng' -and
			$_.FullName -match 'screencapture' -and
			$_.FullName -notmatch '^` + strings.ReplaceAll(syncFolder, `\`, `\\`) + `\\'
			} |
		Sort-Object LastWriteTime -Descending |
		Select-Object -First 1 -ExpandProperty FullName`

	// Gọi PowerShell chạy ngầm (ẩn cửa sổ đen console)
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-Command", psCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	LogUnique(psCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// Chạy và đợi kết quả
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("lỗi thực thi PowerShell: %v, chi tiết: %s", err, out.String())
	}
	// Xử lý chuỗi kết quả (Xóa khoảng trắng thừa)
	path := strings.TrimSpace(out.String())

	re := regexp.MustCompile(`(.*?\\\d{4})\\`)
	match := re.FindStringSubmatch(path)
	if len(match) > 1 {
		return filepath.Dir(match[1]), nil
	} else {
		LogUnique(path)
		os.Exit(0)
	}
	return "", nil
}

// Đồng bộ khi app khởi động
func (wm *WatcherManager) syncUpload() {
	mutexWait.Lock()
	defer mutexWait.Unlock()
	for {
		time.Sleep(time.Second)
		cutoff := time.Now().AddDate(0, 0, -30)
		filepath.Walk(syncFolder, func(path string, fs os.FileInfo, err error) error {
			if fs == nil {
				return nil
			}
			if fs.ModTime().Before(cutoff) {
				if fs.IsDir() {
					os.RemoveAll(path)
					return filepath.SkipDir
				}
				os.Remove(path)
			}
			if fs.IsDir() {
				return nil
			}
			if filepath.Ext(fs.Name()) == ".jpg" {
				rel, _ := filepath.Rel(syncFolder, path)
				targetFile := filepath.Join(RootFolder, rel)
				f, err := os.Stat(targetFile)
				if err != nil || f.Size() < 10 {
					if fs.Size() < 10 {
						os.Remove(path)
						return nil
					}
					wm.CopyJPGFull(path, targetFile)
				}
				_, ok := listUploadDone[path]
				if ok {
					LogUnique("listUploadDone Exits", path)
					return nil
				}
				if isValidFolder(filepath.Base(filepath.Dir(path))) || isValidFolder(filepath.Base(filepath.Dir(filepath.Dir(path)))) {
					if strings.Contains(strings.ToLower(path), "ng") {
						listUploadWait[path] = true
						LogUnique("listUploadWait Add", path)
					}
				}
			}

			return nil
		})
		LogUnique("filepath.Walk()", RootFolder)
		filepath.Walk(RootFolder, func(path string, fs os.FileInfo, err error) error {
			if fs == nil {
				return err
			}
			if fs.IsDir() {
				if strings.Contains(strings.ToLower(fs.Name()), "ok") {
					return filepath.SkipDir
				} else {
					return nil
				}
			}
			ca := strings.ToLower(filepath.Ext(fs.Name()))
			switch ca {
			case ".jpg":
				rel, _ := filepath.Rel(RootFolder, path)
				syncPath := filepath.Join(syncFolder, rel)
				f, err := os.Stat(syncPath)
				if err == nil {
					if f.Size() < 10 {
						os.Remove(syncPath)
					} else {
						return nil
					}
				}

				wm.CopyJPGFull(path, syncPath)
				_, ok := listUploadDone[syncPath]
				if ok {
					LogUnique("listUploadDone Exits", syncPath)
					return nil
				}
				if isValidFolder(filepath.Base(filepath.Dir(syncPath))) || isValidFolder(filepath.Base(filepath.Dir(filepath.Dir(syncPath)))) {
					if strings.Contains(strings.ToLower(path), "ng") {
						listUploadWait[syncPath] = true
						LogUnique("listUploadWait Add", syncPath)
					}
				}
			case ".png", ".bmp", ".raw":
				convertToJPG(path, 90)
			}
			return nil
		})
		if len(listUploadWait) == 0 {
			return
		}
		uploadAOI()
	}
}

var createChan = make(chan string, 100)
var deleteChan = make(chan string, 100)

func (wm *WatcherManager) SyncRealtimeFile(rootFolder string) {
	defer func() {
		if err := recover(); err != nil {
			LogUnique("bắt được panic:" + err.(string))
		}
	}()
	for {
		select {
		case evenPath := <-createChan:
			LogUnique("createChan", evenPath)
			if strings.HasSuffix(evenPath, ".tmp") {
				continue
			}
			LogUnique(evenPath)
			rel, err := filepath.Rel(rootFolder, evenPath)
			if err != nil {
				continue
			}
			fi, err := os.Stat(evenPath)
			if err != nil {
				LogUnique("[LỖI] os.Stat thất bại cho ", evenPath, err)
				continue
			}

			if fi.IsDir() {
				if strings.Contains(strings.ToLower(fi.Name()), "ok") {
					continue
				}
				go wm.watchRecursive(evenPath)
				continue
			}
			if fi.Size() < 10 {
				continue
			}
			ext := strings.ToLower(filepath.Ext(evenPath))
			if ext == ".png" || ext == ".bmp" {
				convertToJPG(evenPath, 90)
				continue // Thoát ra, không đồng bộ file .png/.bmp này sang syncFolder
			}

			if ext == ".jpg" || ext == ".jpeg" {
				destPath := filepath.Join(syncFolder, rel)

				go func(src, dest, rel string) {
					mu.Lock()
					movelist[filepath.Base(rel)] = src
					mu.Unlock()

					wm.CopyJPGFull(src, dest)
					LogUnique("[THÀNH CÔNG] Đã đồng bộ file JPG: ", dest)
				}(evenPath, destPath, rel)
			}

		case evenPath := <-deleteChan:
			rel, err := filepath.Rel(rootFolder, evenPath)
			if err != nil {
				LogUnique(err)
				continue
			}
			syncPath := filepath.Join(syncFolder, rel)
			fi, err := os.Stat(syncPath)
			if err != nil || (fi.Size() < 10 && !fi.IsDir()) {
				LogUnique(err)
				os.Remove(syncPath)
				continue // Bên backup không có thì thôi
			}
			// delay check
			go func(syncPaths, rPath string) {
				defer func() {
					if err := recover(); err != nil {
						LogUnique("bắt được panic:" + err.(string))
					}
				}()
				time.Sleep(100 * time.Millisecond)
				nameFile := filepath.Base(rPath)
				mu.RLock()
				tmpPath, ok := movelist[nameFile]
				mu.RUnlock()
				if !ok || tmpPath == rPath {
					LogUnique("Restore :", rPath)
					if fi.IsDir() {
						go wm.CopyJPGFull(syncPaths, rPath)
					} else {
						filepath.Walk(syncPaths, func(path string, info fs.FileInfo, err error) error {
							if info.IsDir() {
								return nil
							}
							re, _ := filepath.Rel(path, syncPaths)
							wm.CopyJPGFull(path, filepath.Join(rPath, re))
							return nil
						})

					}
				} else {
					LogUnique("Remove :", syncPath)
					os.RemoveAll(syncPath)
				}

			}(syncPath, evenPath)
		}
	}
}

func (wm *WatcherManager) CopyJPGFull(src, dst string) error {
	in, err := os.Open(src)

	if err != nil {
		return err
	}
	defer in.Close()
	dst_folder := filepath.Dir(dst)
	os.MkdirAll(dst_folder, 0755)
	if strings.Contains(dst, RootFolder) {
		go wm.watchRecursive(filepath.Dir(dst_folder))
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err != nil {
		out.Close()
		return err
	}
	out.Close()
	// copy time
	info, err := os.Stat(src)
	if err == nil {
		os.Chtimes(dst, info.ModTime(), info.ModTime())
	}
	return nil
}

func isValidFolder(name string) bool {
	if strings.Contains(strings.ToLower(name), "master") {
		return true
	}
	re := regexp.MustCompile(`^\d{1,2}h\d{1,2}$`)
	return re.MatchString(name)
}
