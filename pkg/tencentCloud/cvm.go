package tencentCloud

import (
	"TEFS-BE/pkg/tencentCloud/batchCompute"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"sync"
)

// 腾讯云cvm服务器
type Cvm struct {
	Credential *Credential
	Region     string
}

var (
	cvmUrlEndpoint  = "cvm.tencentcloudapi.com"
	instanceImageId = "img-bh86p0sv"
	//instanceImageId = "img-b9szn7c4" // cvm镜像已装洋葱，只适用于开发测试用腾讯云账户

	//instanceType = "SA2.MEDIUM4"
	instanceType = "S1.SMALL1"

	// 实例计费方式：PREPAID包年包月POSTPAID_BY_HOUR按小时后付费
	instanceChargeType = "POSTPAID_BY_HOUR"

	// 实例带宽计费方式：BANDWIDTH_POSTPAID_BY_HOUR流量按小时后付费，BANDWIDTH_PREPAID预付费按带宽结算
	instanceInternetChargeType = "TRAFFIC_POSTPAID_BY_HOUR"

	InternetMaxBandwidthOut int64 = 10
	publicIpAssigned        bool  = true

	// cvm API limit describeImage
	describeImageLock            sync.Mutex
	describeImageLimit           int64 = 40
	describeImageRequestCount    int64 = 0
	describeImageLastRequestTime int64 = 0

	// cvm API limit DescribeZones
	describeZonesLock            sync.Mutex
	describeZonesLimit           int64 = 20
	describeZonesRequestCount    int64 = 0
	describeZonesLastRequestTime int64 = 0

	// cvm API limit describeInstances
	describeInstancesLock            sync.Mutex
	describeInstancesLimit           int64 = 40
	describeInstancesRequestCount    int64 = 0
	describeInstancesLastRequestTime int64 = 0
)

// 获取cvm client
func (c Cvm) GetClient() (*cvm.Client, error) {
	credential := common.NewCredential(
		c.Credential.SecretId,
		c.Credential.SecretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = cvmUrlEndpoint
	return cvm.NewClient(credential, c.Region, cpf)
}

// 获取cvm client
func (c Cvm) GetNanJingClient() (*cvm.Client, error) {
	credential := common.NewCredential(
		c.Credential.SecretId,
		c.Credential.SecretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = cvmUrlEndpoint
	return cvm.NewClient(credential, "ap-nanjing", cpf)
}

// cvm镜像id获取镜像信息
func (c Cvm) GetImage(imageID string) ([]*cvm.Image, error) {
	client, err := c.GetClient()
	if err != nil {
		return nil, err
	}
	request := cvm.NewDescribeImagesRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Values: common.StringPtrs([]string{imageID}),
			Name:   common.StringPtr("image-id"),
		},
	}
	request.Offset = common.Uint64Ptr(0)
	request.Limit = common.Uint64Ptr(1)
	describeImageLock.Lock()
	defer describeImageLock.Unlock()
	batchCompute.RequestLimit(describeImageLimit, &describeImageRequestCount, &describeImageLastRequestTime)
	response, err := client.DescribeImages(request)
	if err != nil {
		return nil, err
	}
	return response.Response.ImageSet, nil
}

// 修改镜像分享信息
// permission 操作, SHARE=分享 CANCEL=取消分享
func (c Cvm) ModifyImageSharePermission(imageId, permission string, cloudAccounts []string) error {
	client, err := c.GetNanJingClient()
	if err != nil {
		return err
	}
	request := cvm.NewModifyImageSharePermissionRequest()
	request.AccountIds = common.StringPtrs(cloudAccounts)
	request.ImageId = common.StringPtr(imageId)
	request.Permission = common.StringPtr(permission)
	_, err = client.ModifyImageSharePermission(request)
	if err != nil {
		return err
	}
	return nil
}

// 获取cvm区域信息
func (c Cvm) GetZone() (*cvm.DescribeZonesResponse, error) {
	client, err := c.GetClient()
	if err != nil {
		return nil, err
	}
	request := cvm.NewDescribeZonesRequest()
	describeZonesLock.Lock()
	defer describeZonesLock.Unlock()
	batchCompute.RequestLimit(describeZonesLimit, &describeZonesRequestCount, &describeZonesLastRequestTime)
	response, err := client.DescribeZones(request)
	if err != nil {
		return nil, err
	}
	return response, err
}

// 获取当前地域可用区域
func (c Cvm) GetAvailableZone() ([]*cvm.ZoneInfo, error) {
	zonesResponse, err := c.GetZone()
	if err != nil {
		return nil, err
	}
	var availableZone []*cvm.ZoneInfo
	for _, zone := range zonesResponse.Response.ZoneSet {
		if *zone.ZoneState == "AVAILABLE" {
			availableZone = append(availableZone, zone)
		}
	}
	cvm.NewRunInstancesRequest()
	return availableZone, nil
}

// 获取实例详情
func (c Cvm) GetInstancesDescribe(instanceId string) (*cvm.Instance, error) {
	client, err := c.GetClient()
	if err != nil {
		return nil, err
	}
	request := cvm.NewDescribeInstancesRequest()
	request.InstanceIds = []*string{&instanceId}
	response, err := client.DescribeInstances(request)
	if err != nil {
		return nil, err
	}
	if len(response.Response.InstanceSet) == 0 {
		return nil, fmt.Errorf("not found Instance:%s", instanceId)
	}
	return response.Response.InstanceSet[0], nil
}

// 创建实例
func (c Cvm) CreateInstance(zone, vpcID, subnetID, groupID, passWord string) (*string, *string, error) {
	client, err := c.GetClient()
	if err != nil {
		return nil, nil, err
	}
	instancePassWord := passWord
	request := cvm.NewRunInstancesRequest()
	request.ImageId = &instanceImageId
	request.Placement = &cvm.Placement{Zone: &zone}                         // 可用地区
	request.InstanceType = &instanceType                                    // 实例类型
	request.InstanceChargeType = &instanceChargeType                        // 实例计费类型
	request.LoginSettings = &cvm.LoginSettings{Password: &instancePassWord} // 登录密码
	request.VirtualPrivateCloud = &cvm.VirtualPrivateCloud{ // 网络设置：私有网络 子网
		VpcId:    &vpcID,
		SubnetId: &subnetID,
	}
	request.SecurityGroupIds = []*string{&groupID} // 安全组设置
	request.InternetAccessible = &cvm.InternetAccessible{ // 带宽设置
		InternetChargeType:      &instanceInternetChargeType,
		InternetMaxBandwidthOut: &InternetMaxBandwidthOut,
		PublicIpAssigned:        &publicIpAssigned,
	}
	response, err := client.RunInstances(request)
	if err != nil {
		return nil, nil, err
	}
	return response.Response.InstanceIdSet[0], &instancePassWord, nil
}

// 获取实例机型
func (c Cvm) GetInstanceType(zone string, cpu, memory int64) (*string, error) {
	client, err := c.GetClient()
	if err != nil {
		return nil, err
	}
	request := cvm.NewDescribeInstanceTypeConfigsRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Values: common.StringPtrs([]string{zone}),
			Name:   common.StringPtr("zone"),
		},
	}
	response, err := client.DescribeInstanceTypeConfigs(request)
	if err != nil {
		return nil, err
	}

	instanceTypeList := response.Response.InstanceTypeConfigSet
	for _, instanceType := range instanceTypeList {
		if *instanceType.GPU != 0 || *instanceType.FPGA != 0 {
			continue
		}

		if *instanceType.CPU == cpu && *instanceType.Memory == memory {
			return instanceType.InstanceType, nil
		}
	}
	return nil, fmt.Errorf("no matching instance type was found")
}

// 查看实例列表
// doc: https://cloud.tencent.com/document/product/213/15728
func (c Cvm) DescribeInstances(filters map[string]string, offset, limit int64) (*cvm.DescribeInstancesResponse, error) {
	client, err := c.GetClient()
	if err != nil {
		return nil, err
	}
	request := cvm.NewDescribeInstancesRequest()
	request.Filters = []*cvm.Filter{}
	for k, v := range filters {
		request.Filters = append(request.Filters, &cvm.Filter{
			Values: common.StringPtrs([]string{v}),
			Name:   common.StringPtr(k),
		})
	}
	request.Offset = common.Int64Ptr(offset)
	request.Limit = common.Int64Ptr(limit)
	describeInstancesLock.Lock()
	defer describeInstancesLock.Unlock()
	batchCompute.RequestLimit(describeInstancesLimit, &describeInstancesRequestCount, &describeInstancesLastRequestTime)
	return client.DescribeInstances(request)
}

// 公网ip获取实例信息
func (c Cvm) GetCvmForPublicId(InstanceId string)  (instanceList  []*cvm.Instance, err error) {
	client, err := c.GetClient()
	if err != nil {
		return nil, err
	}

	request := cvm.NewDescribeInstancesRequest()
	request.Filters = []*cvm.Filter {
		&cvm.Filter {
			Name: common.StringPtr("instance-id"),
			Values: common.StringPtrs([]string{InstanceId}),
		},
	}
	request.Offset = common.Int64Ptr(0)
	request.Limit = common.Int64Ptr(1)
	response, err := client.DescribeInstances(request)
	if err != nil {
		return nil, err
	}
	instanceList = response.Response.InstanceSet
	return instanceList, nil
}