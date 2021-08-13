package controllers

import (
	"TEFS-BE/pkg/log"
	tc "TEFS-BE/pkg/tencentCloud"
	"github.com/gin-gonic/gin"
	qvpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// 腾讯云安全组响应体
type SecurityGroupResponse struct {
	SecurityGroupId string `json:"security_group_id"`
}

var (
	// 安全组规则信息
	securityGroupDesc = "tefs任务计算使用安全组,请勿删除。"

	cidrBlock = "0.0.0.0/0"
	productionCidrBlocks = []string{"146.56.192.59", "118.195.139.156", "81.71.9.72"}
	// 外网入站开放端口
	ingressPorts    = "22,3389,80,443,20,21,32500,36000"
	shellPort = "5000-8000"
	// 内网
	intranetIps = []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"}
)

// @Summary 创建腾讯云安全组 seq:5
// @Tags 腾讯云环境
// @Description 创建腾讯云安全组接口
// @Accept  multipart/form-data
// @Produce  json
// @Param tencentCloudSecretId formData string true "腾讯云SecretId"
// @Param tencentCloudSecretKey formData string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"security_group_id":"sg-hkzz4497"}}"
// @Router /cloudEnv/securityGroup [post]
func (cc CloudController) SecurityGroup(c *gin.Context) {
	// 接收腾讯云 Secret id，key, 创建腾讯云web环境的安全组。
	// 当 安全组id 参数传入, 查询是否存在，存在：返回传入安全组id,不存在：创建并返回新建安全组id
	tencentCloudSecretId := c.PostForm("tencentCloudSecretId")
	tencentCloudSecretKey := c.PostForm("tencentCloudSecretKey")
	securityGroupId := TefsKubeSecret.Data.SecurityGroupId

	// 腾讯云vpc client
	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	vpc := tc.Vpc{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	// 查询安全组列表
	ret, err := vpc.QuerySecurityGroup()
	if err != nil {
		fail(c, ErrQueryCloud)
		return
	}

	// securityGroupId参数传入，判断securityGroup是否创建
	if len(securityGroupId) > 0 {
		var securityGroupIsCreate bool = false
		for _, v := range ret.Response.SecurityGroupSet {
			if *v.SecurityGroupId == securityGroupId {
				securityGroupIsCreate = true
			}
		}
		if securityGroupIsCreate {
			resp(c, SecurityGroupResponse{SecurityGroupId: securityGroupId})
			return
		}
	}

	// 如果参数securityGroupId未传入，创建securityGroup
	// 查询配额，判断配额是否足够
	securityGroupLimitSet, err := vpc.QuerySecurityGroupLimit()
	if err != nil {
		log.Error(err.Error())
		fail(c, ErrQueryCloud)
		return
	}
	securityGroupLimit := *securityGroupLimitSet.Response.SecurityGroupLimitSet.SecurityGroupLimit
	if *ret.Response.TotalCount > securityGroupLimit {
		fail(c, ErrSecurityGroupInsufficientQuota)
		return
	}

	// 创建securityGroup
	notAvailableNames := []string{}
	for _, v := range ret.Response.SecurityGroupSet {
		notAvailableNames = append(notAvailableNames, *v.SecurityGroupName)
	}
	newSecurityGroupName := generateRandomName("sg", notAvailableNames)
	securityGroup, err := vpc.CreateSecurityGroup(newSecurityGroupName, securityGroupDesc)
	if err != nil {
		log.Error(err.Error())
		fail(c, ErrSubmitCloud)
		return
	}

	TefsKubeSecret.Data.SecurityGroupId = *securityGroup.SecurityGroupId
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)
	resp(c, SecurityGroupResponse{SecurityGroupId: *securityGroup.SecurityGroupId})
}

// @Summary 腾讯云安全组添加开放访问规则 seq:6
// @Tags 腾讯云环境
// @Description 腾讯云安全组添加开放访问规则接口
// @Accept  multipart/form-data
// @Produce  json
// @Param tencentCloudSecretId formData string true "腾讯云SecretId"
// @Param tencentCloudSecretKey formData string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"ok":true}}"
// @Router /cloudEnv/securityGroup/policies [post]
func (cc CloudController) SecurityGroupPolicies(c *gin.Context) {
	// 接收腾讯云 Secret id，key,安全组id, 添加腾讯云web环境的安全组访问规则。
	tencentCloudSecretId := c.PostForm("tencentCloudSecretId")
	tencentCloudSecretKey := c.PostForm("tencentCloudSecretKey")
	securityGroupId := TefsKubeSecret.Data.SecurityGroupId

	// 腾讯云vpc client
	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	vpc := tc.Vpc{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	// 查询安全组列表
	ret, err := vpc.QuerySecurityGroup()
	if err != nil {
		fail(c, ErrQueryCloud)
		return
	}

	// 判断安全组是否存在
	var securityGroupIsCreate bool = false
	for _, v := range ret.Response.SecurityGroupSet {
		if *v.SecurityGroupId == securityGroupId {
			securityGroupIsCreate = true
			break
		}
	}
	if !securityGroupIsCreate {
		fail(c, ErrNotSetSecurityGroup)
		return
	}

	all := "ALL"
	action := "ACCEPT"
	protocol := "TCP"
	// 添加入站访问规则
	// 1.开放内网规则，
	ingressPolicyList := []*qvpc.SecurityGroupPolicy{}
	// 2.对外开放端口
	ingressPolicyList = append(ingressPolicyList, &qvpc.SecurityGroupPolicy{
		Protocol:  &protocol,  // 协议 TCP
		Port:      &ingressPorts,     // 端口,可以多个 例，80,22
		CidrBlock: &cidrBlock, // ip
		Action:    &action,    // 允许or阻止
	})
	if err := vpc.AddSecurityGroupPolicies(securityGroupId, "ingress", ingressPolicyList); err != nil {
		log.Error(err.Error())
		fail(c, ErrSubmitCloud)
		return
	}

	// 添加出站规则
	egressPolicyList := []*qvpc.SecurityGroupPolicy{}
	egressPolicyList = append(egressPolicyList, &qvpc.SecurityGroupPolicy{
		Protocol:  &all,
		Port:      &all,
		CidrBlock: &cidrBlock,
		Action:    &action,
	})
	if err := vpc.AddSecurityGroupPolicies(securityGroupId,"egress", egressPolicyList); err != nil {
		log.Error(err.Error())
		fail(c, ErrSubmitCloud)
		return
	}

	data := make(map[string]bool)
	data["ok"] = true
	resp(c, data)
}
