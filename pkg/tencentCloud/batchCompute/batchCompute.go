package batchCompute

import (
	"TEFS-BE/pkg/log"
	"github.com/spf13/viper"
	batch "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/batch/v20170312"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"sync"
	"time"
)

var (
	batchClient     *batch.Client
	securityGroupId string
	projectId       int64

	// batch env API limit:提交任务
	submitJobLock             sync.Mutex
	submitJobLimit            int64 = 7
	submitJobLastRequestCount int64 = 0
	submitJobLastRequestTime  int64 = 0

	// batch env API limit:提交任务
	jobDetailsLock             sync.Mutex
	jobDetailsLimit            int64 = 3
	jobDetailsLastRequestCount int64 = 0
	jobDetailsLastRequestTime  int64 = 0

	// batch env API limit:终止任务
	terminateJobLock             sync.Mutex
	terminateJobLimit            int64 = 2
	terminateJobLastRequestCount int64 = 0
	terminateJobLastRequestTime  int64 = 0

	// batch env API limit:删除任务
	deleteJobLock             sync.Mutex
	deleteJobLimit            int64 = 5
	deleteJobLastRequestCount int64 = 0
	deleteJobLastRequestTime  int64 = 0
)

// batch compute 初始设置
func Setup() {
	secretId := viper.GetString("tencentCloud.secretId")
	secretKey := viper.GetString("tencentCloud.secretKey")
	region := viper.GetString("tencentCloud.region")

	// 腾讯云安全组
	securityGroupId = viper.GetString("tencentCloud.securityGroupId")
	projectId = viper.GetInt64("tencentCloud.projectId")

	// 批量计算客户端初始化设置
	credential := common.NewCredential(secretId, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "batch.tencentcloudapi.com"
	client, err := batch.NewClient(credential, region, cpf)
	if err != nil {
		log.Fatal(err.Error())
	}
	batchClient = client
}

// 获取batch compute客户端
func GetClient() *batch.Client {
	return batchClient
}

// 腾讯云接口请求限制
func RequestLimit(limit int64, count, lastTime *int64) {
	nowTime := time.Now().UnixNano()

	// 现在时间相对于上次请求时间，超过1s
	if (nowTime - *lastTime) > int64(time.Second) {
		*lastTime = nowTime
		*count = 1
		return
	}

	// 现在时间相对于上次请求时间，没超过1s
	// 在1s内没有超过请求限制数量
	if *count < limit {
		*lastTime = nowTime
		*count += 1
		return
	}

	// 接口被限制，等待
	time.Sleep(time.Second)
	nowTime = time.Now().UnixNano()
	*lastTime = nowTime
	*count = 1
}

// batch compute提交任务
func SubmitJob(command, name, description, cosPath, envId, zone string,
	timeout *uint64, projectId int64) (jobId *string, err error) {

	deliveryForm := "LOCAL"
	var taskInstanceNum uint64 = 1 // 任务运行实例个数

	application := &batch.Application{
		Command:      &command,      // 计算执行指令
		DeliveryForm: &deliveryForm, // 交付方式:本地
	}
	// RedirectInfo 任务运行时输出重定向路径
	redirectInfo := &batch.RedirectInfo{
		StdoutRedirectPath: &cosPath, // 正常输出
		StderrRedirectPath: &cosPath, // 错误输出
	}
	task := &batch.Task{
		TaskName:        &name,
		TaskInstanceNum: &taskInstanceNum,
		EnvId:           &envId,
		Application:     application,
		RedirectInfo:    redirectInfo,
		Timeout:         timeout,
	}

	// 创建job
	tasks := []*batch.Task{task}
	var priority uint64 = 1
	job := &batch.Job{
		Tasks:          tasks,
		JobName:        &name,
		JobDescription: &description,
		Priority:       &priority,
	}

	// placement 位置信息
	placement := &batch.Placement{
		Zone:      &zone,
		ProjectId: &projectId,
	}

	// 构造请求,提交job
	request := batch.NewSubmitJobRequest()
	request.Placement = placement
	request.Job = job

	client := GetClient()
	submitJobLock.Lock()
	defer submitJobLock.Unlock()
	RequestLimit(submitJobLimit, &submitJobLastRequestCount, &submitJobLastRequestTime)
	response, err := client.SubmitJob(request)
	if err != nil {
		return nil, err
	}
	return response.Response.JobId, nil
}

// 获取任务详情
func GetJobDetails(jid string) (*batch.DescribeJobResponse, error) {
	request := batch.NewDescribeJobRequest()
	request.JobId = common.StringPtr(jid)
	client := GetClient()
	jobDetailsLock.Lock()
	defer jobDetailsLock.Unlock()
	RequestLimit(jobDetailsLimit, &jobDetailsLastRequestCount, &jobDetailsLastRequestTime)
	return client.DescribeJob(request)
}

// 终止计算任务
func TerminateJob(jid string) error {
	request := batch.NewTerminateJobRequest()
	request.JobId = common.StringPtr(jid)
	client := GetClient()
	terminateJobLock.Lock()
	defer terminateJobLock.Unlock()
	RequestLimit(terminateJobLimit, &terminateJobLastRequestCount, &terminateJobLastRequestTime)
	_, err := client.TerminateJob(request)
	return err
}

// 删除计算任务, 任务状态处于SUCCEED 或 FAILED才能删除成功
func DeleteJob(jid string) error {
	request := batch.NewDeleteJobRequest()
	request.JobId = common.StringPtr(jid)
	client := GetClient()
	deleteJobLock.Lock()
	defer deleteJobLock.Unlock()
	RequestLimit(deleteJobLimit, &deleteJobLastRequestCount, &deleteJobLastRequestTime)
	_, err := client.DeleteJob(request)
	return err
}
