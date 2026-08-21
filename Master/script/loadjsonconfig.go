package script

import (
	"Locker/config"
	"encoding/json"
	"path/filepath"
	"sync"
	"time"
)

func LoadMasterConfig(fileName string, target interface{}, mu *sync.RWMutex) error {
	filePath := filepath.Join(config.HttpMasterDir, fileName)

	j, er := LoadJson(filePath)

	if er != nil {
		// Sleep một chút trước khi panic nếu cần thiết như code cũ của bạn
		if fileName == "Model_config.json" {
			time.Sleep(5 * time.Second)
		}
		return er
	}

	// Lock nếu có mutex truyền vào
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}

	er = json.Unmarshal(j.([]byte), target)
	if er != nil {
		return er
	}
	return nil
}
