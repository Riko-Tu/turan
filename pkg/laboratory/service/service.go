package service

import (
	"TEFS-BE/pkg/tencentCloud"
	"fmt"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"net"
	"time"
)

type Service struct {
}

var (
	cosService tencentCloud.Cos
	cvmService tencentCloud.Cvm
	cosBaseUrl string
	cloudAccount string
	cloudProjectId int64
	InstanceId string
	InstancePassWord string
	Region string
)

func Setup() {
	// cos
	cloudRegion := viper.GetString("tencentCloud.region")
	cosBucket := viper.GetString("tencentCloud.cos.bucket")
	cloudAppId := viper.GetString("tencentCloud.appId")
	cloudSecretId := viper.GetString("tencentCloud.secretId")
	cloudSecretKey := viper.GetString("tencentCloud.secretKey")
	cloudAccount = viper.GetString("tencentCloud.account")
	cloudProjectId = viper.GetInt64("tencentCloud.projectId")
	InstanceId = viper.GetString("tencentCloud.instanceId")
	InstancePassWord = viper.GetString("tencentCloud.instancePassWord")
	Region = viper.GetString("tencentCloud.region")
	cosBaseUrl = fmt.Sprintf("cos://%s.cos.%s.myqcloud.com", cosBucket, cloudRegion)
	cosService = tencentCloud.Cos{
		Credential: &tencentCloud.Credential{
			AppId:     cloudAppId,
			SecretId:  cloudSecretId,
			SecretKey: cloudSecretKey,
		},
		Region: cloudRegion,
		Bucket: cosBucket,
	}
	cvmService = tencentCloud.Cvm{
		Credential: &tencentCloud.Credential{
			AppId:     cloudAppId,
			SecretId:  cloudSecretId,
			SecretKey: cloudSecretKey,
		},
		Region: cloudRegion,
	}
}


func getSSHClient(user, password, address string, timeout int) (client *ssh.Client, err error) {
	auth := make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))
	clientConfig := &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: time.Duration(timeout) * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	return ssh.Dial("tcp", address, clientConfig)
}

func execSSHCmd (client *ssh.Client, cmd string) ([]byte, error) {
	session,err := client.NewSession()
	if err != nil {
		return nil, err
	}
	return session.CombinedOutput(cmd)
}

func StartNFS() error {
	// 获取内网ip
	instanceList, err := cvmService.GetCvmForPublicId(InstanceId)
	if err != nil {
		return err
	}
	if len(instanceList) != 1 {
		return fmt.Errorf("query cvm info failed")
	}
	privateIp := *instanceList[0].PrivateIpAddresses[0]
	address := fmt.Sprintf("%s:22", privateIp)
	cli, err := getSSHClient("ubuntu", InstancePassWord, address, 180)
	if err != nil {
		return err
	}

	// 安装nfs
	_, err = execSSHCmd(cli, "sudo apt-get update && sudo apt-get install nfs-kernel-server")
	if err != nil {
		return err
	}

	// 更改问加权限
	_, err = execSSHCmd(cli, "sudo chmod 777 /etc/exports && sudo chmod -R 777 /data")
	if err != nil {
		return err
	}

	// nfs相关文件
	_, err = execSSHCmd(cli,`sudo chmod 777 /etc/exports && sudo chmod -R 777 /data && sudo echo "/data/users *(rw,sync,fsid=0,no_root_squash)" > /etc/exports`)
	if err != nil {
		return err
	}

	// 启动nfs服务
	_, err = execSSHCmd(cli,"sudo /etc/init.d/nfs-kernel-server restart")
	if err != nil {
		return err
	}
	return nil
}