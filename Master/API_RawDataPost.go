package main

import (
	"Locker/config"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

var Lot_chan = sync.Map{} // map[string]chan CsvRow
var mutexReceiver sync.Mutex
var lotCount = make(chan struct{}, 200)
var listFile_Worker = make(map[string]bool)
var mutexListFile sync.RWMutex
var mutexClearCSVFolder = sync.RWMutex{}

type csvIDStruct struct {
	State  int
	result string
	mu     sync.RWMutex
}
type csvStatusStruct struct {
	mu   sync.RWMutex
	Data map[string]*csvIDStruct
}

var mutexCsvStatus sync.RWMutex
var csvStatus = make(map[string]*csvStatusStruct)

func HandleCSV(w http.ResponseWriter, r *http.Request) {
	mutexClearCSVFolder.RLock()
	defer mutexClearCSVFolder.RUnlock()
	lot := r.FormValue("lot")
	model := r.FormValue("model")
	cd := r.FormValue("congdoan")
	date := r.FormValue("date")
	ID := r.FormValue("id")
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	sourcename := handler.Filename
	if len(lot) < 18 || len(model) < 5 || len(cd) < 2 || len(date) != 6 || len(ID) != 9 || len(sourcename) < 24 {
		http.Error(w, "FormValue Fail", 500)
		return
	}
	fullPath := filepath.Join(config.CsvTMPPath, model, cd, date, lot, ID+"_"+sourcename)
	os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)

	tmp, err := os.Create(fullPath + ".tmp")
	if err != nil {
		http.Error(w, "Create Fail: "+err.Error(), 500)
		return
	}

	_, err = io.Copy(tmp, file)
	tmp.Close()
	if err != nil {
		http.Error(w, "IO Copy Fail: "+err.Error(), 500)
		return
	}

	err = os.Rename(fullPath+".tmp", fullPath)
	if err != nil {
		http.Error(w, "Rename Fail: "+err.Error(), 500)
		os.Remove(fullPath + ".tmp")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

	go func(f string) {
		WorkerList.mu.Lock()
		_, ok := WorkerList.Data[f]
		if ok {
			WorkerList.mu.Unlock()
			return
		}
		WorkerList.Data[f] = struct{}{}
		WorkerList.mu.Unlock()
		csvWorkerChan <- f
	}(fullPath)
}

type WorkerListType struct {
	mu   sync.Mutex
	Data map[string]struct{}
}

var WorkerList = WorkerListType{
	mu:   sync.Mutex{},
	Data: map[string]struct{}{},
}

var csvWorkerChan = make(chan string, 50)

func syscCSV() {
	for {
		select {
		case <-ctx.Done():
			return
		case srcPath := <-csvWorkerChan:
			rel := strings.TrimPrefix(srcPath, config.CsvTMPPath)
			part := strings.Split(rel, "\\")
			model, cd, date, lot, sourcename := part[0], part[1], part[02], part[3], part[4]

			Headers.mu.RLock()
			mdHeader, ok := Headers.model[model]
			if !ok || mdHeader == nil {
				Headers.mu.RUnlock()
				continue
			}
			Headers.mu.RUnlock()

			mdHeader.mu.RLock()
			HeaderMaster := mdHeader.cd[cd]
			mdHeader.mu.RUnlock()

			if HeaderMaster == nil {
				continue
			}

			dstPath := filepath.Join(
				config.BackupMes,
				model, cd,
				"20"+date[:2], // Kết quả: 2026
				date[2:4],     // Kết quả: 12 (Sửa từ 2:2 thành 2:4)
				date[4:],      // Kết quả: 03
				lot+".csv",
			)
			os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)

			err := processCSV(srcPath, dstPath, lot, model, cd, date, sourcename, HeaderMaster)
			if err == nil {
				err = os.Remove(srcPath)
				if err != nil {
					Fmt(fmtMaster, "syscCSV", "Remove Fail", srcPath)
					WorkerList.mu.Lock()
					delete(WorkerList.Data, srcPath)
					WorkerList.mu.Unlock()
				}
			} else {
				if err.Error() != "Header - Read File Fail" {
					Fmt(fmtMaster, "syscCSV", sourcename, err)
				}
				WorkerList.mu.Lock()
				delete(WorkerList.Data, srcPath)
				WorkerList.mu.Unlock()
			}
		}
	}
}

func processCSV(srcPath, dstPath, lot, model, cd, date, sourcename string, HeaderMaster *headCD) error {
	actual, _ := Lot_chan.LoadOrStore(model+cd+lot, &sync.Mutex{})
	mu := actual.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()
	HeaderMaster.mu.RLock()
	list := append([]string(nil), HeaderMaster.list...)
	HeaderMaster.mu.RUnlock()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	headNew, err := reader.Read()
	if err != nil {
		return fmt.Errorf("Header - Read File Fail")
	}

	_, err = os.Stat(dstPath)
	isNewFile := os.IsNotExist(err)

	f, err := os.OpenFile(dstPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// 1. Nếu là file mới, ghi dòng tiêu đề (Header) đầu tiên
	if isNewFile {
		if err := writer.Write(list); err != nil {
			return err
		}
	}
	manager.CheckAndLoadUUID(model, cd, lot, dstPath, 0)
	rowTemp := make([]string, len(list))
	var row = map[string]string{}
	for {

		rows, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// TỐI ƯU: Xóa sạch dữ liệu cũ trong map để dùng lại, KHÔNG cấp phát mới
		for k := range row {
			delete(row, k)
		}
		// TỐI ƯU: Xóa dữ liệu cũ trong slice
		for i := range rowTemp {
			rowTemp[i] = ""
		}

		for i, h := range headNew {
			if i < len(rows) {
				row[h] = rows[i]
			}
		}
		row["source"] = sourcename[:24]
		newUUID, ok := row["uuid"]
		if !ok {
			for k, v := range row {
				if len(k) >= 3 && (k == "uid" || k == "uID" || strings.Contains(strings.ToLower(k), "uid")) {
					newUUID = v
					break
				}
			}
		}
		if newUUID == "" || len(newUUID) < 10 || manager.IsDuplicate(model, cd, lot, newUUID) {
			continue
		}
		rowTemp[0] = newUUID
		for i, colName := range list {
			if colName == "uuid" {
				continue
			}
			value, ok := row[colName]
			if cd == "AF" && (value == "" || !ok) {
				value, ok = row[colName+"_90"]
				if value == "" || !ok {
					value = row[colName+"_180"]
				}
			}
			rowTemp[i] = value
		}
		err = writer.Write(rowTemp)
		if err != nil {
			return err
		}
		manager.AddUUID(model, cd, lot, newUUID)
	}
	return nil
}

// Cấu trúc quản lý UUID theo từng Lot
type UUIDManager struct {
	cache *UUIDModel
}

type UUIDStruct struct {
	mu     sync.RWMutex // Không dùng con trỏ cho Mutex để tránh nil pointer
	loaded bool         // Đánh dấu xem Lot này đã được nạp từ file chưa
	uuid   map[uuid.UUID]struct{}
}

type UUIDLot struct {
	mu  sync.RWMutex
	lot map[string]*UUIDStruct
}

type UUIDCD struct {
	mu sync.RWMutex
	cd map[string]*UUIDLot
}

type UUIDModel struct {
	mu    sync.RWMutex
	model map[string]*UUIDCD
}

// Khởi tạo manager chuẩn
var manager = &UUIDManager{
	cache: &UUIDModel{
		model: make(map[string]*UUIDCD),
	},
}

// getOrCreateLot là hàm helper giúp lấy ra UUIDStruct của một Lot cụ thể một cách an toàn (Thread-safe)
func (m *UUIDManager) getOrCreateLot(model, cd, lotID string) *UUIDStruct {
	// 1. Tầng Model
	m.cache.mu.Lock()
	modelNode, exists := m.cache.model[model]
	if !exists {
		modelNode = &UUIDCD{cd: make(map[string]*UUIDLot)}
		m.cache.model[model] = modelNode
	}
	m.cache.mu.Unlock()

	// 2. Tầng CD
	modelNode.mu.Lock()
	cdNode, exists := modelNode.cd[cd]
	if !exists {
		cdNode = &UUIDLot{lot: make(map[string]*UUIDStruct)}
		modelNode.cd[cd] = cdNode
	}
	modelNode.mu.Unlock()

	// 3. Tầng Lot
	cdNode.mu.Lock()
	lotNode, exists := cdNode.lot[lotID]
	if !exists {
		lotNode = &UUIDStruct{uuid: make(map[uuid.UUID]struct{})}
		cdNode.lot[lotID] = lotNode
	}
	cdNode.mu.Unlock()

	return lotNode
}

// CheckAndLoadUUID kiểm tra và nạp UUID từ file nếu cần thiết
func (m *UUIDManager) CheckAndLoadUUID(model, cd, lotID string, filePath string, uuidColIndex int) {
	// Lấy hoặc tạo Lot Struct một cách an toàn
	lotStruct := m.getOrCreateLot(model, cd, lotID)

	lotStruct.mu.Lock()
	defer lotStruct.mu.Unlock()

	// Nếu Lot đã được load từ file trước đó rồi thì thoát
	if lotStruct.loaded {
		return
	}

	// Đánh dấu đã xử lý (hoặc đang xử lý) để các luồng sau không đọc lại file nữa
	lotStruct.loaded = true

	// Kiểm tra xem file CSV của Lot này đã tồn tại trên ổ cứng chưa
	f, err := os.Open(filePath)
	if err != nil {
		// Nếu file chưa có (Lần đầu tạo Lot), không cần đọc, thoát luôn
		return
	}
	defer f.Close()

	// Tiến hành đọc "duy nhất" cột UUID để nạp vào cache
	reader := csv.NewReader(f)
	reader.ReuseRecord = true // Tối ưu bộ nhớ
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {

			continue
		}

		// Bảo vệ Index: Đảm bảo bản ghi có đủ số cột cần thiết
		maxIndex := uuidColIndex
		if 13 > maxIndex {
			maxIndex = 13
		}
		if len(record) <= maxIndex {

			continue
		}

		// Logic kiểm tra nghiệp vụ của bạn
		if len(record[13]) > 1 && len(record[7]) > 3 {
			uuid, _ := uuid.Parse(strings.Clone(record[uuidColIndex]))
			lotStruct.uuid[uuid] = struct{}{}
		}

	}
}

// IsDuplicate kiểm tra nhanh xem UUID đã tồn tại trong Lot đó chưa
func (m *UUIDManager) IsDuplicate(model, cd, lotID string, uuidI string) bool {
	// Sử dụng cơ chế RLock ở các tầng để đọc an toàn, tránh Data Race với hàm Add/Load
	m.cache.mu.RLock()
	modelNode, exists := m.cache.model[model]
	m.cache.mu.RUnlock()
	if !exists {
		return false
	}

	modelNode.mu.RLock()
	cdNode, exists := modelNode.cd[cd]
	modelNode.mu.RUnlock()
	if !exists {
		return false
	}

	cdNode.mu.RLock()
	lotNode, exists := cdNode.lot[lotID]
	cdNode.mu.RUnlock()
	if !exists {
		return false
	}
	u, _ := uuid.Parse(uuidI)
	// Kiểm tra xem UUID có nằm trong map cuối cùng không
	lotNode.mu.RLock()
	_, duplicated := lotNode.uuid[u]
	lotNode.mu.RUnlock()

	return duplicated
}

// AddUUID Thêm UUID mới vào cache sau khi đã ghi file thành công
func (m *UUIDManager) AddUUID(model, cd, lotID string, uuidI string) {
	lotStruct := m.getOrCreateLot(model, cd, lotID)
	u, _ := uuid.Parse(uuidI)
	lotStruct.mu.Lock()
	if lotStruct.uuid == nil {
		lotStruct.uuid = make(map[uuid.UUID]struct{})
	}
	lotStruct.uuid[u] = struct{}{}
	lotStruct.mu.Unlock()
}
