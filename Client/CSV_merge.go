package main

import (
	"Locker/config"
	"Locker/script"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func (a mClient) mergeManager(MergeRuning *atomic.Bool) {
	MergeRuning.Store(true)
	defer func() {
		MergeRuning.Store(false)
		setStatus(keyMerge, config.Locker_state{
			Value: "Stop",
			Color: "Fail",
		})
		LogInfo(&Logger.merge, "INFO:", "Merge Stop")
	}()
	var JobQueue = make(chan string)
	go a.heart(JobQueue)

	defer close(JobQueue)
	Lot_chan = sync.Map{} // map[string]chan CsvRow
	defer Lot_chan.Clear()
	os.MkdirAll(config.UploadRoot, os.ModePerm)
	os.MkdirAll(config.OutputRoot, os.ModePerm)
	go a.ProcessCSV(JobQueue)
	timer := time.NewTicker(25 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-timer.C:
			filepath.WalkDir(config.BackupRoot, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if strings.HasSuffix(strings.ToLower(path), ".csv") {
					JobQueue <- path
				}
				return nil
			})
		}
	}
}

func countCSVFiles(root string) string {
	count := 0

	// WalkDir duyệt qua tất cả thư mục và thư mục con
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Kiểm tra nếu không phải thư mục và có đuôi là .csv
		if !d.IsDir() && strings.ToLower(filepath.Ext(d.Name())) == ".csv" {
			count++
		}
		return nil
	})
	// WalkDir duyệt qua tất cả thư mục và thư mục con
	err = filepath.WalkDir(config.BackupRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Kiểm tra nếu không phải thư mục và có đuôi là .csv
		if !d.IsDir() && strings.ToLower(filepath.Ext(d.Name())) == ".csv" {
			count++
		}
		return nil
	})
	if err != nil {
		fmt.Println("CSV Count Fail", err)
		return "Fail"
	} else {
		return strconv.Itoa(count)
	}
}

func (a mClient) heart(JobQueue chan string) {
	timer := time.NewTicker(10 * time.Second)
	defer timer.Stop()
	timerClear := time.NewTicker(300 * time.Second)
	defer timerClear.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-timerClear.C:
			filepath.WalkDir(config.UploadRoot, func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}
				if strings.HasSuffix(d.Name(), ".tmp") {
					os.Rename(path, path[:len(path)-4])
				} else if strings.HasSuffix(d.Name(), ".log") {
					os.Remove(path)
				}
				return nil
			})
			go script.RemoveEmptyDirs(config.MesPath)
			go script.RemoveEmptyDirs(config.BackupRoot)
			go script.RemoveEmptyDirs(config.UploadRoot)
		case <-timer.C:
			merge_state := "OK"
			entries, err := os.ReadDir(config.MesPath)
			if err == nil && len(entries) > 10 {
				go func() {
					filepath.WalkDir(config.MesPath, func(path string, d fs.DirEntry, err error) error {
						if !d.IsDir() && strings.HasSuffix(d.Name(), ".csv") {
							JobQueue <- path
						}
						return nil
					})
				}()
				merge_state = "Fail"
			} else if err != nil {
				merge_state = "Fail"
			}
			setStatus(keyMerge, config.Locker_state{
				Value: countCSVFiles(config.UploadRoot),
				Color: merge_state,
			})
		}
	}
}

var fileRun = sync.Map{}

func (a mClient) ProcessCSV(JobQueue <-chan string) {
	for {
		select {
		case <-a.ctx.Done():
			return
		case file := <-JobQueue:
			f, err := os.Open(file)
			if err != nil {
				LogInfo(&Logger.merge, "ERROR:", file+"Open error:"+err.Error())
				f.Close()
				continue
			}

			if _, loader := fileRun.LoadOrStore(filepath.Base(file), struct{}{}); loader {
				continue
			}

			reader := csv.NewReader(f)
			reader.Comma = ','
			reader.LazyQuotes = true
			// đọc header
			header, _ := reader.Read()
			var result atomic.Bool
			result.Store(true)
			wgFile := sync.WaitGroup{}
			for {
				record, err := reader.Read()

				if err == io.EOF {
					break
				}
				if err != nil {
					//Fmt("File NG")
					f.Close()
					LogInfo(&Logger.merge, "ERROR:", file, err)
					continue // ❗ file hỏng → dừng luôn
				}

				row := make(map[string]string)
				for i, v := range header {
					row[v] = record[i]
				}

				lot := strings.TrimSpace(row[config.LotColumn])

				if lot == "" {
					f.Close()
					LogInfo(&Logger.merge, "ERROR:", file, "Lot Fail")
					continue
				}
				semaphore <- struct{}{}
				wgFile.Add(1)
				go a.dispatchToLot(lot, config.CsvRow{
					File_name: filepath.Base(file),
					Header:    header,
					Data:      row,
					RawFile: &config.RawFileStatus{
						Wg:     &wgFile,
						Result: &result,
					},
				})
			}
			f.Close()
			go func(wgf *sync.WaitGroup, result *atomic.Bool, filename string) {
				defer func() {
					fileRun.Delete(filename)
				}()
				wgf.Wait()
				if !result.Load() {
					return
				}
				re, _ := filepath.Rel(config.BackupRoot, file)
				dst := filepath.Join(config.BackupRoot2, re)
				os.MkdirAll(filepath.Dir(dst), os.ModePerm)
				if strings.Contains(strings.ToLower(file), "backup_mes") {
					os.Rename(file, dst)
				}
			}(&wgFile, &result, filepath.Base(file))
		}
	}
}

type WorkerCountStruct struct {
	State atomic.Bool
	mu    sync.RWMutex
	ch    chan config.CsvRow
}

var mutexWorkerCount = sync.RWMutex{}
var WorkerCount = make(map[string]*WorkerCountStruct)
var semaphore = make(chan struct{}, 300)

func (a mClient) dispatchToLot(lot string, row config.CsvRow) {
	defer func() { <-semaphore }()
	defer func() {
		if r := recover(); r != nil {
			row.RawFile.Result.Store(false)
			row.RawFile.Wg.Done()
			LogInfo(&Logger.merge, "ERROR:", "Channal Close")
		}
	}()
	actual, loaded := Lot_chan.LoadOrStore(lot, make(chan config.CsvRow, 100))
	ch := actual.(chan config.CsvRow)
	if !loaded {
		// Nếu có thằng khác nhanh chân hơn cộng trước, vòng lặp sẽ chạy lại để check lại 'current'
		go LotWriterClient(a.ctx, lot, ch)
	}
	ch <- row
}
