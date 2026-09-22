package main

import (
	"Locker/config"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var Lot_chan = sync.Map{} // map[string]chan CsvRow
var mutexReceiver sync.Mutex
var lotCount = make(chan struct{}, 200)
var listFile_Worker = make(map[string]bool)
var mutexListFile sync.RWMutex

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
	lot := r.FormValue("lot")
	model := r.FormValue("model")
	cd := r.FormValue("congdoan")
	date := r.FormValue("date")
	ID := r.FormValue("id")
	ver := r.FormValue("ver")
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	sourcename := handler.Filename
	if len(lot) < 18 || len(model) < 5 || len(cd) < 2 || len(date) != 6 || len(sourcename) < 24 {
		http.Error(w, "FormValue Fail", 500)
		return
	}
	dirpath := filepath.Join(config.BackupRawData, model, cd, date, lot)
	fullPath := filepath.Join(dirpath, ID+"-"+sourcename)
	os.MkdirAll(dirpath, os.ModePerm)
	listFile, _ := os.ReadDir(dirpath)
	oldpath := []string{}
	for _, f := range listFile {
		IDFile := strings.Split(ID+"-"+sourcename, "_")
		if len(IDFile) < 2 {
			continue
		}
		if strings.Contains(f.Name(), IDFile[0]) {
			partName := strings.Split(strings.Split(f.Name(), ".")[0], "_")
			if len(partName) != 3 {
				http.Error(w, "File Name Fail", http.StatusBadRequest)
				return
			}
			if partName[2] >= ver {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
				if len(oldpath) > 0 {
					for _, i := range oldpath {
						os.Remove(i)
					}
				}
				return
			} else {
				oldpath = append(oldpath, filepath.Join(dirpath, f.Name()))
			}
		}
	}
	tmp, err := os.Create(fullPath + ".tmp")
	if err != nil {
		http.Error(w, "Create Fail: "+err.Error(), 500)
		return
	}
	defer os.Remove(fullPath + ".tmp")
	_, err = io.Copy(tmp, file)
	tmp.Close()
	if err != nil {
		http.Error(w, "IO Copy Fail: "+err.Error(), 500)
		return
	}

	err = os.Rename(fullPath+".tmp", fullPath)
	if err != nil {
		http.Error(w, "Rename Fail: "+err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
	if len(oldpath) > 0 {
		for _, i := range oldpath {
			os.Remove(i)
		}
	}
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

func syscCSV(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case srcPath := <-csvWorkerChan:
			rel := strings.TrimPrefix(srcPath, config.CsvTMPPath)
			part := strings.Split(rel, "\\")
			model, cd, date, lot, sourcename := part[0], part[1], part[02], part[3], part[4]
			dstPath := filepath.Join(
				config.BackupRawData,
				model, cd,
				"20"+date[:2], // Kết quả: 2026
				date[2:4],     // Kết quả: 12 (Sửa từ 2:2 thành 2:4)
				date[4:],      // Kết quả: 03
				lot+".csv",
			)
			os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)

			err := processCSV(srcPath, dstPath, lot, model, cd, date, sourcename)
			if err == nil {
				err = os.Remove(srcPath)
				if err != nil {
					LogInfo(&Logger.CSV, "syscCSV", "Remove Fail", srcPath)
					WorkerList.mu.Lock()
					delete(WorkerList.Data, srcPath)
					WorkerList.mu.Unlock()
				}
			} else {
				if err.Error() != "Header - Read File Fail" {
					LogInfo(&Logger.CSV, "syscCSV", sourcename, err)
				}
				WorkerList.mu.Lock()
				delete(WorkerList.Data, srcPath)
				WorkerList.mu.Unlock()
			}
		}
	}
}

func processCSV(UpdateFile, SourcFile, lot, model, cd, date, sourcename string) error {
	defer func() {
		if r := recover(); r != nil {
			LogInfo(&Logger.Debug, fmt.Sprintf("panic: %v", r))
		}
	}()
	actual, _ := Lot_chan.LoadOrStore(model+cd+lot, &sync.Mutex{})
	mu := actual.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	file, err := os.Open(UpdateFile)
	if err != nil {
		return err
	}
	defer file.Close()

	readerUpdate := csv.NewReader(file)
	readerUpdate.FieldsPerRecord = -1
	headNew, err := readerUpdate.Read()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(SourcFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()
	readerSourc := csv.NewReader(f)
	readerSourc.FieldsPerRecord = -1
	// 1. Nếu là file mới, ghi dòng tiêu đề (Header) đầu tiên
	csvData := [][]string{}
	heads := []string{}
	var headStatus bool
	headCurrent, _ := readerSourc.Read()
	heads, headStatus = checkHead(headCurrent, headNew)
	if !headStatus {
		csvData, _ = readerSourc.ReadAll()
	}
	if headStatus {
		manager.CheckAndLoadUUID(model, cd, lot, SourcFile, 0)
	} else {
		manager.DeleteUUID(model, cd, lot)
	}
	rowTemp := make([]string, len(heads))
	var row = map[string]string{}
	for {
		rows, err := readerUpdate.Read()
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
			} else {
				row[h] = "-"
			}
		}
		if len(sourcename) < 25 {
			row["source"] = sourcename
		} else {
			row["source"] = sourcename[:24]
		}
		newUUID, ok := row["uuid"]
		if !ok {
			for k, v := range row {
				if len(k) >= 3 && (k == "uid" || k == "uID" || strings.Contains(strings.ToLower(k), "uid")) {
					newUUID = v
					break
				}
			}
		}
		if newUUID == "" || len(newUUID) < 10 || (manager.IsDuplicate(model, cd, lot, newUUID) && ok) {
			continue
		}
		rowTemp[0] = newUUID
		for i, colName := range heads {
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
		tmp := append([]string(nil), rowTemp...)
		csvData = append(csvData, tmp)
		manager.AddUUID(model, cd, lot, newUUID)
	}
	if !headStatus {
		m := make(map[string][]string)
		for _, row := range csvData {
			id := row[0]
			m[id] = row // gặp trùng sẽ ghi đè dòng cũ
		}
		f.Truncate(0)
		f.Seek(0, 0)
		writer = csv.NewWriter(f)
		err = writer.Write(heads)
		if err != nil {
			return err
		}
		for k := range m {
			err = writer.Write(m[k])
			if err != nil {
				return err
			}
		}
	} else {
		f.Seek(0, io.SeekEnd)
		writer.WriteAll(csvData)
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
	last   time.Time
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
		lotNode = &UUIDStruct{uuid: make(map[uuid.UUID]struct{}), last: time.Now()}
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
		lotStruct.last = time.Now()
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

// CheckAndLoadUUID kiểm tra và nạp UUID từ file nếu cần thiết
func (m *UUIDManager) CleanUUID() {
	for {
		time.Sleep(time.Minute * 30)
		// 1. Khóa đọc cache chính
		m.cache.mu.RLock()
		// Tạo slice chứa pointer tới các modelNode để giảm thời gian giữ RLock cấp m.cache
		models := make([]*UUIDCD, 0, len(m.cache.model))
		for _, modelNode := range m.cache.model {
			models = append(models, modelNode)
		}
		m.cache.mu.RUnlock()

		// 2. Duyệt qua từng tầng để thu gom lot
		for _, modelNode := range models {
			modelNode.mu.RLock()
			cdNodes := make([]*UUIDLot, 0, len(modelNode.cd))
			for _, cdNode := range modelNode.cd {
				cdNodes = append(cdNodes, cdNode)
			}
			modelNode.mu.RUnlock()

			for _, cdNode := range cdNodes {
				cdNode.mu.RLock()
				lots := make([]*UUIDStruct, 0, len(cdNode.lot))
				for _, lot := range cdNode.lot {
					lots = append(lots, lot)
				}
				cdNode.mu.RUnlock()

				// 3. Tiến hành cleanup
				for _, lot := range lots {
					lot.mu.Lock()
					if time.Since(lot.last) > time.Hour {
						lot.loaded = false
						lot.uuid = nil // Giải phóng reference để GC thu hồi bộ nhớ ngay
					}
					lot.mu.Unlock()
				}
			}
		}
	}
}

// CheckAndLoadUUID kiểm tra và nạp UUID từ file nếu cần thiết
func (m *UUIDManager) DeleteUUID(model, cd, lotID string) {
	// 1. Tầng Model
	lotStruct := m.getOrCreateLot(model, cd, lotID)

	lotStruct.mu.Lock()
	defer lotStruct.mu.Unlock()
	if time.Since(lotStruct.last) > time.Hour {
		lotStruct.uuid = nil
		lotStruct.loaded = true
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
	lotStruct.loaded = true
	lotStruct.mu.Unlock()
}

func checkHead(src, dst []string) ([]string, bool) {
	exist := make(map[string]bool)
	ok := true
	if len(src) < 2 {
		src = []string{`uuid`, `source`}
	} else {
		for _, h := range src {
			exist[h] = true
		}
	}
	for _, h := range dst {
		s := h
		if strings.HasSuffix(h, "_90") {
			s = h[:len(h)-3]
		} else if strings.HasSuffix(h, "_180") || strings.HasSuffix(h, "_270") {
			s = h[:len(h)-4]
		}
		if !exist[s] {
			src = append(src, s)
			ok = false
			exist[s] = true
		}
	}
	return src, ok
}
