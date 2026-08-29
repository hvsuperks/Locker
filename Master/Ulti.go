package main

import (
	"Locker/config"
	"archive/zip"
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

func ComparePath(scr, dst string) bool {
	// 1. Đưa về đường dẫn tuyệt đối và xử lý Shortcut/Symlink
	path1, err1 := filepath.Abs(scr)
	path2, err2 := filepath.Abs(dst)
	if err1 != nil || err2 != nil {
		return false
	}

	// 2. Làm sạch đường dẫn (xóa các dấu . hoặc .. thừa)
	path1 = filepath.Clean(path1)
	path2 = filepath.Clean(path2)

	// 3. Với Windows: Phải chuyển về cùng chữ thường vì Windows không phân biệt hoa/thường
	// "C:\Users" và "c:\users" là một
	return strings.EqualFold(path1, path2)
}

func CopyFolder(src, dst string) {
	s, err := os.Stat(src)
	if err != nil {
		return
	}
	if s.IsDir() {
		filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(src, path)
			Dst := filepath.Join(dst, rel)
			CopyFileFull(path, Dst)
			return nil
		})
	} else {
		CopyFileFull(src, dst)
	}
}

func CopyFileFull(src, dst string) error {
	in, err := os.Open(src)

	if err != nil {

		return err
	}
	defer in.Close()
	dst_folder := filepath.Dir(dst)
	os.MkdirAll(dst_folder, 0755)
	out, err := os.Create(dst + ".tmp")
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err != nil {
		out.Close()
		return err
	}
	i := 0
	out.Close()
	for {
		err = os.Rename(dst+".tmp", dst)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
		i++
		if i >= 25 {
			return err
		}
	}
	// copy time
	info, err := os.Stat(src)
	if err == nil {
		os.Chtimes(dst, info.ModTime(), info.ModTime())
	}
	return nil
}

func GetMD5(path string) (string, error) {

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func Unzip(zipPath, dst string) error {
	Temp := dst + "_tmp"
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	os.MkdirAll(Temp, os.ModePerm)
	for _, f := range r.File {

		fpath := filepath.Join(Temp, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		os.MkdirAll(filepath.Dir(fpath), os.ModePerm)

		outFile, _ := os.Create(fpath)
		rc, _ := f.Open()

		io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()
	}
	os.Rename(Temp, dst)
	return nil
}

func CreateMasterPGM(path string) (bool, map[string]map[string]string) {
	md5map := make(map[string]map[string]string)
	file, _ := os.Stat(path)
	if file.IsDir() {
		nameFile := filepath.Base(path)
		namePart := strings.Split(nameFile, "_")
		if len(namePart) != 4 {
			return false, nil
		}
		zipPath := filepath.Join(config.HttpMasterDir, namePart[0], namePart[1], nameFile+".zip")
		fmt.Println("ZipPath", zipPath)
		temp := zipPath + ".tmp"
		os.MkdirAll(filepath.Join(config.HttpMasterDir, namePart[0], namePart[1]), os.ModePerm)
		err := zipFolder(path, temp)
		if err != nil {
			return false, nil
		}
		os.Rename(temp, zipPath)
		md5, _ := GetMD5(zipPath)
		md5map[namePart[0]+"_"+namePart[1]] = make(map[string]string)
		md5map[namePart[0]+"_"+namePart[1]][nameFile] = md5
		return true, md5map
	} else {
		if strings.EqualFold(filepath.Ext(path), ".zip") {
			nameFile := filepath.Base(path)
			namePart := strings.Split(nameFile, "_")
			if len(namePart) != 3 {
				return false, nil
			}
			zipPath := filepath.Join(config.HttpMasterDir, namePart[0], namePart[1], nameFile+".zip")
			if CopyFileFull(path, zipPath) != nil {
				return false, nil
			}
			md5, _ := GetMD5(zipPath)
			md5map[namePart[0]+"_"+namePart[1]] = make(map[string]string)
			md5map[namePart[0]+"_"+namePart[1]][nameFile] = md5
			return true, md5map
		}
	}
	return false, nil
}

func zipFolder(source, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(source, path)
		w, err := archive.Create(relPath)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		return err
	})
}

var run = true
var start = false

func Cmd(mode bool, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // Ẩn cửa sổ
		CreationFlags: 0x08000000, // Không tạo cửa sổ mới
	}
	if mode {
		return cmd.Run()
	} else {
		return cmd.Start()
	}
}

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

func IsRunning(exe string) bool {
	out, err := exec.Command(
		"tasklist",
		"/FI", "IMAGENAME eq "+exe,
		"/NH",
	).Output()
	if err != nil {
		return false
	}

	return strings.Contains(strings.ToLower(string(out)), strings.ToLower(exe))
}
