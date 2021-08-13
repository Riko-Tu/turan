package kube

import (
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
)

type Secret struct {
	ApiVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   SecretMetadata `yaml:"metadata"`
	Type       string         `yaml:"type"`
	Data       CloudEnv       `yaml:"data"`
}

type SecretMetadata struct {
	Name string `yaml:"name"`
}

type CloudEnv struct {
	SecretId         string `yaml:"SecretId,omitempty"`
	SecretKey        string `yaml:"SecretKey,omitempty"`
	Region           string `yaml:"Region,omitempty"`
	Zone             string `yaml:"Zone,omitempty"`
	AppId            string `yaml:"AppId,omitempty"`
	CosBucket        string `yaml:"CosBucket,omitempty"`
	SecurityGroupId  string `yaml:"SecurityGroupId,omitempty"`
	Account          string `yaml:"Account,omitempty"`
	ProjectId        string `yaml:"ProjectId,omitempty"`
	VpcId            string `yaml:"VpcId,omitempty"`
	SubnetId         string `yaml:"SubnetId,omitempty"`
	TkeId            string `yaml:"TkeId,omitempty"`
	InstanceId       string `yaml:"InstanceId,omitempty"`
	InstanceIp       string `yaml:"InstanceIp,omitempty"`
	InstancePassWord string `yaml:"InstancePassWord,omitempty"`
}

func NewSecret() *Secret {
	return &Secret{
		ApiVersion: "v1",
		Kind:       "Secret",
		Metadata: SecretMetadata{
			Name: "tefs-laboratory-config",
		},
		Type: "Opaque",
	}
}

func InitSecretFormYaml(filePath string, secret *Secret) error {
	// 读取yaml文件
	bytes, e := ioutil.ReadFile(filePath)
	if e != nil {
		return e
	}
	// yaml转结构体
	if e = yaml.Unmarshal(bytes, &secret); e != nil {
		return e
	}
	return CloudEnvB64(&secret.Data, "decode")
}

func CloudEnvB64(cloudEnv *CloudEnv, op string) error {
	var key = reflect.TypeOf(*cloudEnv)
	var val = reflect.ValueOf(*cloudEnv)
	for i := 0; i < key.NumField(); i++ {
		if len(val.Field(i).String()) > 0 {
			var data string
			switch op {
			case "decode": // 解码
				dataByte, _ := base64.StdEncoding.DecodeString(val.Field(i).String())
				data = string(dataByte)
			case "encode": // 编码
				data = base64.StdEncoding.EncodeToString([]byte(val.Field(i).String()))
			default:
				return fmt.Errorf("invalid op")
			}
			switch key.Field(i).Name {
			case "SecretId":
				cloudEnv.SecretId = data
			case "SecretKey":
				cloudEnv.SecretKey = data
			case "Region":
				cloudEnv.Region = data
			case "Zone":
				cloudEnv.Zone = data
			case "AppId":
				cloudEnv.AppId = data
			case "CosBucket":
				cloudEnv.CosBucket = data
			case "SecurityGroupId":
				cloudEnv.SecurityGroupId = data
			case "Account":
				cloudEnv.Account = data
			case "ProjectId":
				cloudEnv.ProjectId = data
			case "VpcId":
				cloudEnv.VpcId = data
			case "SubnetId":
				cloudEnv.SubnetId = data
			case "TkeId":
				cloudEnv.TkeId = data
			case "InstanceId":
				cloudEnv.InstanceId = data
			case "InstanceIp":
				cloudEnv.InstanceIp = data
			case "InstancePassWord":
				cloudEnv.InstancePassWord = data
			}
		}
	}
	return nil
}

func (s Secret) Write(yamlFile string) error {
	if err := CloudEnvB64(&s.Data, "encode"); err != nil {
		return err
	}

	out, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(yamlFile, out, 0644)
	//// 覆盖写入
	//fd, err := os.OpenFile(yamlFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	//_, err = fd.Write(out)
	return err
}
