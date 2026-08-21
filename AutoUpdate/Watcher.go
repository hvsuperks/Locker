package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

func Watcher(ctxWatcher context.Context) {
	lastLocker := time.Now()
	lastUi := atomic.Value{}
	isupdateUi := atomic.Bool{}
	lastUi.Store(time.Now())
	for {
		select {
		case <-ctxWatcher.Done():
			return
		default:
			func() {

				defer func() {
					if r := recover(); r != nil {
						Logger.Printf("panic: %v", r)
						time.Sleep(time.Second)
					}
				}()
				for {
					select {
					case <-ctxWatcher.Done():
						return
					case url := <-AppLocker.UpdateChan:
						if time.Since(lastLocker) < 30*time.Second {
							continue
						}
						if err := DownloadFile(url, AppLocker.UpdatePath); err == nil {
							Logger.Printf("INFO: DownloadFile Update Locker %s", url)
							lastLocker = time.Now()
						} else {
							Logger.Printf("ERROR: DownloadFile %s \n Err: %v ", url, err)
						}

					case i := <-AppUi.UpdateChan:
						if isupdateUi.Load() {
							continue
						}
						if time.Since(lastUi.Load().(time.Time)) < 30*time.Second {
							continue
						}
						isupdateUi.Store(true)
						go UpdateUI_Download(i, &isupdateUi, &lastUi)

					case <-AppLocker.CloseChan:
						if err := OpenApp(AppLocker.UpdatePath, AppLocker.ExePath, &AppLocker.Ver); err != nil {
							Logger.Printf("ERROR: Open App %s \n Err: %v ", AppLocker.ExePath, err)
						}
						time.Sleep(time.Second * 5)
					case <-AppUi.CloseChan:
						if err := OpenApp(AppUi.UpdatePath, AppUi.ExePath, &AppUi.Ver); err != nil {
							Logger.Printf("ERROR: Open App %s \n Err: %v ", AppLocker.ExePath, err)
						}
						time.Sleep(time.Second * 5)
					default:

						if _, err := os.Stat(AppLocker.UpdatePath); err == nil {
							isShutdownLocker.Store(true)
							time.Sleep(time.Second * 2)
							taskkill("locker.exe")
							time.Sleep(time.Second)
							os.Remove(AppLocker.ExePath)
							if os.Rename(AppLocker.UpdatePath, AppLocker.ExePath) == nil {
								AppLocker.CloseChan <- struct{}{}
							}
						} else {
							last := AppLocker.Last.Load().(time.Time)
							if time.Since(last) > 2*time.Second {
								taskkill("Locker.exe")
								AppLocker.Last.Store(time.Now())
								AppLocker.CloseChan <- struct{}{}
							}
						}

						if _, err := os.Stat(AppUi.UpdatePath); err == nil {
							taskkill("Ui.exe")
							time.Sleep(time.Second)
							os.Remove(AppUi.ExePath)
							if os.Rename(AppUi.UpdatePath, AppUi.ExePath) == nil {
								AppUi.CloseChan <- struct{}{}
							}
						} else {
							last := AppUi.Last.Load().(time.Time)
							if time.Since(last) > 2*time.Second {
								AppUi.Last.Store(time.Now())
								AppUi.CloseChan <- struct{}{}
								taskkill("Ui.exe")
							}
						}
						time.Sleep(500 * time.Millisecond)
					}
				}
			}()
		}
	}
}

func UpdateUI_Download(s string, isupdate *atomic.Bool, last *atomic.Value) {
	defer func() {
		if r := recover(); r != nil {
			Logger.Printf("panic: %v", r)
			time.Sleep(time.Second)
		}
		isupdate.Store(false)
		last.Store(time.Now())
	}()
	url := strings.Split(s, "|")
	if err := DownloadFile(url[0], libzip); err == nil {
		if err := Unzip(libzip, tmpZip); err == nil {
			if filepath.WalkDir(tmpZip, func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(tmpZip, path)
				dst := filepath.Join(libUiFolder, rel)
				if _, err := os.Stat(dst); err != nil {
					os.MkdirAll(filepath.Dir(dst), os.ModePerm)
					if err := os.Rename(path, dst); err != nil {
						return err
					}
				}
				return nil
			}) == nil {
				if err := DownloadFile(url[1], AppUi.UpdatePath); err == nil {
					taskkill("Ui.exe")
				} else {
					Logger.Printf("ERROR: DownloadFile %s \n Err: %v ", url[1], err)
				}
			}
		} else {
			Logger.Printf("ERROR: Lib Extrac %s → %s \n Err: %v ", libzip, tmpZip, err)
		}
	} else {
		Logger.Printf("ERROR: DownloadFile %s \n Err: %v ", url[0], err)
	}
	os.Remove(libzip)
	os.RemoveAll(tmpZip)
}
