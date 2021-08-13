package controllers

import (
	"TEFS-BE/pkg/creator/api/kube"
	"fmt"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// api错误code msg
type ControllerError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

var (
	BasePrefix = "tefs"

	GlobalRegion = "ap-nanjing"

	ErrQueryCloud                     = &ControllerError{Code: 1001, Message: "腾讯云查询失败"}
	ErrSubmitCloud                    = &ControllerError{Code: 1002, Message: "腾讯云提交失败"}
	ErrNotAvailableZone               = &ControllerError{Code: 1003, Message: "没有可用的zone"}
	ErrNotSetSecurityGroup            = &ControllerError{Code: 1004, Message: "腾讯云安全组未创建"}
	ErrQuerySecurityGroup            = &ControllerError{Code: 1004, Message: "腾讯云查询安全组出错"}
	ErrDelSecurityGroup            = &ControllerError{Code: 1004, Message: "腾讯云删除安全组规则出错"}
	ErrParamAppId                     = &ControllerError{Code: 1005, Message: "AppId格式错误,AppId是一个10位整数"}
	ErrParamProjectId                 = &ControllerError{Code: 1006, Message: "projectId错误"}
	ErrNotFoundProject                = &ControllerError{Code: 1007, Message: "该腾讯账户下无此项目"}
	ErrVpcInsufficientQuota           = &ControllerError{Code: 1008, Message: "腾讯云账户VPC地区配额不足"}
	ErrSecurityGroupInsufficientQuota = &ControllerError{Code: 1009, Message: "腾讯云账户SecurityGroup配额不足"}
	ErrCosBucketInsufficientQuota     = &ControllerError{Code: 1010, Message: "腾讯云账户CosBucket配额不足"}
	ErrCosBucketSetCorsFailed         = &ControllerError{Code: 1010, Message: "设置CosBucketCORS失败"}
	ErrNotFoundInstanceForCluster     = &ControllerError{Code: 1011, Message: "腾讯云TKE集群还未创建实例"}
	ErrInstanceNotPublicIpAddress     = &ControllerError{Code: 1012, Message: "腾讯云TKE集群实例没有公网ip"}
	ErrSshInstanceFailed              = &ControllerError{Code: 1013, Message: "连接服务器实例失败"}
	ErrUploadYamlFailed               = &ControllerError{Code: 1013, Message: "上传yaml文件失败"}
	ErrExecRemoteCmdFailed            = &ControllerError{Code: 1013, Message: "执行远程指令失败"}

	CurrentVersion     = "1.0"
	TefsKubeSecret     = kube.NewSecret()
	TefsKubeSecretYaml string
	configDir = "tefs_kube"

	remoteDir = "/home/ubuntu"
	yamlFiles = []string{
		"config.yaml",
		"deployment.yaml",
	}
	remoteCmds = []string{
		"sudo kubectl apply -f  /home/ubuntu/config.yaml",
		"sudo kubectl apply -f  /home/ubuntu/deployment.yaml",
	}
)

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

func init() {
	TefsYamlBasePath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	configDirPath := filepath.Join(TefsYamlBasePath, configDir)
	isExists,err := PathExists(configDirPath)
	if err != nil {
		panic(err)
	}
	if !isExists {
		if err = os.Mkdir(configDirPath, os.ModePerm); err != nil {
			panic(err)
		}
	}

	for i := range yamlFiles {
		yamlFiles[i] = filepath.Join(TefsYamlBasePath, configDir, yamlFiles[i])
		isExists, err := PathExists(yamlFiles[i])
		if err != nil {
			panic(err)
		}
		if !isExists {
			if err := ioutil.WriteFile(yamlFiles[i], []byte(""), 0644); err != nil {
				panic(err)
			}
		}
	}

	TefsKubeSecretYaml = yamlFiles[0]
	if err = ioutil.WriteFile(yamlFiles[1], []byte(kube.Deployment), 0644); err != nil {
		panic(err)
	}
	if err := kube.InitSecretFormYaml(TefsKubeSecretYaml, TefsKubeSecret); err != nil {
		panic(err)
	}
}

// 腾讯云api控制器
type CloudController struct {
}

func generateRandomName(prefix string, notAvailableNames []string) string {
	var randomSuffix string
	var randomName string
	var nameIsAvailable bool
	for {
		randomSuffix = strings.ToLower(strings.ReplaceAll(uuid.NewV4().String(), "-", "")[0:6])
		randomName = fmt.Sprintf("%s-%s-%s", BasePrefix, prefix, randomSuffix)
		nameIsAvailable = true
		for _, name := range notAvailableNames {
			if name == randomName {
				nameIsAvailable = false
				break
			}
		}
		if nameIsAvailable {
			break
		}
	}
	return randomName
}

func resp(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": data,
	})
}

func fail(c *gin.Context, errs *ControllerError) {
	c.JSON(http.StatusOK, gin.H{
		"code": errs.Code,
		"msg":  errs.Message,
	})
}

func failCloudMsg(c *gin.Context, errs *ControllerError, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code": errs.Code,
		"msg":  errs.Message + ":" + msg,
	})
}

// @Summary 创建腾讯云应用程序版本
// @Tags 腾讯云环境
// @Description 获取创建腾讯云应用程序版本接口
// @Produce  json
// @Success 200 {string} json "{"code":200,"data":"1.0"}"
// @Router /cloudEnv/version [get]
func (cc CloudController) Version(c *gin.Context) {
	resp(c, CurrentVersion)
}