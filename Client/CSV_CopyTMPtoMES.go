package main

import (
	"Locker/script"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func copyTMPtoMes(ctx context.Context) {
	srcPath := `C:\xOIS\MES_tmp`
	dstPath := `D:\MES`
	os.MkdirAll(srcPath, os.ModePerm)
	os.MkdirAll(dstPath, os.ModePerm)
	timer := time.NewTicker(20 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			filepath.WalkDir(srcPath, func(srcFile string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					return nil
				}
				if strings.HasSuffix(srcFile, ".csv") {
					dstFile := filepath.Join(dstPath, d.Name())
					if script.CopyFileFull(srcFile, dstFile) != nil {
						return nil
					}
					os.Remove(srcFile)
				}
				return nil
			})
		}
	}
}
