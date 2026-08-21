package script

import (
	"fmt"

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

func RegWrite(path string, name string, value string) bool {

	k, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		path,
		registry.SET_VALUE,
	)

	if err != nil {
		fmt.Println(err)
		return false
	}
	defer k.Close()

	err = k.SetStringValue(name, value)
	fmt.Println(err)
	return err == nil
}
func RegDelete(path string) {
	registry.DeleteKey(registry.CURRENT_USER, path)
}
