package compute

import (
	"TEFS-BE/pkg/admin/model"
	laboratoryCli "TEFS-BE/pkg/laboratory/client"
	pb "TEFS-BE/pkg/laboratory/proto"
	"TEFS-BE/pkg/tencentCloud/batchCompute"
	"encoding/base64"
	"fmt"
	"github.com/RichardKnop/machinery/v1/log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	// 远程计算服务的hosts文件
	hosts      = "hosts"
	hostfile   = "hostfile"
	syncHostSh = "syncHost.sh"
	mountSh    = "mount.sh"

	vaspJobNamePrefix         = "tefs-vasp"
	vaspJobTime        uint64 = 60 * 60 * 24 * 7 // 一周
	vaspJobDescription        = "tefs vasp计算任务，请勿删除。"

	settingService = model.SettingService{}
)

// 生产hosts和hostfile文件,以及同步节点host文件脚本
func GenerateHostFile(vaspType string, privateIps []string, cpuCount map[string]int,
	tmpDir, cosSaveBasePath string) (map[string]string, error) {
	// 临时保存目录
	_, err := os.Stat(tmpDir)
	if err == nil {
		// 文件加存在，删除
		err := os.RemoveAll(tmpDir)
		if err != nil {
			return nil, err
		}
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	// 创建文件夹
	err = os.Mkdir(tmpDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	// 创建文件并打开
	hostsPath := filepath.Join(tmpDir, hosts)
	hostfilePath := filepath.Join(tmpDir, hostfile)
	syncHostShPath := filepath.Join(tmpDir, syncHostSh)
	mountPath := filepath.Join(tmpDir, mountSh)
	fhosts, err := os.OpenFile(hostsPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer fhosts.Close()
	fhostfile, err := os.OpenFile(hostfilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer fhostfile.Close()
	fsyncHostSh, err := os.OpenFile(syncHostShPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer fsyncHostSh.Close()
	fMount, err := os.OpenFile(mountPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer fMount.Close()

	// 写入文件内容
	var hostsContent, hostfileContent, syncHostShContent, mountShCount string
	for i, ip := range privateIps {
		hostsContent += ip + " " + "node" + strconv.Itoa(i) + "\n"
		if vaspType == "gpu_vasp" {
			hostfileContent += "node" + strconv.Itoa(i) + "\n"
		} else {
			hostfileContent += "node" + strconv.Itoa(i) + " slots=" + strconv.Itoa(cpuCount[ip]) + "\n"
		}
		syncHostShContent += `ssh -q -o StrictHostKeyChecking=no ` + ip + ` "cat ` + batchCompute.RemoteWorkDir + `hosts >> /etc/hosts"` + "\n"
		if i > 0 {
			mountShCount += fmt.Sprintf("ssh -q -o StrictHostKeyChecking=no %s mount %s:%s %s\n", ip,
				privateIps[0], batchCompute.MountWorkDir, batchCompute.MountWorkDir)
		}
	}
	_, err = fhosts.WriteString(hostsContent)
	if err != nil {
		return nil, err
	}
	_, err = fhostfile.WriteString(hostfileContent)
	if err != nil {
		return nil, err
	}
	_, err = fsyncHostSh.WriteString(syncHostShContent)
	if err != nil {
		return nil, err
	}
	_, err = fMount.WriteString(mountShCount)
	if err != nil {
		return nil, err
	}

	uploadFiles := make(map[string]string)
	uploadFiles[cosSaveBasePath+hosts] = hostsPath
	uploadFiles[cosSaveBasePath+hostfile] = hostfilePath
	uploadFiles[cosSaveBasePath+syncHostSh] = syncHostShPath
	uploadFiles[cosSaveBasePath+mountSh] = mountPath

	return uploadFiles, nil
}

// 上传host文件到用户cos
func UploadHostFileToCos(client pb.LaboratoryClient, files map[string]string, experimentCosPath string) error {
	// 获取用户服务cos临时秘钥
	cosTmpSecret, cosBaseUrl, err := laboratoryCli.GetCosUploadTmpSecret(client, experimentCosPath)
	if err != nil {
		return err
	}
	return laboratoryCli.UploadCos(cosTmpSecret, *cosBaseUrl, files)
}

// 获取实验执行指令
func GetExperimentCmd(nodeIp string, nodeNum int64, cpuCount int64, experimentType string, nodeCoreInfo map[string]int) (string, error) {
	// vasp计算指令
	if strings.HasPrefix(experimentType, "vasp") {
		b64Cmd := settingService.GetVaspB64Cmd()
		runCmdByte, err := base64.StdEncoding.DecodeString(b64Cmd)
		if err != nil {
			return "", err
		}
		vaspExecPath := settingService.GetVaspExecPath(experimentType)
		if len(vaspExecPath) == 0 {
			return "", fmt.Errorf(fmt.Sprintf("db not found %s exec path", experimentType))
		}
		runCmd := fmt.Sprintf(string(runCmdByte), nodeIp, nodeNum*cpuCount, vaspExecPath)
		return runCmd, nil
	} else if strings.HasPrefix(experimentType, "gpu_vasp") {
		b64Cmd := settingService.GetVaspGPUB64Cmd()
		runCmdByte, err := base64.StdEncoding.DecodeString(b64Cmd)
		if err != nil {
			return "", err
		}
		vaspExecPath := settingService.GetVaspGPUExecBasePath()
		if len(vaspExecPath) == 0 {
			return "", fmt.Errorf(fmt.Sprintf("db not found %s exec path", experimentType))
		}

		tmp := strings.Split(experimentType, "gpu_")
		if len(tmp) > 1 {
			vaspExecPath += tmp[1]
		} else {
			return "", fmt.Errorf("experiment type error")
		}

		var gpuNodes []string
		for k, v := range nodeCoreInfo {
			for i := 0; i < v; i++ {
				gpuNodes = append(gpuNodes, k)
			}
		}
		gpuCmdParamH := strings.Join(gpuNodes, ",")
		runCmd := fmt.Sprintf(string(runCmdByte), nodeIp, gpuCmdParamH, nodeNum*cpuCount, vaspExecPath)
		return runCmd, nil
	}
	return "", fmt.Errorf(fmt.Sprintf("unknown experiment type:%s", experimentType))
}

// vasp提交之前处理
// 返回计算指令（base64编码）
func VaspSubmit(vaspType string, nodeIps []string, cpuCount map[string]int, experiment *model.Experiment,
	tmpSaveDir string, client *pb.LaboratoryClient) (batchJid, errMsg string) {
	cosSaveBasePath := fmt.Sprintf("/users/%d/experiments/%d/", experiment.UserId, experiment.Id)
	tmpDir := fmt.Sprintf("%s/%d", tmpSaveDir, experiment.Id)

	// 1.生成hosts文件，保存到临时文件目录
	uploadFiles, err := GenerateHostFile(vaspType, nodeIps, cpuCount, tmpDir, cosSaveBasePath)
	if err != nil {
		log.ERROR.Println(fmt.Sprintf("experiment %d, GenerateHostFile failed, err:%s", experiment.Id, err.Error()))
		return "", "生成host文件失败"
	}
	defer os.RemoveAll(tmpDir)

	// 2.上传到用户cos
	if err := UploadHostFileToCos(*client, uploadFiles, cosSaveBasePath+"*"); err != nil {
		log.ERROR.Println(fmt.Sprintf("experiment %d, UploadHostFileToCos failed, err:%s", experiment.Id, err.Error()))
		return "", "上传host文件到cos失败"
	}

	// 3.获取计算指令
	var minCupCount int64
	for _, v := range cpuCount {
		if minCupCount == 0 {
			minCupCount = int64(v)
		} else {
			if int64(v) < minCupCount {
				minCupCount = int64(v)
			}
		}
	}

	nodeCoreInfo := make(map[string]int)
	for i, nodeIp := range nodeIps {
		nodeCoreInfo[fmt.Sprintf("node%d", i)] = cpuCount[nodeIp]
	}
	cmd, err := GetExperimentCmd(nodeIps[0], experiment.ComputeNodeNum, minCupCount, experiment.ExperimentType, nodeCoreInfo)
	if err != nil {
		log.ERROR.Println(fmt.Sprintf("experiment %d, GetExperimentCmd failed,err:%s", experiment.Id, err.Error()))
		return "", "获取计算指令失败"
	}
	runCmd := base64.StdEncoding.EncodeToString([]byte(cmd))
	log.ERROR.Println(cmd)

	// 4.提交任务
	jobName := fmt.Sprintf("%s-%d", vaspJobNamePrefix, experiment.Id)
	jid, err := laboratoryCli.SubmitExperiment(*client, jobName, vaspJobDescription, runCmd, experiment.CosBasePath,
		experiment.BatchComputeEnvId, experiment.Zone, vaspJobTime)
	if err != nil {
		log.ERROR.Println(fmt.Sprintf("experiment %d, SubmitExperiment failed,err:%s", experiment.Id, err.Error()))
		return "", "提交任务到batch失败"
	}
	return *jid, ""
}
