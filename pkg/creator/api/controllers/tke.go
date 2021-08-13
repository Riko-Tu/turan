package controllers

import (
	"TEFS-BE/pkg/log"
	tc "TEFS-BE/pkg/tencentCloud"
	"TEFS-BE/pkg/utils"
	"TEFS-BE/pkg/utils/ssh"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/sftp"
	qvpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	cssh "golang.org/x/crypto/ssh"
	"strconv"
	"time"
)

// 腾讯云tke创建响应
type TkeResponse struct {
	ClusterId        string `json:"cluster_id"`
}

// 腾讯云tke创建状态响应
type TkeStatusResponse struct {
	Status string `json:"status"`
}

// 腾讯云tke创建cvm实例响应
type TkeInstanceStatusResponse struct {
	Status     string `json:"status"`
	InstanceId string `json:"instance_id"`
}

// 腾讯云tke创建cvm实例状态响应
type TkeInstancePublicIpAddressResponse struct {
	PublicIpAddress string `json:"public_ip_address"`
}

// @Summary 创建腾讯云TKE集群 seq:8
// @Tags 腾讯云环境
// @Description 创建腾讯云TKE集群接口
// @Accept  multipart/form-data
// @Produce  json
// @Param tencentCloudSecretId formData string true "腾讯云SecretId"
// @Param tencentCloudSecretKey formData string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"cluster_id":"tke-hb528v8v","instance_pass_word":"xxxxxxxxxxxx"}}"
// @Router /cloudEnv/tke [post]
func (cc CloudController) Tke(c *gin.Context) {
	// 接收腾讯云 Secret id，key, 创建腾讯云TKE,并创建cvm实例节点。
	// cvm实例节点配置默认2核4G
	// 参数 TKE集群id (clusterId)传入，查询存在：直接返回传入clusterId，否则创建新的TKE，返回新建clusterId
	tencentCloudSecretId := c.PostForm("tencentCloudSecretId")
	tencentCloudSecretKey := c.PostForm("tencentCloudSecretKey")
	zone := TefsKubeSecret.Data.Zone
	projectId := TefsKubeSecret.Data.ProjectId
	vpcId := TefsKubeSecret.Data.VpcId
	subnetId := TefsKubeSecret.Data.SubnetId
	securityGroupId := TefsKubeSecret.Data.SecurityGroupId
	clusterId := TefsKubeSecret.Data.TkeId

	// 腾讯云tke client
	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	tke := tc.Tke{
		Credential: &credential,
		Region:     GlobalRegion,
	}
	cvm := tc.Cvm{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	// 查询tke列表
	response, err := tke.QueryCluster()
	if err != nil {
		log.Error(err.Error())
		failCloudMsg(c, ErrQueryCloud, err.Error())
		return
	}

	// tke已使用的名字
	var notAvailableNames = []string{}

	// tke集群是否已创建
	clusterIsCreate := false
	clusters := response.Response.Clusters
	for _, cluster := range clusters {
		notAvailableNames = append(notAvailableNames, *cluster.ClusterName)
		if *cluster.ClusterId == clusterId {
			clusterIsCreate = true
			break
		}
	}

	// tke集群已创建，直接返回成功
	if clusterIsCreate {
		resp(c, TkeResponse{ClusterId: clusterId})
		return
	}

	// tke集群未创建， 创建
	tkeName := generateRandomName("cluster", notAvailableNames)
	instancePassWord := utils.GeneratePassWord(16)
	instanceType, err := cvm.GetInstanceType(zone, 2, 4)
	if err != nil {
		log.Error(err.Error())
		failCloudMsg(c, ErrQueryCloud, err.Error())
		return
	}

	projectIdNum, _ := strconv.ParseInt(projectId, 10, 64)
	clusterIdPointer, err := tke.CreateCluster(tkeName, zone,
		instancePassWord, vpcId, subnetId, securityGroupId, instanceType, &projectIdNum)
	if err != nil {
		log.Error(err.Error())
		failCloudMsg(c, ErrSubmitCloud, err.Error())
		return
	}
	TefsKubeSecret.Data.InstancePassWord = instancePassWord
	TefsKubeSecret.Data.TkeId = *clusterIdPointer
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)
	resp(c, TkeResponse{ClusterId: *clusterIdPointer})
}

// @Summary 获取腾讯云TKE集群状态 seq:9
// @Tags 腾讯云环境
// @Description 获取腾讯云TKE集群状态接口 status=Creating(创建中) status=Abnormal(创建异常) status=Running(创建成功)
// @Accept  json
// @Produce  json
// @Param tencentCloudSecretId query string true "腾讯云SecretId"
// @Param tencentCloudSecretKey query string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"status":"Creating"}}"
// @Router /cloudEnv/tke/status [get]
func (cc CloudController) TkeStatus(c *gin.Context) {
	// 查询腾讯云获取tke状态
	tencentCloudSecretId := c.Query("tencentCloudSecretId")
	tencentCloudSecretKey := c.Query("tencentCloudSecretKey")
	clusterId := TefsKubeSecret.Data.TkeId

	// 腾讯云tke client
	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	tke := tc.Tke{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	cluster, err := tke.DescribeCluster(clusterId)
	if err != nil {
		log.Error(err.Error())
		failCloudMsg(c, ErrQueryCloud, err.Error())
		return
	}
	resp(c, TkeStatusResponse{Status: *cluster.ClusterStatus})
}

// @Summary 获取腾讯云TKE cvm实例状态和id seq:10
// @Tags 腾讯云环境
// @Description 获取腾讯云TKE cvm实例状态和id态接口 status=initializing(创建中) status=failed(创建异常) status=running(创建成功)
// @Accept  json
// @Produce  json
// @Param tencentCloudSecretId query string true "腾讯云SecretId"
// @Param tencentCloudSecretKey query string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"status":"initializing","instance_id":"xxxxx"}}"
// @Router /cloudEnv/tke/instance/status [get]
func (cc CloudController) TkeInstanceStatus(c *gin.Context) {
	// 查询腾讯云 获取tke节点实例状态
	tencentCloudSecretId := c.Query("tencentCloudSecretId")
	tencentCloudSecretKey := c.Query("tencentCloudSecretKey")
	clusterId := TefsKubeSecret.Data.TkeId

	// 腾讯云tke client
	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	tke := tc.Tke{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	instanceSet, err := tke.QueryClusterInstance(clusterId)
	if err != nil {
		log.Error(err.Error())
		failCloudMsg(c, ErrQueryCloud, err.Error())
		return
	}
	if len(instanceSet) <= 0 {
		fail(c, ErrNotFoundInstanceForCluster)
		return
	}

	resp(c, TkeInstanceStatusResponse{
		Status:     *instanceSet[0].InstanceState,
		InstanceId: *instanceSet[0].InstanceId,
	})
	TefsKubeSecret.Data.InstanceId = *instanceSet[0].InstanceId
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)
}

// @Summary 获取腾讯云TKE 实例公网ip地址 seq:11
// @Tags 腾讯云环境
// @Description 获取腾讯云TKE 实例公网ip地址接口
// @Accept  json
// @Produce  json
// @Param tencentCloudSecretId query string true "腾讯云SecretId"
// @Param tencentCloudSecretKey query string true "腾讯云SecretKey"
// @Param instanceId query string true "腾讯云TKE的cvm实例id"
// @Success 200 {string} json "{"code":200,"data":{"public_ip_address":"127.0.0.1"}}"
// @Router /cloudEnv/tke/instance/publicIpAddress [get]
func (cc CloudController) TkeInstancePublicIpAddress(c *gin.Context) {
	// 查询腾讯云，获取tke节点实例的公网ip地址
	tencentCloudSecretId := c.Query("tencentCloudSecretId")
	tencentCloudSecretKey := c.Query("tencentCloudSecretKey")
	instanceId := TefsKubeSecret.Data.InstanceId
	securityGroupId := TefsKubeSecret.Data.SecurityGroupId

	// 腾讯云tcvm client
	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	cvm := tc.Cvm{
		Credential: &credential,
		Region:     GlobalRegion,
	}
	vpc := tc.Vpc{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	instanceDescribe, err := cvm.GetInstancesDescribe(instanceId)
	if err != nil {
		log.Error(err.Error())
		failCloudMsg(c, ErrQueryCloud, err.Error())
		return
	}
	PublicIpAddresses := instanceDescribe.PublicIpAddresses
	if len(PublicIpAddresses) <= 0 {
		fail(c, ErrInstanceNotPublicIpAddress)
		return
	}
	publicIpAddress := *PublicIpAddresses[0]

	TefsKubeSecret.Data.InstanceIp = publicIpAddress
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)
	password := TefsKubeSecret.Data.InstancePassWord
	ip := TefsKubeSecret.Data.InstanceIp

	// 上传yaml文件
	var sftpClient *sftp.Client
	var sshClient *cssh.Client
	var connectErr error
	// 链接服务器失败重试,最多3次
	for i := 0; i < 3; i++ {
		sftpClient, sshClient, connectErr = ssh.Connect("ubuntu", password, ip, 22)
		if connectErr != nil {
			time.Sleep(time.Second * 30)
		} else {
			break
		}
	}
	if connectErr != nil {
		fail(c, ErrSshInstanceFailed)
		return
	}
	defer sftpClient.Close()
	defer sshClient.Close()

	// 上传文件
	for _, yamlFile := range yamlFiles {
		if err = ssh.PushFile(sftpClient, yamlFile, remoteDir); err != nil {
			fmt.Println(err.Error())
			fail(c, ErrUploadYamlFailed)
			return
		}
	}

	// 执行指令
	for _, cmd := range remoteCmds {
		err = ssh.RemoteCmd(sshClient, cmd)
		if err != nil {
			fmt.Println(err.Error())
			fail(c, ErrExecRemoteCmdFailed)
			return
		}
	}

	// 查询安全规则
	securityGroupPolicySet, e := vpc.QuerySecurityGroupPolicies(securityGroupId)
	if e != nil {
		log.Error(e.Error())
		fail(c, ErrQuerySecurityGroup)
		return
	}
	var policyIndexList []int64
	for _, securityGroupPolicy := range securityGroupPolicySet.Ingress {
		policyIndexList = append(policyIndexList, *securityGroupPolicy.PolicyIndex)
	}

	// 删除安全组入站规则
	if len(policyIndexList) > 0 {
		if e = vpc.DeleteSecurityGroupPolicies(securityGroupId, policyIndexList); e != nil {
			log.Error(e.Error())
			fail(c, ErrDelSecurityGroup)
			return
		}
	}

	// 添加安全组入站规则
	all := "ALL"
	action := "ACCEPT"
	protocol := "TCP"
	ingressPolicyList := []*qvpc.SecurityGroupPolicy{}
	for index := range intranetIps {
		ingressPolicyList = append(ingressPolicyList, &qvpc.SecurityGroupPolicy{
			Protocol:  &all,  // 协议 TCP
			Port:      &all,
			CidrBlock: &intranetIps[index],
			Action:    &action,
		})
	}
	for index := range productionCidrBlocks {
		ingressPolicyList = append(ingressPolicyList, &qvpc.SecurityGroupPolicy{
			Protocol:  &protocol,  // 协议 TCP
			Port:      &ingressPorts,     // 端口,可以多个 例，80,22
			CidrBlock: &productionCidrBlocks[index], // ip
			Action:    &action,    // 允许or阻止
		})
		ingressPolicyList = append(ingressPolicyList, &qvpc.SecurityGroupPolicy{
			Protocol:  &protocol,  // 协议 TCP
			Port:      &shellPort,     // 端口,可以多个 例，80,22
			CidrBlock: &productionCidrBlocks[index], // ip
			Action:    &action,    // 允许or阻止
		})
	}

	if e = vpc.AddSecurityGroupPolicies(securityGroupId, "ingress", ingressPolicyList); e != nil {
		log.Error(e.Error())
		fail(c, ErrSubmitCloud)
		return
	}

	resp(c, TkeInstancePublicIpAddressResponse{PublicIpAddress: publicIpAddress})
}
