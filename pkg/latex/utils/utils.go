package utils

import (
	"archive/zip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	DirType = "dir"
	FileType = "file"
)

// Does the path exist
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func GetFileType(file string) string {
	if IsDir(file) {
		return DirType
	}
	return FileType
}

// Gets the size of the given directory
func GetDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

//RSA加密
func RsaEncrypt(plainText []byte, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, _ := file.Stat()
	buf := make([]byte, info.Size())
	file.Read(buf)
	//pem解码
	block, _ := pem.Decode(buf)
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}
	publicKey := publicKeyInterface.(*rsa.PublicKey)
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, plainText)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// RSA解密
func RsaDecrypt(cipherText string, path string) (string, error) {
	cipherByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, _ := file.Stat()
	buf := make([]byte, info.Size())
	file.Read(buf)
	block, _ := pem.Decode(buf)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	plainText, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, cipherByte)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}

// 解压zip
// 不带zip最外层文件夹名字
func Unzip(zipFile, dest string) error {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer reader.Close()

	zipName := strings.Split(filepath.Base(zipFile), ".zip")[0]
	firstFileCount := 0
	var firstFileName string
	for _, f := range reader.File {
		tmp := strings.Split(f.Name, "/")
		fmt.Println(tmp)
		if len(tmp) == 1 {
			firstFileCount += 1
			firstFileName = tmp[0]
		}
		if len(tmp) == 2 && tmp[1] == "" {
			firstFileCount += 1
			firstFileName = tmp[0]
		}
	}

	var isSkip bool
	if firstFileCount == 1 && firstFileName == zipName {
		isSkip = true
	}

	var fileName string
	for i, f := range reader.File {
		if isSkip && i == 0 {
			continue
		}
		if isSkip {
			fileName = strings.Join(strings.Split(f.Name, "/")[1:], "/")
		} else {
			fileName = f.Name
		}

		fpath := filepath.Join(dest, fileName)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			inFile, err := f.Open()
			if err != nil {
				return err
			}
			defer inFile.Close()

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, inFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// zip压缩
func ZipCompressor(target, filter string, sources ...string) error {
	var err error
	zipfile, err := os.Create(target)
	if err != nil {
		return errors.WithStack(err)
	}

	defer zipfile.Close()
	zw := zip.NewWriter(zipfile)
	defer zw.Close()

	for _, source := range sources {
		if isAbs := filepath.IsAbs(source); !isAbs {
			source, err = filepath.Abs(source) // 将传入路径直接转化为绝对路径
			if err != nil {
				return errors.WithStack(err)
			}
		}

		info, err := os.Stat(source)
		if err != nil {
			return errors.WithStack(err)
		}

		var baseDir string
		if info.IsDir() {
			//baseDir = filepath.Base(source) // 包含最外层路径
			baseDir = "" // 不包含最外层路径
		}

		err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
			if path == source {
				return nil
			}
			if err != nil {
				return errors.WithStack(err)
			}
			ism, err := filepath.Match(filter, info.Name())
			if err != nil {
				return errors.WithStack(err)
			}
			if ism {
				return nil
			}
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return errors.WithStack(err)
			}
			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			} else {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))[1:]
			}
			if info.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}
			writer, err := zw.CreateHeader(header)
			if err != nil {
				return errors.WithStack(err)
			}

			if info.IsDir() {
				return nil
			}
			file, err := os.Open(path)
			if err != nil {
				return errors.WithStack(err)
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			return errors.WithStack(err)
		})

		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
