package tencentCloud

import (
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// 腾讯云vpc私有网络
type Vpc struct {
	Credential *Credential
	Region     string
}

var (
	vpcUrlEndpoint    = "vpc.tencentcloudapi.com"
	defaultVpcCidr    = "172.16.0.0/16"
	defaultSubnetCidr = "172.16.0.0/24"
)

// 获取vpc client
func (v Vpc) GetClient() (*vpc.Client, error) {
	credential := common.NewCredential(
		v.Credential.SecretId,
		v.Credential.SecretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = vpcUrlEndpoint
	return vpc.NewClient(credential, v.Region, cpf)
}

// 创建私有网络
func (v Vpc) CreateVpc(name string) (*vpc.Vpc, error) {
	client, err := v.GetClient()
	if err != nil {
		return nil, err
	}
	request := vpc.NewCreateVpcRequest()
	request.VpcName = &name
	request.CidrBlock = &defaultVpcCidr
	response, err := client.CreateVpc(request)
	if err != nil {
		return nil, err
	}
	return response.Response.Vpc, nil
}

// 删除私有网络
func (v Vpc) DeleteVpc(vpcID string) error {
	client, err := v.GetClient()
	if err != nil {
		return err
	}
	request := vpc.NewDeleteVpcRequest()
	request.VpcId = &vpcID
	_, err = client.DeleteVpc(request)
	return err
}

// 查询私有网络
func (v Vpc) QueryVpc() (*vpc.DescribeVpcsResponse, error) {
	client, err := v.GetClient()
	if err != nil {
		return nil, err
	}
	request := vpc.NewDescribeVpcsRequest()
	response, err := client.DescribeVpcs(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// 查询配额
func (v Vpc)QueryLimit(limitType string) (*uint64, error) {
	client, err := v.GetClient()
	if err != nil {
		return nil, err
	}
	request := vpc.NewDescribeVpcLimitsRequest()
	request.LimitTypes = common.StringPtrs([]string{ limitType })
	response, err := client.DescribeVpcLimits(request)
	if err != nil {
		return nil, err
	}
	limitValue := response.Response.VpcLimitSet[0].LimitValue
	return limitValue, nil
}

// 查询私有网络配额
func (v Vpc)QueryVpcLimit() (*uint64, error) {
	return v.QueryLimit("appid-max-vpcs")
}

// 查询子网配额
func (v Vpc)QuerySubnetLimit() (*uint64, error) {
	return v.QueryLimit("vpc-max-subnets")
}

// 创建子网
func (v Vpc) CreateSubnet(name, zone, vpcID string) (*vpc.Subnet, error) {
	client, err := v.GetClient()
	if err != nil {
		return nil, err
	}
	request := vpc.NewCreateSubnetRequest()
	request.VpcId = &vpcID
	request.Zone = &zone
	request.SubnetName = &name
	request.CidrBlock = &defaultSubnetCidr
	response, err := client.CreateSubnet(request)
	if err != nil {
		return nil, err
	}
	return response.Response.Subnet, nil
}

// 查询子网
func (v Vpc) QuerySubnet(vpcId string) (*vpc.DescribeSubnetsResponse, error) {
	client, err := v.GetClient()
	if err != nil {
		return nil, err
	}
	request := vpc.NewDescribeSubnetsRequest()
	request.Filters = []*vpc.Filter {
		&vpc.Filter {
			Name: common.StringPtr("vpc-id"),
			Values: common.StringPtrs([]string{ vpcId }),
		},
	}
	response, err := client.DescribeSubnets(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// 创建安全组
func (v Vpc) CreateSecurityGroup(name, description string) (*vpc.SecurityGroup, error) {
	client, err := v.GetClient()
	if err != nil {
		return nil, err
	}
	request := vpc.NewCreateSecurityGroupRequest()
	request.GroupName = &name
	request.GroupDescription = &description
	response, err := client.CreateSecurityGroup(request)
	if err != nil {
		return nil, err
	}
	return response.Response.SecurityGroup, nil
}

// 安全组查询
func (v Vpc) QuerySecurityGroup() (*vpc.DescribeSecurityGroupsResponse, error) {
	client, err := v.GetClient()
	if err != nil {
		return nil, err
	}
	request := vpc.NewDescribeSecurityGroupsRequest()
	response, err := client.DescribeSecurityGroups(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// 查询安全组配额
func (v Vpc) QuerySecurityGroupLimit() (*vpc.DescribeSecurityGroupLimitsResponse, error) {
	client, err := v.GetClient()
	if err != nil {
		return nil, err
	}
	request := vpc.NewDescribeSecurityGroupLimitsRequest()
	response, err := client.DescribeSecurityGroupLimits(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// 安全组添加规则
func (v Vpc) AddSecurityGroupPolicies(securityGroupId, policyType string, policyList []*vpc.SecurityGroupPolicy) error {
	client, err := v.GetClient()
	if err != nil {
		return err
	}
	request := vpc.NewCreateSecurityGroupPoliciesRequest()
	request.SecurityGroupId = &securityGroupId

	switch policyType {
	case "ingress":
		request.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
			Ingress: policyList,
		}
	case "egress":
		request.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
			Egress: policyList,
		}
	default:
		return fmt.Errorf("policyType error, is ingress or egress")
	}
	_, err = client.CreateSecurityGroupPolicies(request)
	return err
}

// 查询安全组规则
func (v Vpc) QuerySecurityGroupPolicies(securityGroupId string) (*vpc.SecurityGroupPolicySet, error) {
	client, err := v.GetClient()
	if err != nil {
		return nil, err
	}
	request := vpc.NewDescribeSecurityGroupPoliciesRequest()
	request.SecurityGroupId = common.StringPtr(securityGroupId)
	response, err := client.DescribeSecurityGroupPolicies(request)
	if err != nil {
		return nil, err
	}
	return response.Response.SecurityGroupPolicySet, nil
}

// 删除安全组规则
func (v Vpc) DeleteSecurityGroupPolicies(securityGroupId string, policyIndexList []int64) error {
	client, err := v.GetClient()
	if err != nil {
		return err
	}
	request := vpc.NewDeleteSecurityGroupPoliciesRequest()
	var delList = []*vpc.SecurityGroupPolicy{}
	for _, index := range policyIndexList {
		delList = append(delList, &vpc.SecurityGroupPolicy {
			PolicyIndex: common.Int64Ptr(index),
		})
	}
	request.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{Ingress:delList}
	request.SecurityGroupId = common.StringPtr(securityGroupId)
	_, err = client.DeleteSecurityGroupPolicies(request)
	if err != nil {
		return err
	}
	return nil
}