package utils

import (
	cRand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	xj "github.com/basgys/goxml2json"
	"math/rand"
	"os"
	"strings"
	"time"
)

// 生产密码,开头一位字母，第二位数字,其余位数随机
func GeneratePassWord(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%^,;*")
	letterChars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	intChars := []rune("0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		switch i {
		case 0:
			b.WriteRune(letterChars[rand.Intn(len(letterChars))])
		case 1:
			b.WriteRune(intChars[rand.Intn(len(intChars))])
		default:
			b.WriteRune(chars[rand.Intn(len(chars))])
		}
	}
	passWord := b.String()
	return passWord
}

// 判断文件是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// rsa加密
// todo:(v_vwwwang) test
func RsaEncrypt(origData, publicKey string) (string, error) {
	//解密pem格式的公钥
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return "", errors.New("public key error")
	}
	// 解析公钥
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}
	// 类型断言
	pub := pubInterface.(*rsa.PublicKey)
	//加密
	ret, err := rsa.EncryptPKCS1v15(cRand.Reader, pub, []byte(origData))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ret), nil
}

// rsa解密
// todo:(v_vwwwang) test
func RsaDecrypt(ciphertext, privateKey string) (string, error) {
	// base64解码
	cipherByte, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	//解密
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return "", errors.New("private key error!")
	}
	//解析PKCS1格式的私钥
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	// 解密
	ret, err := rsa.DecryptPKCS1v15(cRand.Reader, priv, cipherByte)
	if err != nil {
		return "", err
	}
	return string(ret), nil
}

func findDir(nodes xj.Nodes, fileType, name string) xj.Nodes {
	var currentDirName string
	for _, n := range nodes {
		obj, ok := n.Children["-name"]
		if ok {
			currentDirName = obj[0].Data
			currentFile, ok := n.Children[fileType]
			if currentDirName == name && ok {
				return currentFile
			}
		}
	}
	return nil
}

func find(tree xj.Nodes, filePath []string) (isFind bool) {
	n := len(filePath)
	switch n {
	case 0:
		return
	case 1:
		for _, node := range tree {
			obj, ok := node.Children["-name"]
			if ok {
				if len(obj) > 0 && obj[0].Data == filePath[0] {
					return true
				}
			}
		}
	default:
		fileType := "directory"
		if len(filePath) == 2 {
			fileType = "file"
		}
		dir := findDir(tree, fileType, filePath[0])
		if dir != nil {
			return find(dir, filePath[1:])
		}
	}
	return
}

// 查询文件是否存在
func FindFile(fileXml, filePath string) (isSuccess bool, err error) {
	root := &xj.Node{}
	xml := strings.NewReader(fileXml)
	if err = xj.NewDecoder(xml).Decode(root); err != nil {
		return false, err
	}
	paths := strings.Split(filePath, "/")
	isSuccess = find(root.Children["tree"][0].Children["directory"], paths)
	return isSuccess, nil
}

// 需要转义的特殊字符
const SpecialStr = `*.?+$^[](){}|\/`

// 对特殊字符添加转义
func Addcslashes(raw, specialStr string) (ret string) {
	for _, v := range raw {
		if strings.Contains(specialStr, string(v)) {
			ret += `\`
			ret += string(v)
		} else {
			ret += string(v)
		}
	}
	return ret
}

// RSA解密
func Decrypt(cipherText string, path string) (string, error) {
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
	plainText, err := rsa.DecryptPKCS1v15(cRand.Reader, privateKey, cipherByte)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}

// 解密参数
func GetEncryptParams(cipherText string, path string) (params map[string]string, err error) {
	ret, err := Decrypt(cipherText, path)
	if err != nil {
		return nil, err
	}
	params = make(map[string]string)
	paramsSplit := strings.Split(ret, "&")
	for _, v := range paramsSplit {
		t := strings.Split(v, "=")
		if len(t) == 2 {
			params[t[0]] = t[1]
		}
	}
	return
}