package main

import (
	"Locker/config"
	"Locker/script"
	"context"
	"encoding/csv"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func LotWriterClient(ctx context.Context, lot string, ch chan config.CsvRow) {
	dayFolder := "unknown"
	wg.Add(1)
	defer func() {
		copyToUpload(dayFolder, lot, ID)
		Lot_chan.Delete(lot)
		wg.Done()
	}()
	if len(lot) >= 11 {
		dayFolder = lot[len(lot)-11 : len(lot)-5]
	}
	uuidColumn := 0
	Fullname := filepath.Join(config.OutputRoot, dayFolder, lot+"_"+ID[len(ID)-3:]+".csv")
	os.MkdirAll(filepath.Dir(Fullname), os.ModePerm)

	// 1. Mở file bằng quyền Đọc/Ghi, tạo mới nếu chưa có
	f, err := os.OpenFile(Fullname, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	// 2. Kiểm tra Header
	var hasHeader bool = false
	stat, _ := f.Stat()
	var header = []string{}
	if stat.Size() > 0 {
		// Nếu file có dữ liệu, thử đọc dòng đầu
		reader := csv.NewReader(f)
		header, err = reader.Read()
		if err == nil {
			// Ở đây bạn có thể so sánh existingHeader với header chuẩn của bạn
			// fmt.Println("Đã có header:", existingHeader)
			hasHeader = true
		}
	}

	// 3. Di chuyển con trỏ về cuối file để chuẩn bị ghi tiếp (Append)
	// Nếu chưa có header, con trỏ đang ở đầu (Size=0), SeekEnd vẫn là 0 -> Đúng.
	// Nếu đã có header, SeekEnd đưa ta xuống cuối để ghi tiếp dữ liệu -> Đúng.
	f.Seek(0, io.SeekEnd)
	writer := csv.NewWriter(f)
	defer writer.Flush()

	// 1. Nếu là file mới, ghi dòng tiêu đề (Header) đầu tiên

	manager.CheckAndLoadUUID(lot, Fullname, uuidColumn)
	// 3. Ghi Header (Lấy từ Key của Map đầu tiên)
	t := rand.Intn(30) + 150
	timer := time.NewTicker(time.Duration(t) * time.Second)
	defer timer.Stop()
	var lotRun atomic.Bool
	lotRun.Store(true)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if !lotRun.Load() {
				return
			}
			lotRun.Store(false)
		case msg, ok := <-ch:
			if !ok {
				msg.RawFile.Result.Store(false)
				msg.RawFile.Wg.Done()
				return
			}
			lotRun.Store(true)
			if !hasHeader {
				writer.Write(msg.Header)
				header = msg.Header
				hasHeader = true
			} else {
				headcheck := slices.Equal(header, msg.Header)
				if !headcheck {
					headchange := false
					for _, i := range msg.Header {
						if slices.Contains(header, i) {
							continue
						}
						header = append(header, i)
						headchange = true
					}
					if headchange {
						f.Close()
						updateFirstLine(Fullname, header)
						f, err = os.OpenFile(Fullname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
						if err != nil {
							msg.RawFile.Result.Store(false)
							msg.RawFile.Wg.Done()
							return
						}
						writer = csv.NewWriter(f)
					}
				}
			}
			writerFunc(writer, msg, lot, header)
		}
	}
}

func updateFirstLine(fileName string, newHeader []string) error {
	// 1. Đọc toàn bộ file
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	file.Close()
	if err != nil {
		return err
	}

	// 2. Thay đổi dòng đầu (index 0)
	if len(records) > 0 {
		records[0] = newHeader
	} else {
		records = append(records, newHeader)
	}

	// 3. Ghi đè lại toàn bộ
	outFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()
	return writer.WriteAll(records)
}

func writerFunc(writer *csv.Writer, msg config.CsvRow, lot string, header []string) {
	defer msg.RawFile.Wg.Done()
	defer writer.Flush()
	row := msg.Data
	newUUID, ok := row["uuid"]
	if !ok {
		for k, v := range row {
			// strings.ToLower giúp tìm cả "UUID", "Uuid", "uID"...
			if strings.Contains(strings.ToLower(k), "uid") {
				newUUID = v
				break // Tìm thấy rồi thì thoát vòng lặp ngay
			}
		}
	}
	if newUUID == "" || len(newUUID) < 10 {
		msg.RawFile.Result.Store(false)
		return // Phòng trường hợp dòng bị lỗi thiếu UUID
	}
	if manager.IsDuplicate(lot, newUUID) {
		// Log hoặc bỏ qua nếu trùng
		return
	}
	r := make([]string, len(header))
	r[0] = newUUID
	for i, colName := range header {
		if strings.EqualFold(colName, "uid") {
			continue
		}
		value, ok := row[colName]
		if !ok {
			r[i] = "-"
		}
		r[i] = value
	}
	if err := writer.Write(r); err != nil {
		msg.RawFile.Result.Store(false)
		return
	}
	go manager.AddUUID(lot, newUUID)
}

func copyToUpload(dayFolder, lot, ID string) error {
	src := filepath.Join(config.OutputRoot, dayFolder, lot+"_"+ID[len(ID)-3:]+".csv")
	os.MkdirAll(config.UploadRoot, os.ModePerm)
	dst := filepath.Join(config.UploadRoot, dayFolder, lot+"_"+ID[len(ID)-3:]+"_"+time.Now().Format("150405.00000")+".csv")
	return script.CopyFileFull(src, dst)
}

// Cấu trúc quản lý UUID theo từng Lot
type UUIDManager struct {
	// Key ngoài là LotID, Key trong là UUID
	cache map[string]map[string]struct{}
	mu    sync.RWMutex
}

var manager = &UUIDManager{
	cache: make(map[string]map[string]struct{}),
}

// CheckAndLoadUUID kiểm tra và nạp UUID từ file nếu cần thiết
func (m *UUIDManager) CheckAndLoadUUID(lotID string, filePath string, uuidColIndex int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Nếu Lot đã có trong Map (đã được load hoặc khởi tạo trước đó)
	if _, exists := m.cache[lotID]; exists {
		return
	}

	// 2. Nếu Lot chưa có trong Map, khởi tạo map con
	m.cache[lotID] = make(map[string]struct{})

	// 3. Kiểm tra xem file CSV của Lot này đã tồn tại trên ổ cứng chưa
	f, err := os.Open(filePath)
	if err != nil {
		// Nếu file chưa có (Lần đầu tạo Lot), không cần đọc, thoát luôn
		return
	}
	defer f.Close()

	// 4. Nếu file đã có, tiến hành đọc "duy nhất" cột UUID để nạp vào cache
	reader := csv.NewReader(f)
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if len(record) > 12 {
			if len(record[13]) > 1 && len(record[7]) > 3 {
				uuid := record[uuidColIndex]
				m.cache[lotID][uuid] = struct{}{}
			}
		}

	}
}

// IsDuplicate kiểm tra nhanh xem UUID đã tồn tại trong Lot đó chưa
func (m *UUIDManager) IsDuplicate(lotID string, uuid string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if lotMap, exists := m.cache[lotID]; exists {
		_, duplicated := lotMap[uuid]
		return duplicated
	}
	return false
}

// AddUUID Thêm UUID mới vào cache sau khi đã ghi file thành công
func (m *UUIDManager) AddUUID(lotID string, uuid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.cache[lotID]; !exists {
		m.cache[lotID] = make(map[string]struct{})
	}
	m.cache[lotID][uuid] = struct{}{}
}
