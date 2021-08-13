package batchCompute

import (
	batch "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/batch/v20170312"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"sync"
)

var (
	envType = "MANAGED"
	envName = "tefs-vasp-env"

	// 宽带参数
	internetMaxBandwidthOut int64 = 10
	publicIpAssigned              = true

	// 远程节点工作目录
	//RemoteWorkDir = "/root/workspace/"
	MountWorkDir = "/root/computing/"
	RemoteWorkDir = "/root/workspace/"

	// batch env API limit:create
	createEnvLock             sync.Mutex
	createEnvLimit            int64 = 2
	createEnvLastRequestCount int64 = 0
	createEnvLastRequestTime  int64 = 0

	// batch env API limit:describe
	describeEnvLock             sync.Mutex
	describeEnvLimit            int64 = 2
	describeEnvLastRequestCount int64 = 0
	describeEnvLastRequestTime  int64 = 0

	// batch env API limit:describe list
	describeEnvsLock             sync.Mutex
	describeEnvsLimit            int64 = 2
	describeEnvsLastRequestCount int64 = 0
	describeEnvsLastRequestTime  int64 = 0

	// batch env API limit:delete
	deleteEnvLock             sync.Mutex
	deleteEnvLimit            int64 = 2
	deleteEnvLastRequestCount int64 = 0
	deleteEnvLastRequestTime  int64 = 0
)

// 创建计算环境
// 给定计算节点数量，最少1。
// 给定cos映射路径（输入源文件）
func CreateEnv(nodeNum int64, cvmImageId, cvmPassWord, cosPath, zone, instanceType, diskType string, diskSize int64) (*string, error) {
	request := batch.NewCreateComputeEnvRequest()
	inputMapping := &batch.InputMapping{
		SourcePath:      &cosPath,
		DestinationPath: &RemoteWorkDir,
	}
	request.ComputeEnv = &batch.NamedComputeEnv{
		EnvName:                 &envName,
		DesiredComputeNodeCount: &nodeNum,
		EnvType:                 &envType,
		InputMappings:           []*batch.InputMapping{inputMapping},
		EnvData: &batch.EnvData{
			InstanceType: &instanceType,
			ImageId:      &cvmImageId,
			LoginSettings: &batch.LoginSettings{
				Password: &cvmPassWord,
			},
			SecurityGroupIds: []*string{&securityGroupId},
			InternetAccessible: &batch.InternetAccessible{
				InternetMaxBandwidthOut: &internetMaxBandwidthOut,
				PublicIpAssigned:        &publicIpAssigned,
			},
			SystemDisk: &batch.SystemDisk{
				DiskType: &diskType,
				DiskSize: &diskSize,
			},
		},
	}
	request.Placement = &batch.Placement{
		Zone:      &zone,
		ProjectId: &projectId,
	}

	client := GetClient()
	createEnvLock.Lock()
	defer createEnvLock.Unlock()
	RequestLimit(createEnvLimit, &createEnvLastRequestCount, &createEnvLastRequestTime)
	response, err := client.CreateComputeEnv(request)
	if err != nil {
		return nil, err
	}
	return response.Response.EnvId, nil
}

// 查询计算环境详情
func DescribeEnv(envId string) (*batch.DescribeComputeEnvResponse, error) {
	request := batch.NewDescribeComputeEnvRequest()
	request.EnvId = common.StringPtr(envId)

	client := GetClient()
	describeEnvLock.Lock()
	defer describeEnvLock.Unlock()
	RequestLimit(describeEnvLimit, &describeEnvLastRequestCount, &describeEnvLastRequestTime)
	return client.DescribeComputeEnv(request)
}

// 查询计算环境列表详情
func DescribeEnvs(offset, limit uint64) (*batch.DescribeComputeEnvsResponse, error) {
	request := batch.NewDescribeComputeEnvsRequest()

	request.Offset = common.Uint64Ptr(offset)
	request.Limit = common.Uint64Ptr(limit)

	client := GetClient()
	describeEnvsLock.Lock()
	defer describeEnvsLock.Unlock()

	RequestLimit(describeEnvsLimit, &describeEnvsLastRequestCount, &describeEnvsLastRequestTime)
	return client.DescribeComputeEnvs(request)
}

// 删除计算环境
func DeleteEnv(envId string) error {
	request := batch.NewDeleteComputeEnvRequest()
	request.EnvId = common.StringPtr(envId)

	client := GetClient()
	deleteEnvLock.Lock()
	defer deleteEnvLock.Unlock()
	RequestLimit(deleteEnvLimit, &deleteEnvLastRequestCount, &deleteEnvLastRequestTime)
	_, err := client.DeleteComputeEnv(request)
	return err
}
