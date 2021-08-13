package controllers

import (
	"TEFS-BE/pkg/log"
	tc "TEFS-BE/pkg/tencentCloud"
	"github.com/gin-gonic/gin"
)

// 腾讯云私有网络响应体
type VpcResponse struct {
	VpcId    string `json:"vpc_id"`
	SubnetId string `json:"subnet_id"`
}

// @Summary 创建腾讯云私有网络 seq:4
// @Tags 腾讯云环境
// @Description 创建腾讯云私有网络和子网接口
// @Accept  multipart/form-data
// @Produce  json
// @Param tencentCloudSecretId formData string true "腾讯云SecretId"
// @Param tencentCloudSecretKey formData string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"vpc_id":"vpc-hb528v8v", "subnet_id":"xxxxx"}}"
// @Router /cloudEnv/vpc [post]
func (cc CloudController) Vpc(c *gin.Context) {
	// 创建腾讯云私有网络和子网
	// 如果vpcId和subnetId传入，查询腾讯云，如果私有网络和子网存在，直接返回。否真新建，返回新的vpcId和subnetId
	tencentCloudSecretId := c.PostForm("tencentCloudSecretId")
	tencentCloudSecretKey := c.PostForm("tencentCloudSecretKey")

	zone := TefsKubeSecret.Data.Zone
	vpcId := TefsKubeSecret.Data.VpcId
	subnetId := TefsKubeSecret.Data.SubnetId

	// 腾讯云vpc client
	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	vpc := tc.Vpc{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	// 查询vpc列表
	ret, err := vpc.QueryVpc()
	if err != nil {
		fail(c, ErrQueryCloud)
		return
	}
	vpcInfo := ret.Response

	// vpcId subnetId传入，判断vpc和subnet是否已创建
	if len(vpcId) > 0 && len(subnetId) > 0 {
		var vpcIsCreate bool = false
		var subnetIsCreate bool = false

		for _, vpcItem := range vpcInfo.VpcSet {
			if *vpcItem.VpcId == vpcId {
				vpcIsCreate = true
				break
			}
		}

		if vpcIsCreate {
			// 查询subnet列表
			retSubnet, err := vpc.QuerySubnet(vpcId)
			if err != nil {
				log.Error(err.Error())
				fail(c, ErrQueryCloud)
				return
			}
			for _, v := range retSubnet.Response.SubnetSet {
				if *v.SubnetId == subnetId {
					subnetIsCreate = true
					break
				}
			}
		}

		if vpcIsCreate && subnetIsCreate {
			resp(c, VpcResponse{VpcId: vpcId, SubnetId: subnetId})
			return
		}
	}

	// vpc未创建
	// 获取vpc配额，判断配额是否充足
	vpcLimit, err := vpc.QueryVpcLimit()
	if err != nil {
		log.Error(err.Error())
		fail(c, ErrQueryCloud)
		return
	}
	if *vpcInfo.TotalCount >= *vpcLimit {
		fail(c, ErrVpcInsufficientQuota)
		return
	}

	// vpc创建
	notAvailableNames := []string{}
	for _, v := range vpcInfo.VpcSet {
		notAvailableNames = append(notAvailableNames, *v.VpcName)
	}
	newVpcName := generateRandomName("vpc", notAvailableNames)
	newVpc, err := vpc.CreateVpc(newVpcName)
	if err != nil {
		fail(c, ErrSubmitCloud)
		return
	}
	TefsKubeSecret.Data.VpcId = *newVpc.VpcId
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)

	// 创建子网
	newSubnetName := generateRandomName("subnet", []string{})
	subnet, err := vpc.CreateSubnet(newSubnetName, zone, *newVpc.VpcId)
	if err != nil {
		log.Error(err.Error())
		fail(c, ErrSubmitCloud)
		return
	}

	// 返回数据
	data := VpcResponse{
		VpcId:    *newVpc.VpcId,
		SubnetId: *subnet.SubnetId,
	}
	TefsKubeSecret.Data.SubnetId = *subnet.SubnetId
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)
	resp(c, data)
}
