package utils

import (
	"TEFS-BE/pkg/log"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

type Dir struct {
	Name  string `json:"name"`
	Dirs  []Dir  `json:"dirs"`
	Files []File `json:"files"`
	Id    string `json:"id"`
}

type File struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Id   string `json:"id"`
}

func getFileType(filePath string) (fileType string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Error(err.Error())
		return
	}
	buff := make([]byte, 512)
	_, err = file.Read(buff)
	if err != nil {
		log.Error(err.Error())
		return
	}
	return http.DetectContentType(buff)
}

func GetSha1(data string) string {
	t := sha1.New()
	io.WriteString(t, data)
	return fmt.Sprintf("%x", t.Sum(nil))
}

func DirTree(dirPath string, dir *Dir) error {
	fileInfoItems, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, v := range fileInfoItems {
		if v.IsDir() {
			currDir := Dir{
				Name: v.Name(),
				Id:   GetSha1(path.Join(dirPath, v.Name())),
			}
			if err := DirTree(filepath.Join(dirPath, v.Name()), &currDir); err != nil {
				return err
			}
			dir.Dirs = append(dir.Dirs, currDir)
		} else {
			dir.Files = append(dir.Files, File{
				Name: v.Name(),
				Type: getFileType(filepath.Join(dirPath, v.Name())),
				Id:   GetSha1(path.Join(dirPath, v.Name())),
			})
		}
	}
	return nil
}

func GetTree(path string) (ret string, err error) {
	tree := &Dir{
		Name: ".",
	}
	if err = DirTree(path, tree); err != nil {
		return
	}
	retByte, err := json.Marshal(tree)
	if err != nil {
		return
	}
	ret = string(retByte)
	return
}