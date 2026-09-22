package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

func Watcher(ctxWatcher context.Context) {
	lastLocker := time.Now()
	lastUiUpdate := atomic.Value{}
	isupdateUi := atomic.Bool{}
	lastUiUpdate.Store(time.Now())

	for {
		select {
		case <-ctxWatcher.Done():
			return
		default:
			func() {

				defer func() {
					if r := recover(); r != nil {
						LogUnique(fmt.Sprintf("panic: %v", r))
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
						LogUnique("Debug:", "Watcher.go:42", `AppLocker Update`, url)
						if err := DownloadFile(url, AppLocker.UpdatePath); err == nil {
							LogUnique(fmt.Sprintf("INFO: DownloadFile Update Locker %s", url))
							lastLocker = time.Now()
						} else {
							LogUnique(fmt.Sprintf("ERROR: DownloadFile %s \n Err: %v ", url, err))
						}

					case i := <-AppUi.UpdateChan:
						if isupdateUi.Load() {
							continue
						}
						if time.Since(lastUiUpdate.Load().(time.Time)) < 30*time.Second {
							continue
						}
						isupdateUi.Store(true)
						go UpdateUI_Download(i, &isupdateUi, &lastUiUpdate)

					default:
						if _, err := os.Stat(AppLocker.UpdatePath); err == nil {
							LogUnique("debug", `AppLocker.UpdatePath: `, AppLocker.UpdatePath, `stat`, err)
							isShutdownLocker.Store(true)
							running := true
							for range 21 {
								time.Sleep(100 * time.Millisecond)
								if !IsRunning("Locker.exe") {
									running = false
									break
								}
							}
							if running {
								taskkill("locker.exe")
								time.Sleep(time.Second)
							}
							os.Remove(AppLocker.ExePath)
							if os.Rename(AppLocker.UpdatePath, AppLocker.ExePath) == nil {
								go func() {
									AppLocker.Last.Store(time.Now())
									if err := OpenApp(AppLocker.UpdatePath, AppLocker.ExePath, &AppLocker.Ver); err != nil {
										LogUnique(fmt.Sprintf("ERROR: Open App %s \n Err: %v ", AppLocker.ExePath, err))
									}
									AppLocker.Last.Store(time.Now())
								}()
							}
						} else {
							last, _ := AppLocker.Last.Load().(time.Time)
							if time.Since(last) > 3*time.Second {
								LogUnique("debug", `watcher.go:87`, last)
								taskkill("locker.exe")
								for range 21 {
									time.Sleep(100 * time.Millisecond)
									if !IsRunning("Locker.exe") {
										break
									}
								}
								go func() {
									AppLocker.Last.Store(time.Now())
									if err := OpenApp(AppLocker.UpdatePath, AppLocker.ExePath, &AppLocker.Ver); err != nil {
										LogUnique(fmt.Sprintf("ERROR: Open App %s \n Err: %v ", AppLocker.ExePath, err))
									}
									AppLocker.Last.Store(time.Now())
								}()

							}
						}

						if _, err := os.Stat(AppUi.UpdatePath); err == nil {
							taskkill("Ui.exe")
							time.Sleep(time.Second)
							os.Remove(AppUi.ExePath)
							if os.Rename(AppUi.UpdatePath, AppUi.ExePath) == nil {
								if err := OpenApp(AppUi.UpdatePath, AppUi.ExePath, &AppUi.Ver); err != nil {
									LogUnique(fmt.Sprintf("ERROR: Open App %s \n Err: %v ", AppLocker.ExePath, err))
								}
								last := time.Now()
								AppUi.Last.Store(last)
								for i := range 12 {
									if last != AppUi.Last.Load().(time.Time) {
										break
									}
									if i >= 10 {
										AppUi.Ver.Store("1")
										break
									}
									time.Sleep(200 * time.Millisecond)
								}
							}
						} else {
							last, _ := AppUi.Last.Load().(time.Time)
							if time.Since(last) > 3*time.Second {
								taskkill("Ui.exe")
								time.Sleep(500 * time.Millisecond)

								if err := OpenApp(AppUi.UpdatePath, AppUi.ExePath, &AppUi.Ver); err != nil {
									LogUnique(fmt.Sprintf("ERROR: Open App %s \n Err: %v ", AppLocker.ExePath, err))
								}
								last = time.Now()
								AppUi.Last.Store(last)
								for i := range 12 {
									if last != AppUi.Last.Load().(time.Time) {
										break
									}
									if i >= 10 {
										AppUi.Ver.Store("1")
										break
									}
									time.Sleep(200 * time.Millisecond)
								}

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
			LogUnique(fmt.Sprintf("panic: %v", r))
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
					LogUnique(fmt.Sprintf("ERROR: DownloadFile %s \n Err: %v ", url[1], err))
				}
			}
		} else {
			LogUnique(fmt.Sprintf("ERROR: Lib Extrac %s → %s \n Err: %v ", libzip, tmpZip, err))
		}
	} else {
		LogUnique(fmt.Sprintf("ERROR: DownloadFile %s \n Err: %v ", url[0], err))
	}
	os.Remove(libzip)
	os.RemoveAll(tmpZip)
}

func IsRunning(exe string) bool {
	out, err := exec.Command(
		"tasklist",
		"/FI", "IMAGENAME eq "+exe,
		"/NH",
	).Output()
	if err != nil {
		return false
	}

	return strings.Contains(strings.ToLower(string(out)), strings.ToLower(exe))
}
