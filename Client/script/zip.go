package script

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

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

// Hàm nén folder
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
