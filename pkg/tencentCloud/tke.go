package tencentCloud

import (
	"encoding/json"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
)

// 腾讯云tke容器服务
type Tke struct {
	Credential *Credential
	Region     string
}

var (
	tkeUrlEndpoint     = "tke.tencentcloudapi.com"
	clusterType        = "MANAGED_CLUSTER"
	defaultClusterCIDR = "192.168.0.0/16"
	//defaultClusterCIDR = "172.18.0.0/16" // test
	defaultNodeRole = "WORKER"

	InstanceChargePrepaid       = "PREPAID"               // 实例计费类型 PREPAID：预付费，即包年包月
	InstanceRenewFlag           = "NOTIFY_AND_AUTO_RENEW" // 实力自动续费标识：通知过期且自动续费
	InstancePeriod        int64 = 6                       // 实力购买时长，单位月
	DiskSize              int64 = 1024                    // 硬盘size
)

// 获取TKE client
func (t Tke) GetClient() (*tke.Client, error) {
	credential := common.NewCredential(
		t.Credential.SecretId,
		t.Credential.SecretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = tkeUrlEndpoint
	return tke.NewClient(credential, t.Region, cpf)
}

// 创建TKE集群(托管集群)
func (t Tke) CreateCluster(name, zone, instancePassWord, vpcId,
	subnetId, securityGroupId string, instanceType *string, projectId *int64) (ClusterId *string, err error) {

	client, err := t.GetClient()
	if err != nil {
		return nil, err
	}
	request := tke.NewCreateClusterRequest()

	// 集群类型 托管集群：MANAGED_CLUSTER，独立集群：INDEPENDENT_CLUSTER
	request.ClusterType = &clusterType

	// 集群cidr用于集群ip创建，需要注意与集群VPC的CIDR冲突。这里写死
	clusterCIDRSettings := tke.ClusterCIDRSettings{
		ClusterCIDR: &defaultClusterCIDR,
	}
	request.ClusterCIDRSettings = &clusterCIDRSettings

	// 集群基本配置
	clusterBasicSettings := tke.ClusterBasicSettings{
		ClusterName: &name, // 集群名称
		VpcId:       &vpcId,
		ProjectId:   projectId, // 所属项目ID
	}
	request.ClusterBasicSettings = &clusterBasicSettings

	//cvm实例配置 按时收费
	var defaultInstance = cvm.Instance{
		// 硬盘size
		SystemDisk: &cvm.SystemDisk{
			DiskSize: &DiskSize,
		},
		// 镜像
		ImageId: &instanceImageId,
		// 实例类型：标准型 2核4G, 收费 0.31元/小时
		InstanceType: instanceType,

		InstanceChargeType: &instanceChargeType,
		// 地区信息,如果地区限购会创建实例失败
		Placement: &cvm.Placement{
			Zone:      &zone,
			ProjectId: projectId,
		},
		// 登录密码
		LoginSettings: &cvm.LoginSettings{
			Password: &instancePassWord,
		},
		// 私有网络配置
		VirtualPrivateCloud: &cvm.VirtualPrivateCloud{
			VpcId:    &vpcId,
			SubnetId: &subnetId,
		},
		// 安全组配置
		SecurityGroupIds: []*string{&securityGroupId},
		// 带宽信息，公网可访问性，公网使用计费模式，最大带宽等
		InternetAccessible: &cvm.InternetAccessible{
			InternetChargeType:      &instanceInternetChargeType, // 宽带计费方式:流量按小时后付费
			InternetMaxBandwidthOut: &InternetMaxBandwidthOut,    // 最大带宽2m
			PublicIpAssigned:        &publicIpAssigned,           // 分配公网ip
		},
	}

	//// cvm实例配置 包月，通知过期切自动续费
	//var defaultInstance = cvm.InquiryPriceRunInstancesRequest{
	//	// ImageId: &instanceImageId,
	//	// 实例类型：标准型 2核4G, 收费 0.31元/小时
	//	InstanceType: instanceType,
	//
	//	// 付费方式：预付费，即包年包月
	//	InstanceChargeType: &InstanceChargePrepaid,
	//
	//	// 预付费模式
	//	InstanceChargePrepaid: &cvm.InstanceChargePrepaid{
	//		// 包月个数
	//		Period:    &InstancePeriod,
	//		// 到期通知切自动续费
	//		RenewFlag: &InstanceRenewFlag,
	//	},
	//
	//	// 地区信息,如果地区限购会创建实例失败
	//	Placement: &cvm.Placement{
	//		Zone: &zone,
	//		ProjectId:projectId,
	//	},
	//	// 登录密码
	//	LoginSettings: &cvm.LoginSettings{
	//		Password: &instancePassWord,
	//	},
	//	// 私有网络配置
	//	VirtualPrivateCloud: &cvm.VirtualPrivateCloud{
	//		VpcId:    &vpcId,
	//		SubnetId: &subnetId,
	//	},
	//	// 安全组配置
	//	SecurityGroupIds: []*string{&securityGroupId},
	//	// 带宽信息，公网可访问性，公网使用计费模式，最大带宽等
	//	InternetAccessible: &cvm.InternetAccessible{
	//		InternetChargeType:      &instanceInternetChargeType, // 宽带计费方式:流量按小时后付费
	//		InternetMaxBandwidthOut: &InternetMaxBandwidthOut,    // 最大带宽2m
	//		PublicIpAssigned:        &publicIpAssigned,           // 分配公网ip
	//	},
	//}

	InstancesParaByte, _ := json.Marshal(defaultInstance)
	InstancesPara := string(InstancesParaByte)
	runInstancesForNode := tke.RunInstancesForNode{
		NodeRole:         &defaultNodeRole,          // 节点角色 MASTER_ETCD, WORKER, 这里创建托管集群，指定WORKER
		RunInstancesPara: []*string{&InstancesPara}, // 创建cvm实例json参数
	}
	runInstancesForNodeItems := []*tke.RunInstancesForNode{&runInstancesForNode}
	request.RunInstancesForNode = runInstancesForNodeItems

	response, err := client.CreateCluster(request)
	if err != nil {
		return nil, err
	}
	return response.Response.ClusterId, nil
}

// 使用集群id,删除TKE集群, 策略为销毁实例(仅支持按量计费云主机实例)
func (t Tke) DeleteCluster(clusterID string) error {
	client, err := t.GetClient()
	if err != nil {
		return err
	}
	request := tke.NewDeleteClusterRequest()
	request.ClusterId = &clusterID
	deleteModel := "terminate" // 销毁实例(仅支持按量计费云主机实例)
	request.InstanceDeleteMode = &deleteModel
	_, err = client.DeleteCluster(request)
	return err
}

// 查询集群节点实例
func (t Tke) QueryClusterInstance(clusterId string) ([]*tke.Instance, error) {
	client, err := t.GetClient()
	if err != nil {
		return nil, err
	}
	request := tke.NewDescribeClusterInstancesRequest()
	request.ClusterId = &clusterId
	response, err := client.DescribeClusterInstances(request)
	if txErr, ok := err.(*errors.TencentCloudSDKError); ok {
		if txErr.GetCode() == "ResourceUnavailable.ClusterState" {
			return []*tke.Instance{}, nil
		}
	}
	if err != nil {
		return nil, err
	}
	return response.Response.InstanceSet, nil
}

// 查询集群状态
func (t Tke) DescribeCluster(clusterId string) (*tke.Cluster, error) {
	client, err := t.GetClient()
	if err != nil {
		return nil, err
	}
	request := tke.NewDescribeClustersRequest()
	request.ClusterIds = []*string{&clusterId}
	response, err := client.DescribeClusters(request)
	if err != nil {
		return nil, err
	}
	return response.Response.Clusters[0], err
}

// 添加实例到集群
func (t Tke) AddInstanceToCluster(clusterId, instanceId, securityGroupId string) error {
	client, err := t.GetClient()
	if err != nil {
		return err
	}
	request := tke.NewAddExistedInstancesRequest()
	request.ClusterId = &clusterId
	request.InstanceIds = []*string{&instanceId}
	request.SecurityGroupIds = []*string{&securityGroupId}

	_, err = client.AddExistedInstances(request)
	if err != nil {
		return err
	}
	return nil
}

// 集群查询
func (t Tke) QueryCluster() (*tke.DescribeClustersResponse, error) {
	client, err := t.GetClient()
	if err != nil {
		return nil, err
	}
	request := tke.NewDescribeClustersRequest()
	response, err := client.DescribeClusters(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// CVM实例创建透传参数，json化字符串格式
func RunInstancesParaJson(zone, instancePassWord, vpcId,
	subnetId, securityGroupId string, instanceType *string) string {

	var instanceParams = cvm.InquiryPriceRunInstancesRequest{
		// 实例类型
		InstanceType: instanceType,

		// 付费方式：预付费，即包年包月
		InstanceChargeType: &InstanceChargePrepaid,

		// 预付费模式
		InstanceChargePrepaid: &cvm.InstanceChargePrepaid{
			Period:    &InstancePeriod,
			RenewFlag: &InstanceRenewFlag,
		},

		// 地区信息,如果地区限购会创建实例失败
		Placement: &cvm.Placement{
			Zone: &zone,
		},
		// 登录密码
		LoginSettings: &cvm.LoginSettings{
			Password: &instancePassWord,
		},
		// 私有网络配置
		VirtualPrivateCloud: &cvm.VirtualPrivateCloud{
			VpcId:    &vpcId,
			SubnetId: &subnetId,
		},
		// 安全组配置
		SecurityGroupIds: []*string{&securityGroupId},
		// 带宽信息，公网可访问性，公网使用计费模式，最大带宽等
		InternetAccessible: &cvm.InternetAccessible{
			InternetChargeType:      &instanceInternetChargeType, // 宽带计费方式:流量按小时后付费
			InternetMaxBandwidthOut: &InternetMaxBandwidthOut,    // 最大带宽2m
			PublicIpAssigned:        &publicIpAssigned,           // 分配公网ip
		},
	}

	InstancesParaByte, _ := json.Marshal(instanceParams)
	return string(InstancesParaByte)
}
