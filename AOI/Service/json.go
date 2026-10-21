package main

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

func LoadJson(filename string) (error, map[string]bool) {
	data, err := os.ReadFile(filename)

	if err != nil {
		return err, nil
	}
	var d map[string]bool
	err = json.Unmarshal(data, &d)
	if err != nil || d == nil {
		return err, nil
	}
	return nil, d
}
