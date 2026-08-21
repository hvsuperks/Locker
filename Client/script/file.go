package script

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

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
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err != nil {
		out.Close()
		return err
	}
	out.Close()
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

func RemoveEmptyDirs(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Chỉ xử lý folder
		if !info.IsDir() {
			return nil
		}

		// Không xóa root
		if path == root {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		if len(entries) == 0 {
			_ = os.Remove(path)
		}

		return nil
	})
}
