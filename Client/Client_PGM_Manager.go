package main

import (
	"Locker/config"
	"Locker/script"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (a mClient) pgmManager(pgmRuning *atomic.Bool, LockerChan, Restart_XOIS chan struct{}) {
	defer RecoverLog()
	defer func() {
		pgmRuning.Store(false)

	}()

	files, err := os.ReadDir(config.MasterDir)
	if err != nil {
		os.MkdirAll(config.MasterDir, os.ModePerm)
		return
	}
	for _, filename := range files {
		if strings.ToLower(filepath.Ext(filename.Name())) == ".zip" {
			continue
		}
		os.RemoveAll(filepath.Join(config.MasterDir, filename.Name()))
	}
	pgmRuning.Store(true)
	//Kiểm tra PGM
	tmppgm := STATUS_GET(&STATUS.MasterMap.PGM).(FileMap_struct)
	pgm := tmppgm.PGM
	md5 := tmppgm.MD5
	url := tmppgm.URL
	if pgm == "" || pgm == "Fail" || md5 == "" || url == "" {
		return
	}
	masterZip := filepath.Join(config.MasterDir, pgm+".zip")
	masterXOISdir := filepath.Join(config.MasterDir, pgm)
	if url[:1] == "/" {
		url = url[1:]
	}
	if url[:3] != "PGM" {
		url = "PGM/" + url
	}
	url = httpDownload.Load().(string) + url
	if _, err := os.Stat(masterZip); err != nil {
		isdownload.Store(true)
		err := script.DownloadFile(url, masterZip)
		isdownload.Store(false)
		if err != nil {
			LogInfo(&Logger.debug, url, masterZip, err)
			STATUS_PUT(&STATUS.MasterMap.PGM, FileMap_struct{PGM: "Fail", Color: "Fail"})
		}
		return
	}
	isMd5, err := script.GetMD5(masterZip)
	if err != nil || md5 != isMd5 {
		tmppgm.MD5 = isMd5
		tmppgm.Color = "Fail"
		tmppgm.PGM = pgm
		STATUS_PUT(&STATUS.MasterMap.PGM, tmppgm)
		isdownload.Store(true)
		err := script.DownloadFile(url, masterZip)
		isdownload.Store(false)
		if err != nil {
			LogInfo(&Logger.debug, url, masterZip, err)
			STATUS_PUT(&STATUS.MasterMap.PGM.PGM, "Fail")
		}
		return
	}
	tmppgm.MD5 = isMd5
	tmppgm.Color = "Fail"
	if script.Unzip(masterZip, masterXOISdir) != nil {
		STATUS_PUT(&STATUS.MasterMap.PGM, tmppgm)
		return
	}

	var md5Map = make(map[string]string)
	err = filepath.WalkDir(masterXOISdir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if m, err := script.GetMD5(path); err == nil {
			md5Map[path] = m
			return nil
		} else {
			return err
		}
	})
	if err != nil {
		isdownload.Store(true)
		err := script.DownloadFile(url, masterZip)
		isdownload.Store(false)
		if err != nil {
			LogInfo(&Logger.debug, url, masterZip, err)
			tmppgm.PGM = `Fail`
			tmppgm.Color = `Fail`
			STATUS_PUT(&STATUS.MasterMap.PGM.PGM, "Fail")
		}
		<-a.ctx.Done()
		return
	}
	go a.WatchLock_file(LockerChan, md5Map, []string{".exe", ".dll", ".ini", ".xml"})
	go a.PGM_checkfile(LockerChan, Restart_XOIS, md5Map)
	<-a.ctx.Done()

}

func (a mClient) WatchLock_file(LockerChan chan struct{}, Md5Mon map[string]string, exts []string) {
	defer RecoverLog()
	extMap := make(map[string]struct{})
	for _, e := range exts {
		extMap[strings.ToLower(e)] = struct{}{}
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		LogInfo(&Logger.debug, "watcher error:"+err.Error())
		panic("watcher error:" + err.Error())
	}
	defer watcher.Close()

	err = watcher.Add(config.XoisPath)
	if err != nil {
		LogInfo(&Logger.debug, "watch add error:"+err.Error())
		panic("watch add error:" + err.Error())
	}
	eventChan := make(chan struct{})
	defer close(eventChan)
	go a.WatchDebounce(LockerChan, eventChan)
	for {
		select {
		case <-a.ctx.Done():
			for {
				select {
				case <-watcher.Events:
				default:
					return
				}
			}
		case event := <-watcher.Events:

			// chỉ xử lý khi file bị sửa / tạo
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				ext := strings.ToLower(filepath.Ext(event.Name))
				if _, ok := extMap[ext]; ok {
					LogInfo(&Logger.pgm, "CHANGE:", "File changed:"+event.Name)
					eventChan <- struct{}{}
				}
			}

		case err := <-watcher.Errors:
			LogInfo(&Logger.debug, "PGM watch error:"+err.Error())
			panic("PGM watch error:" + err.Error())
		}
	}
}

func (a mClient) WatchDebounce(LockerChan chan struct{}, eventChan <-chan struct{}) {
	defer RecoverLog()
	timer := time.NewTimer(time.Minute * 5)
	defer timer.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-eventChan:
			if !timer.Stop() {
				select {
				case <-timer.C: // Xả channel nếu timer đã kịp nổ
				default:
				}
			}
			timer.Reset(2 * time.Second)
		case <-timer.C:
			LockerChan <- struct{}{}
		}
	}
}

func (a mClient) PGM_checkfile(LockerChan, Restart_XOIS chan struct{}, Md5Map map[string]string) {
	defer RecoverLog()
	LockerChan <- struct{}{}
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-LockerChan:
			var err_list = []KV{}
			var b = true
			pgm := STATUS_GET(&STATUS.MasterMap.PGM).(FileMap_struct)
			for path, md5master := range Md5Map {
				rel, _ := filepath.Rel(filepath.Join(config.MasterDir, pgm.PGM), path)
				curentXOISFile := filepath.Join(config.XoisPath, rel)
				md5file, err := script.GetMD5(curentXOISFile)
				if err != nil || !strings.EqualFold(md5file, md5master) {
					err_list = append(err_list, KV{Key: rel, Value: md5master})
					b = false
					LogInfo(&Logger.debug, "MD5", md5file, err)
				}
			}
			LogInfo(&Logger.pgm, "error_list", err_list)
			if b {
				STATUS_PUT(&STATUS.MasterMap.PGM, FileMap_struct{PGM: pgm.PGM, Color: "OK", MD5: pgm.MD5, URL: pgm.URL})
				continue
			}

			lockerState := STATUS_GET(&STATUS.Locker_PGM.Value).(string)
			if strings.EqualFold(strings.ToLower(lockerState), "unlock") {
				text := "File Fail :"
				for _, i := range err_list {
					text += "\n" + i.Key + " : " + i.Value
				}
				text += "\n"
				LogInfo(&Logger.pgm, "File Fail"+string(text))
				STATUS_PUT(&STATUS.Locker_PGM, Locker_state{Value: "Unlock", Color: "Fail"})
				continue
			}
			Restart_XOIS <- struct{}{}
			a.UpdatePGM(Restart_XOIS, pgm.PGM, err_list)
			LockerChan <- struct{}{}
		}

	}
}

func (a mClient) UpdatePGM(Restart_XOIS chan struct{}, pgm string, files []KV) bool {
	now := time.Now().Format("2006_01_02_15_04_05")
	var text_log []string
	move_status := true
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for _, f := range files {
		select {
		case <-a.ctx.Done():
			return false
		case <-timer.C:
			Restart_XOIS <- struct{}{}
			timer.Reset(time.Second)
		default:
			src := filepath.Join(config.MasterDir, pgm, f.Key)
			dst := filepath.Join(config.XoisPath, f.Key)
			if !FileExists(src) {
				return false
			}
			md5, err := script.GetMD5(src)
			if err != nil || md5 != f.Value {
				LogInfo(&Logger.debug, src, "File Fail", err)
				return false
			}
			backup := filepath.Join("D:\\PGM_Backup", now, f.Key)
			os.MkdirAll(filepath.Join("D:\\PGM_Backup", now), os.ModePerm)
			if FileExists(dst) {
				err = script.CopyFileFull(dst, backup)
				if err == nil {
					os.Remove(dst)
				} else {
					text_log = append(text_log, dst+" ----- Move Error : "+err.Error())
					move_status = false
				}

				if !move_status {
					text := ""
					for _, i := range text_log {
						text = text + "\n" + i
					}
					LogInfo(&Logger.pgm, "CHANGE:", text)
					return false
				}
				timer.Reset(time.Second)
			}
		}
	}
	for _, f := range files {
		select {
		case <-a.ctx.Done():
			return true
		case <-timer.C:
			Restart_XOIS <- struct{}{}
			timer.Reset(time.Second)
		default:
			src := filepath.Join(config.MasterDir, pgm, f.Key)
			dst := filepath.Join(config.XoisPath, f.Key)
			if !FileExists(src) {
				os.Remove(dst)
				text_log = append(text_log, dst+" ----- Delete File")
			}
			if script.CopyFileFull(src, dst) == nil {
				LogInfo(&Logger.pgm, "INFO:", src+" ----- Copy Pass")
			} else {
				text_log = append(text_log, src+" ----- Copy Error\n")
				return false
			}
		}
	}
	text := ""
	for _, i := range text_log {
		text = text + "\n" + i
	}
	LogInfo(&Logger.pgm, "CHANGE:", text)

	return true

}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
