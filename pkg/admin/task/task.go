package task

import (
	"TEFS-BE/pkg/admin/compute"
	"TEFS-BE/pkg/admin/model"
	"TEFS-BE/pkg/database"
	laboratoryCli "TEFS-BE/pkg/laboratory/client"
	pb "TEFS-BE/pkg/laboratory/proto"
	"context"
	"encoding/json"
	"fmt"
	"github.com/RichardKnop/machinery/v1/backends/result"
	"github.com/RichardKnop/machinery/v1/log"
	mchTasks "github.com/RichardKnop/machinery/v1/tasks"
	"github.com/jinzhu/gorm"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc/status"
)

var (
	// 监控实验环境
	MonitoringExperimentEnvFunc     = "monitoringExperimentEnv"
	MonitoringExperimentComputeFunc = "monitoringExperimentCompute"
	DeleteExperimentFunc            = "deleteExperimentFunc"

	// 腾讯云错误信息前缀
	jobIdNotFoundErrPrefix = "[TencentCloudSDKError] Code=ResourceNotFound.Job"
	jobIdInvalidErrPrefix  = "[TencentCloudSDKError] Code=InvalidParameter.JobIdMalformed"
	envIdNotFoundErrPrefix = "[TencentCloudSDKError] Code=ResourceNotFound.ComputeEnv"
	envIdInvalidErrPrefix  = "[TencentCloudSDKError] Code=InvalidParameter.EnvIdMalformed"

	// 删除实验的最大重试次数和每次重试的延时时间
	deleteExperimentRetryMaxCount int64         = 6
	deleteExperimentETA           time.Duration = 60 * 10

	// task延时执行时间，单位秒
	taskEtaTime time.Duration = 30
)

// 发送实验任务
func SendExperimentTask(ctx context.Context, retryCount int,
	name string, experimentId int64, eta *time.Time, args ...mchTasks.Arg) (*result.AsyncResult, error) {

	task, err := mchTasks.NewSignature(name, args)
	if err != nil {
		fmt.Println(err)
	}
	task.RetryCount = retryCount
	task.ETA = eta
	task.UUID = fmt.Sprintf("experiment_%d", experimentId)
	task.Headers = make(map[string]interface{})
	task.RoutingKey = queue

	return taskCenter.SendTaskWithContext(ctx, task)
}

// 删除实验(在腾讯云batch上，终止job,删除job,删除env)
func deleteBatchCompute(client *pb.LaboratoryClient, envId, jid string, experimentId int64) (success bool) {
	var delJobSuccess, delEnvSuccess bool

	if len(jid) > 0 {
		// 查询任务,如果任务处于运行状态就终止任务
		response, err := laboratoryCli.QueryExperiment(*client, jid)
		if err != nil {
			log.ERROR.Println(fmt.Sprintf("experiment %d, query batch job:%s, failed, err:%s",
				experimentId, jid, err.Error()))
		} else {
			jobStatus := *response.Response.JobState
			if jobStatus == "RUNNING" {
				err := laboratoryCli.TerminateExperiment(*client, jid)
				if err != nil {
					log.ERROR.Println(fmt.Sprintf("experiment %d, terminate batch job:%s, failed, err:%s",
						experimentId, jid, err.Error()))
				} else {
					log.INFO.Println(fmt.Sprintf("experiment %d, terminate batch job %s, success.",
						experimentId, jid))
				}
			}
		}

		// 删除任务
		err = laboratoryCli.DeleteExperiment(*client, jid)
		if err != nil {
			rpcErr, ok := status.FromError(err)
			if ok {
				if strings.HasPrefix(rpcErr.Message(), jobIdNotFoundErrPrefix) || strings.HasPrefix(rpcErr.Message(),
					jobIdInvalidErrPrefix) {
					delJobSuccess = true
				}
			}
			log.ERROR.Println(fmt.Sprintf("experiment %d, delete batch job %s failed, err:%s",
				experimentId, jid, err.Error()))
		} else {
			log.INFO.Println(fmt.Sprintf("experiment %d, delete batch job %s success.", experimentId, jid))
			delJobSuccess = true
		}
	} else {
		delJobSuccess = true
	}

	// 删除计算环境
	err := laboratoryCli.DeleteExperimentEnv(*client, envId)
	if err != nil {
		rpcErr, ok := status.FromError(err)
		if ok {
			if strings.HasPrefix(rpcErr.Message(), envIdNotFoundErrPrefix) || strings.HasPrefix(rpcErr.Message(),
				envIdInvalidErrPrefix) {
				delEnvSuccess = true
			}
		}
		log.ERROR.Println(fmt.Sprintf("experiment %d, delete batch env %s failed, err:%s",
			experimentId, envId, err.Error()))
	} else {
		delEnvSuccess = true
		log.INFO.Println(fmt.Sprintf("experiment %d, delete batch env %s success.", experimentId, envId))
	}
	if delJobSuccess && delEnvSuccess {
		return true
	}
	return
}

// 加锁获取实验数据库记录
func getExperimentForUpdate(experimentId int64) (tx *gorm.DB, experiment *model.Experiment, success bool) {
	db := database.GetDb()
	tx = db.Begin()
	experiment = &model.Experiment{}
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(experiment, experimentId).Error; err != nil {
		log.ERROR.Println("experiment %s, get db record for update failed, err:%s", experiment.Id, err.Error())
		return tx, experiment, false
	}
	return tx, experiment, true
}

// 更新实验数据库记录
func updateExperiment(tx *gorm.DB, experiment *model.Experiment, ups map[string]interface{}) (success bool) {
	upsJosn, _ := json.Marshal(ups)
	log.INFO.Println(fmt.Sprintf("experiment %d, update db record, ups:%s", experiment.Id, string(upsJosn)))
	if err := tx.Model(experiment).Update(ups).Error; err != nil {
		log.ERROR.Println("experiment %s, update db record failed,err:%s", experiment.Id, err.Error())
		return
	}
	log.INFO.Println(fmt.Sprintf("experiment %d, update db record success", experiment.Id))
	return true
}

// 获取用户实验室后台grpc Client
func getLaboratoryClient(experimentId int64, laboratoryAddress string) (client *pb.LaboratoryClient, success bool) {
	LaboratoryClient, err := laboratoryCli.GetClient(laboratoryAddress)
	if err != nil {
		log.ERROR.Println(fmt.Sprintf("experiment %d, get laboratory clinet err:%s", experimentId, err.Error()))
		return nil, false
	}
	return &LaboratoryClient, true
}

// 验证实验是否计算成功
func verifyExperimentIsSuccess(experiment *model.Experiment) (bool, error) {
	tmpSavePath := filepath.Join(tmpOszicarDir, fmt.Sprintf("%s_%d", compute.OszicarName, experiment.Id))
	if err := compute.GetExperimentFileForCos(experiment, tmpSavePath, compute.OszicarName); err != nil {
		return false, err
	}
	defer os.Remove(tmpSavePath)
	lastLine, err := compute.GetLastLine(tmpSavePath)
	if err != nil {
		return false, nil
	}
	return compute.Check(lastLine, compute.LastLineCorrectRe), nil
}

// 获取实验OSZICAR信息（迭代次数，能量）
func getOszicarInfo(experiment *model.Experiment) (oszicarJson string, err error) {
	tmpSavePath := filepath.Join(tmpOszicarDir, fmt.Sprintf("%s_%d", compute.ExpDict, experiment.Id))
	if err = compute.GetExperimentFileForCos(experiment, tmpSavePath, compute.ExpDict); err != nil {
		return "", err
	}
	defer os.Remove(tmpSavePath)
	o := compute.OSZICAR{
		Path: tmpSavePath,
	}
	exp, err := o.ReadExpDictJson()
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(exp)
	oszicarJson = string(b)
	return
}

// 任务重新发送
func reSendExperiment(experimentId int64, taskFunc string, eta *time.Time, taskArgs ...mchTasks.Arg) bool {
	_, err := SendExperimentTask(context.Background(), 0, taskFunc, experimentId, eta, taskArgs...)
	if err != nil {
		log.ERROR.Println(fmt.Sprintf("experiment %d, task resend to %s failed,err:%s",
			experimentId, taskFunc, err.Error()))
		return false
	}
	log.INFO.Println(fmt.Sprintf("experiment %d, task resend to %s success", experimentId, taskFunc))
	return true
}

// 发送任务到删除实验batch队列
func sendToDeleteExperiment(experiment *model.Experiment) bool {
	taskExperimentId := mchTasks.Arg{
		Name:  "experimentId",
		Type:  "int64",
		Value: experiment.Id,
	}
	taskEnvId := mchTasks.Arg{
		Name:  "envId",
		Type:  "string",
		Value: experiment.BatchComputeEnvId,
	}
	taskJid := mchTasks.Arg{
		Name:  "jid",
		Type:  "string",
		Value: experiment.BatchJid,
	}
	taskLaboratoryAddress := mchTasks.Arg{
		Name:  "laboratoryAddress",
		Type:  "string",
		Value: experiment.LaboratoryAddress,
	}
	taskRetryCount := mchTasks.Arg{
		Name:  "retryCount",
		Type:  "int64",
		Value: 1,
	}
	//eta := time.Now().UTC().Add(time.Second * deleteExperimentETA) // 延时执行
	eta := time.Now().UTC().Add(time.Second * 0) // 延时执行
	_, err := SendExperimentTask(context.Background(), 0, DeleteExperimentFunc, experiment.Id, &eta,
		taskEnvId, taskJid, taskLaboratoryAddress, taskExperimentId, taskRetryCount)
	if err != nil {
		logMsg := fmt.Sprintf("experiment %d, send to DeleteExperimentFunc failed, jid:%s, envId:%s, err:%s",
			experiment.Id, experiment.BatchJid, experiment.BatchComputeEnvId, err.Error())
		log.ERROR.Println(logMsg)
		return false
	} else {
		logMsg := fmt.Sprintf("experiment %d, send to DeleteExperimentFunc success, jid:%s, envId:%s",
			experiment.Id, experiment.BatchJid, experiment.BatchComputeEnvId)
		log.INFO.Println(logMsg)
		return true
	}
}

// task 监控实验计算
func MonitoringExperimentCompute(experimentId int64, batchJid string) error {
	experimentJobIsDone := false // 实验计算任务是否完成
	defer func() {
		// 获取用户服务client失败，或者任务还在计算中，再次打入监控任务计算队列
		if !experimentJobIsDone {
			// 计算环境还未创建完成
			taskArgExperimentId := mchTasks.Arg{
				Name:  "experimentId",
				Type:  "int64",
				Value: experimentId,
			}
			taskArgBatchJid := mchTasks.Arg{
				Name:  "batchJid",
				Type:  "string",
				Value: batchJid,
			}
			eta := time.Now().UTC().Add(time.Second * taskEtaTime) // 延时30s执行
			reSendExperiment(experimentId, MonitoringExperimentComputeFunc, &eta, taskArgExperimentId, taskArgBatchJid)
		}
	}()

	// 数据库查询实验记录
	tx, experiment, success := getExperimentForUpdate(experimentId)
	if !success {
		return nil
	}
	defer tx.Commit()

	// 获取client
	client, success := getLaboratoryClient(experiment.Id, experiment.LaboratoryAddress)
	if !success {
		return nil
	}

	// 更新数据库记录
	ups := make(map[string]interface{})
	defer func() {
		if len(ups) > 0 {
			updateExperiment(tx, experiment, ups)
			var experimentSuccess bool
			_, ok := ups["status"]
			if ok {
				switch ups["status"].(int) {
				case 5:
					experimentSuccess = false
				case 6:
					experimentSuccess = true
				default:
					return
				}
				compute.ExperimentNotify(experimentSuccess, experiment)
			}
		}
	}()

	// 删除实验的batch
	isDeleteExperiment := false
	defer func() {
		if isDeleteExperiment {
			success := deleteBatchCompute(client, experiment.BatchComputeEnvId, experiment.BatchJid, experiment.Id)
			if !success {
				sendToDeleteExperiment(experiment)
			}
		}
	}()

	// 任务被终止
	if experiment.Status == 4 {
		// 任务被终止
		ups["status"] = 5
		ups["done_at"] = time.Now().Unix()
		ups["errMsg"] = "任务被终止"
		isDeleteExperiment = true  // 删除实验batch env和job
		experimentJobIsDone = true // 实验结束
		return nil
	}

	response, err := laboratoryCli.QueryExperiment(*client, batchJid)
	if err != nil {
		rpcErr, ok := status.FromError(err)
		if ok {
			// batch jid 在腾讯云batch 查询不到。可能被用户手动删除等。任务结束，状态为失败，删除计算环境
			if strings.HasPrefix(rpcErr.Message(), jobIdNotFoundErrPrefix) || strings.HasPrefix(rpcErr.Message(),
				jobIdInvalidErrPrefix) {
				experimentJobIsDone = true // 实验结束
				isDeleteExperiment = true  // 删除batch env 和 job
				ups["status"] = 5
				ups["done_at"] = time.Now().Unix()
				ups["err_msg"] = fmt.Sprintf("计算任务(%s)不存在", experiment.BatchJid)
			}
		}
		logMsg := fmt.Sprintf("experiment %d, QueryExperiment batch job status faield,err:%s",
			experiment.Id, err.Error())
		log.ERROR.Println(logMsg)
		return nil
	}

	jobStatus := *response.Response.JobState
	// 任务完成
	if jobStatus == "SUCCEED" {
		experimentJobIsDone = true // 实验结束
		isDeleteExperiment = true  // 删除batch env 和 job
		ups["status"] = 6
		ups["done_at"] = time.Now().Unix()
		logMsg := fmt.Sprintf("experiment %d, batch job status: SUCCEED", experiment.Id)
		log.INFO.Println(logMsg)
		oszicarJson, err := getOszicarInfo(experiment)
		if err != nil {
			log.ERROR.Println(err.Error())
		} else {
			log.INFO.Println(oszicarJson)
			ups["oszicar_json"] = oszicarJson
		}
		experimentSuccess, err := verifyExperimentIsSuccess(experiment)
		if err != nil {
			log.ERROR.Println(err)
			return nil
		}
		if !experimentSuccess {
			ups["err_msg"] = "计算错误，请查看日志"
			ups["status"] = 5
		}
		return nil
	}

	if jobStatus == "FAILED" {
		experimentJobIsDone = true // 实验结束
		isDeleteExperiment = true  // 删除batch env 和 job
		ups["status"] = 5
		ups["done_at"] = time.Now().Unix()
		ups["err_msg"] = "作业提交指令有误"
		logMsg := fmt.Sprintf("experiment %d, batch job status: FAILED", experiment.Id)
		log.INFO.Println(logMsg)
		return nil
	}

	// job还在计算中
	logMsg := fmt.Sprintf("experiment %d, batch job status: %s", experiment.Id, *response.Response.JobState)
	log.INFO.Println(logMsg)

	oszicarJson, err := getOszicarInfo(experiment)
	if err != nil {
		log.ERROR.Println(err.Error())
	} else {
		log.INFO.Println(oszicarJson)
		ups["oszicar_json"] = oszicarJson
	}
	return nil
}

const (
	vasp     = "vasp"
	gpu_vasp = "gpu_vasp"
)

// task 监控实验计算环境的创建
func MonitoringExperimentEnv(experimentId int64) error {
	// 创建计算环境是否结束。
	// 1 创建成功视为结束。
	// 2 创建失败并删除计算环境视为结束。
	// 3 任务被终止并删除计算环境视为结束。
	experimentEnvIsDone := false
	defer func() {
		if !experimentEnvIsDone {
			taskArg := mchTasks.Arg{
				Name:  "experimentId",
				Type:  "int64",
				Value: experimentId,
			}
			eta := time.Now().UTC().Add(time.Second * taskEtaTime) // 延时30s执行
			reSendExperiment(experimentId, MonitoringExperimentEnvFunc, &eta, taskArg)
		}
	}()

	// 数据库获取实验记录
	tx, experiment, success := getExperimentForUpdate(experimentId)
	if !success {
		return nil
	}
	defer tx.Commit()

	// 获取用户项目的实验室的grpc客户端
	client, success := getLaboratoryClient(experiment.Id, experiment.LaboratoryAddress)
	if !success {
		return nil
	}

	// 删除batch计算环境和job
	isDeleteBatch := false
	defer func() {
		if isDeleteBatch {
			success := deleteBatchCompute(client, experiment.BatchComputeEnvId, experiment.BatchJid, experiment.Id)
			if !success {
				sendToDeleteExperiment(experiment)
			}
		}
	}()

	// 更新数据库
	ups := make(map[string]interface{})
	defer func() {
		if len(ups) > 0 {
			updateExperiment(tx, experiment, ups)
			// 实验失败，通知
			if ups["status"].(int) == 5 {
				compute.ExperimentNotify(false, experiment)
			}
		}
	}()

	// 实验被终止
	if experiment.Status == 4 {
		ups["status"] = 5
		ups["err_msg"] = "任务被终止"
		ups["done_at"] = time.Now().Unix()
		isDeleteBatch = true       // 删除batch env
		experimentEnvIsDone = true // 实验结束
		return nil
	}

	// 实验状态 不等于2(创建环境中),丢弃
	if experiment.Status != 2 {
		experimentEnvIsDone = true
		log.WARNING.Println(fmt.Sprintf("experiment %d, db record status:%d, discard task",
			experimentId, experiment.Status))
		return nil
	}

	// 查询环境,获取各种状态的节点数量
	response, err := laboratoryCli.QueryExperimentEnv(*client, experiment.BatchComputeEnvId)
	if err != nil {
		rpcErr, ok := status.FromError(err)
		if ok {
			// bathc envid 在腾讯云查询不到(可能被用户手动删除等情况)，结束任务，更新数据库
			if strings.HasPrefix(rpcErr.Message(), envIdNotFoundErrPrefix) || strings.HasPrefix(rpcErr.Message(),
				envIdInvalidErrPrefix) {
				experimentEnvIsDone = true // 不在重发任务
				isDeleteBatch = true
				ups["status"] = 5 // 失败
				ups["done_at"] = time.Now().Unix()
				ups["errMsg"] = fmt.Sprintf("计算环境(%s)不存在", experiment.BatchComputeEnvId)
			}
		}
		// 其他错误，会重发任务，延时30S处理
		log.ERROR.Println(fmt.Sprintf("experiment %d, query batch env status failed, err:%s",
			experiment.Id, err.Error()))
		return nil
	}
	runningCount := *response.Response.ComputeNodeMetrics.RunningCount               // 运行中计算节点数量
	creationFailedCount := *response.Response.ComputeNodeMetrics.CreationFailedCount // 创建失败节点数量
	abnormalCount := *response.Response.ComputeNodeMetrics.AbnormalCount             // 创建完成，但是存在异常的节点数量

	// 创建环境异常，删除计算环境
	if creationFailedCount > 0 || abnormalCount > 0 {
		isDeleteBatch = true       // 需要删除计算环境
		experimentEnvIsDone = true // 任务结束
		ups["status"] = 5          // 失败
		ups["done_at"] = time.Now().Unix()
		ups["errMsg"] = "计算环境创建节点失败"
		log.ERROR.Println(
			fmt.Sprintf("experiment %d, batch create env faield: creationFailedCount=%d, abnormalCount=%d",
				experiment.Id, creationFailedCount, abnormalCount))
		return nil
	}

	// 创建情况
	if int64(runningCount) < experiment.ComputeNodeNum {
		// 创建中
		log.INFO.Println(fmt.Sprintf("experiment %d, batch env creating...", experiment.Id))
		return nil
	}

	// 计算环境创建完成，计算前准备，提交计算任务
	// 计算节点ip和cup数量
	nodeIps := []string{}
	cpuCount := make(map[string]int)
	var batchJid, errMsg, experimentType string
	if strings.HasPrefix(experiment.ExperimentType, "vasp") {
		experimentType = vasp
	} else if strings.HasPrefix(experiment.ExperimentType, "gpu_vasp") {
		experimentType = gpu_vasp
	}
	switch experimentType {
	case vasp:
		for _, v := range response.Response.ComputeNodeSet {
			nodeIps = append(nodeIps, *v.PrivateIpAddresses[0])
			cpuCount[*v.PrivateIpAddresses[0]] = int(*v.Cpu)
		}
		batchJid, errMsg = compute.VaspSubmit(vasp, nodeIps, cpuCount, experiment, tmpExperimentBaseDir, client)
	case gpu_vasp:
		for _, v := range response.Response.ComputeNodeSet {
			nodeIps = append(nodeIps, *v.PrivateIpAddresses[0])
			// gpu计算固定机型 GN10X.2XLARGE40，每台GPU数量为1
			cpuCount[*v.PrivateIpAddresses[0]] = 1
		}
		batchJid, errMsg = compute.VaspSubmit(gpu_vasp, nodeIps, cpuCount, experiment, tmpExperimentBaseDir, client)
	default:
		errMsg = "未知任务类型"
		isDeleteBatch = true
	}
	if len(errMsg) > 0 {
		ups["status"] = 5
		ups["err_msg"] = errMsg
		ups["done_at"] = time.Now().Unix()
		isDeleteBatch = true
		experimentEnvIsDone = true
		return nil
	}

	experiment.BatchJid = batchJid

	// 计算任务监控task发送
	taskArgExperimentId := mchTasks.Arg{
		Name:  "experimentId",
		Type:  "int64",
		Value: experimentId,
	}
	taskArgBatchJid := mchTasks.Arg{
		Name:  "batchJid",
		Type:  "string",
		Value: batchJid,
	}
	experimentEnvIsDone = true
	eta := time.Now().UTC().Add(time.Second * taskEtaTime) // 延迟30s执行
	_, err = SendExperimentTask(context.Background(), 0, MonitoringExperimentComputeFunc,
		experiment.Id, &eta, taskArgExperimentId, taskArgBatchJid)
	if err != nil {
		log.ERROR.Println(fmt.Sprintf("experiment %d, send to MonitoringExperimentCompute failed,err:%s",
			experiment.Id, err.Error()))
		ups["batch_jid"] = batchJid
		ups["status"] = 5
		ups["err_msg"] = "发送到计算监控队列失败"
		ups["done_at"] = time.Now().Unix()
		isDeleteBatch = true
		return nil
	}
	log.INFO.Println(fmt.Sprintf("experiment %d, task send to monitoringExperimentCompute success",
		experiment.Id))

	ups["status"] = 3 // 任务计算中
	ups["batch_jid"] = batchJid
	return nil
}

// task 删除实验的batchCompute (job, env)
func DeleteExperiment(envId, jid, laboratoryAddress string, experimentId, retryCount int64) error {
	reSend := false
	defer func() {
		retryCount += 1
		if reSend {
			if retryCount < deleteExperimentRetryMaxCount {
				taskExperimentId := mchTasks.Arg{
					Name:  "experimentId",
					Type:  "int64",
					Value: experimentId,
				}
				taskEnvId := mchTasks.Arg{
					Name:  "envId",
					Type:  "string",
					Value: envId,
				}
				taskJid := mchTasks.Arg{
					Name:  "jid",
					Type:  "string",
					Value: jid,
				}
				taskLaboratoryAddress := mchTasks.Arg{
					Name:  "laboratoryAddress",
					Type:  "string",
					Value: laboratoryAddress,
				}
				taskRetryCount := mchTasks.Arg{
					Name:  "retryCount",
					Type:  "int64",
					Value: retryCount,
				}
				eta := time.Now().UTC().Add(time.Second * deleteExperimentETA) // 延时执行
				log.WARNING.Println(fmt.Sprintf("experiment %d, delete batch compute count:%d",
					experimentId, retryCount-1))
				reSendExperiment(experimentId, DeleteExperimentFunc, &eta, taskEnvId, taskJid,
					taskLaboratoryAddress, taskExperimentId, taskRetryCount)
			} else {
				log.ERROR.Println(fmt.Sprintf("experiment %d, delete batch compute failed, retry exceeded",
					experimentId))
				// 删除失败发送邮件,邮件重试3次
				for i := 0; i < 3; i++ {
					if !compute.DelEnvFailedEmailNotify(experimentId, envId, jid, laboratoryAddress) {
						time.Sleep(time.Duration(30) * time.Second)
						log.ERROR.Println(fmt.Sprintf("experiment %d, delete env failed send email notify failed, count:%d", experimentId, i))
						continue
					} else {
						log.INFO.Println(fmt.Sprintf("experiment %d, delete env failed send email notify success", experimentId))
						break
					}
				}
			}
		}
	}()

	client, success := getLaboratoryClient(experimentId, laboratoryAddress)
	if !success {
		reSend = true
		return nil
	}

	success = deleteBatchCompute(client, envId, jid, experimentId)
	if !success {
		reSend = true
	}
	return nil
}
