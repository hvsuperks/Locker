package main

import (
	"Locker/script"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func SendFileToMaster(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		func() {
			defer RecoverLog()
			sendFilePath := `D:\SendFile`
			outPath := `D:\SendFile\uploadQueue`
			os.MkdirAll(outPath, os.ModePerm)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				time.Sleep(2 * time.Second)
				listdir, err := os.ReadDir(sendFilePath)
				if err != nil {
					LogInfo(&Logger.debug, err)
					time.Sleep(3 * time.Second)
					continue
				}
				if len(listdir) == 0 {
					continue
				}
				for _, f := range listdir {
					src := filepath.Join(sendFilePath, f.Name())
					if strings.EqualFold(f.Name(), "uploadQueue") {
						continue
					}
					if file, err := os.Stat(src); err == nil {
						name := f.Name()
						if matched, _ := regexp.MatchString(`^(?:\d{9}|unknown)_\d{8}_\d{6}_.+$`, f.Name()); !matched {
							s := time.Now().Format("20060102_150405")
							name = fmt.Sprintf(
								"%s_%s_%s",
								ID,
								s,
								f.Name(),
							)
						}
						dst := filepath.Join(outPath, name)

						if file.IsDir() {
							dst := dst + `.zip`
							tmp := filepath.Join(`D:\`, name+`.tmp`)
							if err := script.ZipFolder(src, tmp); err == nil {
								if os.Rename(tmp, dst) == nil {
									os.RemoveAll(src)
								}
							} else {
								LogInfo(&Logger.debug, err)
							}
						} else {
							os.Rename(src, filepath.Join(outPath, name))
						}
					}
				}

			}
		}()
		time.Sleep(10 * time.Second)
	}
}
