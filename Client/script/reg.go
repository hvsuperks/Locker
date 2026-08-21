package script

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func RegRead(path string, name string) string {

	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		path,
		registry.QUERY_VALUE,
	)

	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer k.Close()

	val, _, err := k.GetStringValue(name)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return val
}

func RegWrite(path string, name string, value string) error {

	k, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		path,
		registry.SET_VALUE,
	)

	if err != nil {
		fmt.Println(err)
		return err
	}
	defer k.Close()

	err = k.SetStringValue(name, value)
	fmt.Println(err)
	return err
}

func LoadDeepReg(rootPath string) (map[string]map[string]string, error) {
	res := make(map[string]map[string]string)

	// 1. Mở thư mục gốc (cha)
	root, err := registry.OpenKey(registry.CURRENT_USER, rootPath, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	// 2. Lấy danh sách tên các thư mục con (SubKeys)
	subKeyNames, _ := root.ReadSubKeyNames(0)

	for _, subName := range subKeyNames {
		// 3. Mở từng thư mục con
		subKey, err := registry.OpenKey(root, subName, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		innerMap := make(map[string]string)
		valNames, _ := subKey.ReadValueNames(0)

		// 4. Đọc toàn bộ Key-Value bên trong thư mục con
		for _, vName := range valNames {
			val, _, _ := subKey.GetStringValue(vName)
			innerMap[vName] = val
		}

		res[strings.ReplaceAll(subName, "|", "\\")] = innerMap
		subKey.Close()
	}
	return res, nil
}

func SaveMapToReg(rootPath string, data map[string]map[string]string) error {
	// 1. Mở hoặc Tạo mới thư mục gốc (cha)
	// Dùng SET_VALUE và CREATE_SUB_KEY để có quyền ghi
	registry.DeleteKey(registry.CURRENT_USER, rootPath)
	root, _, err := registry.CreateKey(registry.CURRENT_USER, rootPath, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer root.Close()

	for subName, innerMap := range data {
		// 2. Tạo hoặc Mở thư mục con (SubKey)
		subName = strings.ReplaceAll(subName, "\\", "|")
		subKey, _, err := registry.CreateKey(root, subName, registry.ALL_ACCESS)
		fmt.Println(subKey, "___", subName)
		if err != nil {
			continue
		}

		// 3. Ghi toàn bộ Key-Value từ innerMap vào SubKey này
		for vName, vValue := range innerMap {
			subKey.SetStringValue(vName, vValue)
		}
		subKey.Close() // Xong nhánh nào đóng nhánh đó luôn
	}
	return nil
}

func RegDelete(path string) {
	registry.DeleteKey(registry.CURRENT_USER, path)
}
