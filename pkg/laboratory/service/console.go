package service

import (
	pb "TEFS-BE/pkg/laboratory/proto"
	"TEFS-BE/pkg/utils/ssh"
	"context"
	"fmt"
	"strconv"
	"strings"
)

const (
	user           = "ubuntu"
	containerNum   = "echo `sudo docker ps -a | grep -c -w tefs-shell-user%d`"
	containerIsRun = "sudo docker inspect -f {{.State.Running}} tefs-shell-user%d"
	newContainer   = "sudo docker run -d --cap-add SYS_ADMIN --device /dev/fuse --privileged --net host -v /data/users/%d:/root/home -e TEFS_GOTTY_PORT=%s -e TEFS_FILESERVER_PORT=%s -e COS_BUCKENT=%s -e EXPERIMENTS_COS_PATH=%s -e COS_ORUL=%s -e SECRET_ID=%s -e SECRET_KEY=%s -e TEFS_SECRET=%s -e TEFS_USER_ID=%d -e TEFS_PROJECT_ID=%d -e TEFS_URL=%s --name tefs-shell-user%d ccr.ccs.tencentyun.com/tefs/shell"
	startContainer = "echo `sudo docker start tefs-shell-user%d`"

	// 获取容器已使用的端口(包括已停止的container)
	containerUsePort = `sudo docker ps -a | grep tefs-shell-user | awk -F "0.0.0.0:" '{ print $2}' | awk -F "->" '{ print $1 }'`
)

func getContainerUsePort(cli *ssh.Cli) (ports []int, err error) {
	var ret []byte
	ret, err = cli.Run(containerUsePort)
	if err != nil {
		return
	}
	retList := strings.Split(string(ret), "\n")
	for _, v := range retList {
		port, err := strconv.Atoi(v)
		if err == nil {
			ports = append(ports, port)
		}
	}
	return
}

func portIsUse(port int, ports []int) bool {
	for _, v := range ports {
		if v == port {
			return true
		}
	}
	return false
}

func GetRandomPort(cli *ssh.Cli) (wettyPort, fileServerPort int, err error) {
	var randomPort = 5000
	var maxPort = 8000

	// 获取容器已使用的port,包括已停止的容器
	usePorts, err := getContainerUsePort(cli)
	if err != nil {
		return 0, 0, err
	}
	for randomPort <= maxPort {
		targetPort := strconv.Itoa(randomPort)
		TCPListeningnum := "`" + `netstat -an | grep ":` + targetPort + `" | awk '$1 == "tcp" && $NF == "LISTEN" {print $0}' | wc -l` + "`"
		TCPL6isteningnum := "`" + `netstat -an | grep ":` + targetPort + `" | awk '$1 == "tcp6" && $NF == "LISTEN" {print $0}' | wc -l` + "`"
		UDPListeningnum := "`" + `netstat -an | grep ":` + targetPort + ` " | awk '$1 == "udp" && $NF == "0.0.0.0:*" {print $0}' | wc -l` + "`"
		UDP6Listeningnum := "`" + `netstat -an | grep ":` + targetPort + ` " | awk '$1 == "udp6" && $NF == "0.0.0.0:*" {print $0}' | wc -l` + "`"
		cmd := fmt.Sprintf("expr %s + %s + %s + %s + 1", TCPListeningnum, TCPL6isteningnum, UDPListeningnum, UDP6Listeningnum)
		useNumByte, err := cli.Run(cmd)
		if err != nil {
			return 0, 0, err
		}
		useNum := string(useNumByte)
		useNum = strings.Split(useNum, " ")[0]
		useNum = strings.Split(useNum, "\n")[0]
		if useNum == "1" && !portIsUse(randomPort, usePorts) {
			if wettyPort == 0 {
				wettyPort = randomPort
			} else {
				fileServerPort = randomPort
			}
		}
		if wettyPort != 0 && fileServerPort != 0 {
			return wettyPort, fileServerPort, nil
		}
		randomPort += 1
	}
	err = fmt.Errorf("no ports available")
	return
}

func (s Service) CreateWebConsole(ctx context.Context,
	in *pb.CreateWebConsoleRequest) (*pb.CreateWebConsoleReply, error) {

	userId := in.GetUserId()
	projectId := in.GetProjectId()
	secret := in.GetSecret()
	tefsUrl := in.GetTefsUrl()
	if len(secret) == 0 {
		return nil, fmt.Errorf("invalid secret")
	}
	if userId <= 0 {
		return nil, fmt.Errorf("invalid userId")
	}
	if projectId <= 0 {
		return nil, fmt.Errorf("invalid projectId")
	}

	// 获取内网ip
	instanceList, err := cvmService.GetCvmForPublicId(InstanceId)
	if err != nil {
		return nil, err
	}
	if len(instanceList) != 1 {
		return nil, fmt.Errorf("%s query cvm info failed")
	}
	privateIp := *instanceList[0].PrivateIpAddresses[0]

	// ssh连接
	sshCli := ssh.New(privateIp, user, InstancePassWord)
	client, err := sshCli.GetClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var runPort string

	// 查询docker container数量
	ret, err := sshCli.Run(fmt.Sprintf(containerNum, userId))
	if err != nil {
		return nil, fmt.Errorf("exec cmd failed:ret:%s,err:%s", string(ret), err.Error())
	}

	existNum, err := strconv.Atoi(strings.Split(string(ret), "\n")[0])
	if err != nil {
		return nil, err
	}

	// 查询docker container是否运行
	if existNum > 0 {
		ret, _ = sshCli.Run(fmt.Sprintf(containerIsRun, userId))
		if string(ret) == "true\n" {
			return &pb.CreateWebConsoleReply{Message: runPort}, nil
		} else {
			startCmd := fmt.Sprintf(startContainer, userId)
			_, err = sshCli.Run(startCmd)
			if err != nil {
				return nil, err
			} else {
				return &pb.CreateWebConsoleReply{Message: runPort}, nil
			}
		}
	}

	// 获取端口
	// 创建docker container
	var runErr error
	if existNum == 0 {
		for i := 0; i <= 3; i++ {
			// 获取可用端口
			wettyPort, fileServerPort, err := GetRandomPort(sshCli)
			if err != nil {
				return nil, err
			}
			if wettyPort == 0 || fileServerPort == 0 {
				return nil, fmt.Errorf("get not use port err")
			}
			wettyPortStr := strconv.Itoa(wettyPort)
			fileServerPortStr := strconv.Itoa(fileServerPort)
			experimentCosPath := fmt.Sprintf("/users/%d/experiments", userId)
			cosOurl := fmt.Sprintf("http://cos.%s.myqcloud.com", cosService.Region)
			runCmd := fmt.Sprintf(newContainer, userId, wettyPortStr, fileServerPortStr, cosService.Bucket, experimentCosPath, cosOurl, cosService.Credential.SecretId, cosService.Credential.SecretKey, secret, userId, projectId, tefsUrl, userId)
			_, runErr = sshCli.Run(runCmd)
			if runErr == nil {
				runPort = fmt.Sprintf("%d:%d", wettyPort, fileServerPort)
				break
			}
		}
	}
	if runErr != nil {
		return nil, runErr
	}
	return &pb.CreateWebConsoleReply{Message: runPort}, nil
}
