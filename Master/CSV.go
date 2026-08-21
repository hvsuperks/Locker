package main

import (
	"Locker/config"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func RemoveEmptyDirs(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Chỉ xử lý folder
		if !info.IsDir() {
			if strings.HasSuffix(path, ".csv") && info.Size() == 0 {
				os.Remove(path)
			}
			return nil
		}
		// Không xóa root
		if path == root {
			return nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		if len(entries) == 0 {
			_ = os.Remove(path)
		}
		return nil
	})
}

func CSVMerge() {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			modelList, err := os.ReadDir(config.CsvTMPPath)
			if err != nil {
				timer.Reset(time.Second)
				Fmt(fmtMaster, "CSVMerge", "CSV Tmp No File")
				continue
			}
			for _, model := range modelList {
				Headers.mu.RLock()
				headModel := Headers.model[model.Name()]
				Headers.mu.RUnlock()

				if headModel == nil {
					continue
				}
				partList, err := os.ReadDir(filepath.Join(config.CsvTMPPath, model.Name()))
				if err != nil || len(partList) == 0 {
					continue
				}
				for _, part := range partList {
					headModel.mu.RLock()
					headPart := headModel.cd[part.Name()]
					headModel.mu.RUnlock()
					if headPart == nil {
						continue
					}
					filepath.WalkDir(filepath.Join(config.CsvTMPPath, model.Name(), part.Name()), func(path string, d fs.DirEntry, err error) error {
						if d.IsDir() {
							return nil
						}
						if !strings.HasSuffix(path, ".csv") {
							return nil
						}

						if len(strings.Split(path, "\\")) != 7 {
							Fmt(fmtMaster, "CSVMerge", "Len Path != 7", path)
							return nil
						}
						WorkerList.mu.Lock()
						_, ok := WorkerList.Data[path]
						if ok {
							WorkerList.mu.Unlock()
							return nil
						}
						WorkerList.Data[path] = struct{}{}
						WorkerList.mu.Unlock()
						csvWorkerChan <- path
						return nil

					})
				}
			}
			timer.Reset(time.Minute)
			mutexClearCSVFolder.Lock()
			RemoveEmptyDirs(config.CsvTMPPath)
			mutexClearCSVFolder.Unlock()
		}
	}

}
