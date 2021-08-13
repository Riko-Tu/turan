package utils

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"testing"
)

const (
	publicKey = "-----BEGIN PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDlaMKILaYSKNjBanIgq2REgEQ4\nsvMqhUCa4eUKsqMrWg/RY+52CIe92C0GHwqVmdGLdLtHVdMYhYNOHwbYDyizNgVB\nRkVdPSOqwLxeUNIO/FTk7M21KzIZGNO1a7PTowfUmZBkpFO1N0a1Tmm7wtqF9551\nT3OXsa3b7C/fca89+QIDAQAB\n-----END PUBLIC KEY-----"
	privateKey = "-----BEGIN RSA PRIVATE KEY-----\nMIICXAIBAAKBgQDlaMKILaYSKNjBanIgq2REgEQ4svMqhUCa4eUKsqMrWg/RY+52\nCIe92C0GHwqVmdGLdLtHVdMYhYNOHwbYDyizNgVBRkVdPSOqwLxeUNIO/FTk7M21\nKzIZGNO1a7PTowfUmZBkpFO1N0a1Tmm7wtqF9551T3OXsa3b7C/fca89+QIDAQAB\nAoGAOyaR0g8DHPePO/+4QZgvmEICVSQ+8p29FLJeHi4FSG5GWdUMbT6x0U9l/IgQ\ncJZiozSL/U6xyUbTnlb9qsPt2BqCQWsMJiXHBzHqkzcEXO4eZcJ/B9yhwWhf9iSR\nZnTVfpEdOIkBPQE1aoOGJ7nuFeqW3lGjbCu1yiqaoGM7PUkCQQDz+xZuDAzJy4DR\nql3EYKUV6nWVSXsMFzh1HHiqIVCzgavEYBRlBkBHJcxyzcKjLMjqMsf520a/MGOJ\ncC1+dgwTAkEA8LXn3PUIEPl+XTZYJoWnjVh4HXspfgI2kRdW3cH5pTNE8QG/n+aP\nSq/tW7SG7IF5K+PUt0jIdhyLUSIAdUY3QwJBALvrU3Vjlp3/PrM/E4XkIoNk2Tgp\nJrtDT1r0mQQBMRVx9QkGL+84B15FgNmUHixsnDu27UxHVpCABsqfOotDBT0CQGm1\nqS67GSDDQMBUtl+sgImtWYqw5Obmt+n+EvLuVeE748Hnn6zsRu9o1VdZr4s7zOf+\ndRNMzmQ4YuJtiT/3ZxsCQHTeAQkoXnjaOic0fYbS8jQAXxcYQky3boZ5rrty9SOP\ngtepBIqQkCEcBPAs0Bm/jEJTW9a7LqD530kCVrN02SY=\n-----END RSA PRIVATE KEY-----"
)

func TestGeneratePassWord(t *testing.T) {
	//密码长度
	length := 16
	passWord := GeneratePassWord(length)
	fmt.Println("generated password:", passWord)
	//第一位是字母，第二位是数字，其余位数随机(由length决定长度)
	reg := "[A-Za-z]\\d[A-Za-z0-9!#$%^,;*]{" + fmt.Sprintf("%d", length - 2) + "}"
	matched, err := regexp.MatchString(reg, passWord)
	if err != nil {
		//正则表达式语法错误
		t.Error(`regexp: Compile(` + reg + `): ` + err.Error())
	}
	if !matched {
		t.Error("GeneratePassWord validation error!")
	}
}

func TestRsaEncrypt(t *testing.T) {
	// 需要加密的数据以及密钥
	data := "this is a test message"
	cipherText, err := RsaEncrypt(data, publicKey)
	if err != nil {
		t.Error("Rsa Encrypt Error: " + err.Error())
	}
	fmt.Println("the result of encrypt Data is", cipherText)
}

func TestRsaDecrypt(t *testing.T) {
	// 需要加密的数据
	rawData := "this is a test message jlfsdkal"
	//公钥
	cipherText, err := RsaEncrypt(rawData, publicKey)
	if err != nil {
		t.Error("Rsa Encrypt Error: " + err.Error())
	}
	// 私钥
	msg, err := RsaDecrypt(cipherText, privateKey)
	if err != nil {
		t.Error("Rsa Decrypt Error: " + err.Error())
	}
	if msg != rawData {
		t.Error("encrypt & decrypt error!")
	}
}

func TestPathExists(t *testing.T) {
	var testPaths = []struct {
		// 文件所在路径
		Path string
		// 文件是否存在
		Expected bool
	}{
		{
			Path: "D:\\env\\pem\\rsa_public_key.pem",
			Expected: true,
		}, // exists
		{
			Path: "D:\\env123\\pem\\rsa_public_key.pem",
			Expected: false,
		}, // none-exists
	}
	for _, test := range testPaths {
		if isExist, err := PathExists(test.Path); isExist != test.Expected {
			// 非法路径error
			if err != nil {
				t.Errorf("path error:" + err.Error())
			}
			t.Errorf("path(%v) exists(%v) test result: %v", test.Path, test.Expected, isExist)
		}
	}
}

func TestAddcslashes(t *testing.T) {
	// 需要转义的特殊字符
	const SpecialStr = `*.?+$^[](){}|\/`
	raw := "this is a raw message:*.?+$^[](){}|\\/"
	ret := Addcslashes(raw, SpecialStr)
	fmt.Printf("add cslashes result:\"%s\"\n", ret)
}

// 从文件中读取字符串，公钥or密钥
func ReadFile(filePath string) (ret string, err error) {
	content ,err := ioutil.ReadFile(filePath)
	if err !=nil {
		fmt.Printf("open file error," + err.Error())
	}
	return string(content), nil
}

