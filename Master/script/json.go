package script

import (
	"encoding/json"
	"os"
)

func SaveJson(filename string, data any) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Giúp file dễ đọc hơn
	return encoder.Encode(data)
}

func LoadJson(filename string) (any, error) {

	data, err := os.ReadFile(filename)
	if err != nil {
		return "Lỗi đọc file:", err
	}
	return data, err
}

func MergeHeaders(base map[string]map[string][]string, extra map[string]map[string][]string) {
	for model, types := range extra {
		// Nếu model chưa tồn tại trong base, tạo mới
		if _, exists := base[model]; !exists {
			base[model] = make(map[string][]string)
		}

		// Gộp các type (AF, OIS, Tilt)
		for typeName, columns := range types {
			base[model][typeName] = columns
		}
	}
}
