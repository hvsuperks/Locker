package script

import (
	"Locker/config"
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
