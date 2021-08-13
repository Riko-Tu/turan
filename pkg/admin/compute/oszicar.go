package compute

import (
	"TEFS-BE/pkg/admin/model"
	"TEFS-BE/pkg/laboratory/client"
	"TEFS-BE/pkg/tencentCloud"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	IonicRowRe        = regexp.MustCompile(`[0-9]+[ ]{1}F=.*$`)
	IonicIterationsRe = regexp.MustCompile(`[0-9]+[ ]{1}F=`)
	IonicEnergyRe     = regexp.MustCompile(`E0=[ ]+.*[ ]+d[ ]+E`)

	ElectronRowRe        = regexp.MustCompile(`^[A-Z0-9]+:.*$`)
	ElectronIterationsRe = regexp.MustCompile(`[A-Z0-9]+:[ ]+\d+`)
	ElectronEnergyRe     = regexp.MustCompile(`[A-Z0-9]+:[ ]+\d+[ ]+[-.+0-9A-Z]+`)
)

const OszicarName = "OSZICAR"
const ExpDict = "expDict.json"
const Output = "output"

// 实验结果文件OSZICAR
type OSZICAR struct {
	Path         string   `json:"-"`
	IonicStep    string   `json:"ionicStep"`
	ElectronStep string   `json:"electronStep"`
	Energy       string   `json:"energy"`
	Algorithm    string   `json:"algorithm"`
	Data         []string `json:"-"`
}

// 实验结果文件expDict.json
type ExpDictJson struct {
	IonicStep int `json:"ionicStep"`
	ElectronStep int `json:"electronStep"`
	Energy float64 `json:"energy"`
	Time string `json:"time"`
}

// todo:(v_vwwwang)单元测试
// 实验获取cos cli
func getCosCliForExperiment(experiment *model.Experiment) (cli *cos.Client, e error) {
	labAddress := experiment.LaboratoryAddress
	labCli, e := client.GetClient(labAddress)
	if e != nil {
		return nil, e
	}
	cosCredential, cosBaseUrl, e := client.GetCosDownloadTmpSecret(labCli, experiment.UserId)
	if e != nil {
		return nil, e
	}
	cosUrl := strings.Replace(*cosBaseUrl, "cos:", "http:", 1)
	cosCli, e := tencentCloud.TmpSecretGetCosCli(cosCredential.TmpSecretID,
		cosCredential.TmpSecretKey,
		cosCredential.SessionToken,
		cosUrl)
	if e != nil {
		return nil, e
	}
	if cosCli == nil {
		return nil, fmt.Errorf("cos cli is nil")
	}
	return cosCli, nil
}

// cos获取实验文件
func GetExperimentFileForCos(experiment *model.Experiment, localPath, cosFileName string) error {
	cosCli, e := getCosCliForExperiment(experiment)
	if e != nil {
		return e
	}
	key := fmt.Sprintf("users/%d/experiments/%d/%s", experiment.UserId, experiment.Id, cosFileName)
	_, e = cosCli.Object.GetToFile(context.Background(), key, localPath, nil)
	if e != nil {
		return e
	}
	return nil
}

func ReadAll(filePth string) ([]byte, error) {
	f, err := os.Open(filePth)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

func (o *OSZICAR) ReadData() error {
	dataByte, e := ReadAll(o.Path)
	if e != nil {
		return e
	}
	data := strings.Split(string(dataByte), "\n")
	o.Data = data
	return nil
}

func (o *OSZICAR) ReadIterationsAndEnergyAndAlgorithm() {
	var lastIonicRow, lastElectronRow string
	var lastIonicRowNum, lastElectronRowNum int
	for i := len(o.Data); i >= 1; i-- {
		if IonicRowRe.MatchString(o.Data[i-1]) {
			if lastIonicRowNum == 0 {
				lastIonicRowNum = i - 1
				lastIonicRow = o.Data[i-1]
			}
		}
		if ElectronRowRe.MatchString(o.Data[i-1]) {
			if lastElectronRowNum == 0 {
				lastElectronRowNum = i - 1
				lastElectronRow = o.Data[i-1]
			}
		}
	}

	if lastIonicRowNum == 0 && lastElectronRowNum == 0 {
		o.ElectronStep = "0"
		o.IonicStep = "1"
		o.Energy = "0"
		return
	}

	if lastIonicRowNum > 0 {
		ionicIterationsStr := IonicIterationsRe.FindAllString(lastIonicRow, -1)
		if len(ionicIterationsStr) > 0 {
			o.IonicStep = strings.Split(ionicIterationsStr[0], " ")[0]
		}
	} else {
		o.IonicStep = "1"
	}

	electronIterationsStr := ElectronIterationsRe.FindAllString(lastElectronRow, -1)
	if len(electronIterationsStr) > 0 {
		electronItems := strings.Split(electronIterationsStr[0], ":")
		o.Algorithm = electronItems[0]
	}

	if lastIonicRowNum > lastElectronRowNum {
		o.ElectronStep = "0"
		ionicEnergyStr := IonicEnergyRe.FindAllString(lastIonicRow, -1)
		if len(ionicEnergyStr) > 0 {
			o.Energy = strings.Split(ionicEnergyStr[0], " ")[1]
		}
	} else {
		ionicStepInt, _ := strconv.Atoi(o.IonicStep)
		o.IonicStep = strconv.Itoa(ionicStepInt + 1)
		if len(electronIterationsStr) > 0 {
			items := strings.Split(electronIterationsStr[0], " ")
			o.ElectronStep = items[len(items)-1]

		}
		electroEnergyStr := ElectronEnergyRe.FindAllString(lastElectronRow, -1)
		if len(electroEnergyStr) > 0 {
			items := strings.Split(electroEnergyStr[0], " ")
			o.Energy = items[len(items)-1]
		}
	}
}

func (o OSZICAR) GetJson() (jsonStr string, e error) {
	jsonByte, e := json.Marshal(o)
	if e != nil {
		return "", e
	}
	return string(jsonByte), nil
}

// 读取exp dict json
func (o OSZICAR) ReadExpDictJson() (exp *ExpDictJson, err error) {
	dataByte, e := ReadAll(o.Path)
	if e != nil {
		return nil, e
	}
	exp = &ExpDictJson{}
	if e = json.Unmarshal(dataByte, exp); e != nil {
		return nil, e
	}
	return exp, nil
}

// 计算成功的log文件最后一行正则
var LastLineCorrectRe = regexp.MustCompile(`^ +[0-9]+ +[F|T]=.*$`)

// 获取实验log最后一行数据
func GetLastLine(fileName string) (lastLine string, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var lineText string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineText = scanner.Text()
	}
	return string(lineText), nil
}

// 检查最后一行输出是否符合正则预期
func Check(LastLine string, re *regexp.Regexp) bool {
	return re.MatchString(LastLine)
}