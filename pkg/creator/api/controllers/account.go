package controllers

import (
	"TEFS-BE/pkg/log"
	tc "TEFS-BE/pkg/tencentCloud"
	"github.com/gin-gonic/gin"
	"strconv"
)

// 腾讯云账户响应体
type AccountResponse struct {
	Account int `json:"account"`
}

// 腾讯云账户验证响应体
type AccountVerifyResponse struct {
	Ok bool `json:"ok"`
}

// @Summary 获取腾讯云账户id seq:3
// @Tags 腾讯云环境
// @Description 获取腾讯云主账户id
// @Accept  json
// @Produce  json
// @Param tencentCloudSecretId query string true "腾讯云SecretId"
// @Param tencentCloudSecretKey query string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"account":"100012471344"}}"
// @Router /cloudEnv/account [get]
func (cc CloudController) Account(c *gin.Context) {
	// 使用腾讯云Secret id key,腾讯云项目id, 获取腾讯主账户id的处理函数
	tencentCloudSecretId := c.Query("tencentCloudSecretId")
	tencentCloudSecretKey := c.Query("tencentCloudSecretKey")

	projectIdStr := TefsKubeSecret.Data.ProjectId
	projectId, err := strconv.Atoi(projectIdStr)
	if err != nil {
		fail(c, ErrParamProjectId)
		return
	}

	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	tencentCloudAccount := tc.Account{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	// 查询已有腾讯云项目
	projects, err := tencentCloudAccount.GetProjects()
	if err != nil {
		log.Error(err.Error())
		fail(c, ErrQueryCloud)
		return
	}

	var account int
	for _, v := range projects {
		if v.ProjectId == projectId {
			account = v.CreatorUin
		}
	}

	if account == 0 {
		fail(c, ErrNotFoundProject)
		return
	}
	TefsKubeSecret.Data.Account = strconv.Itoa(account)
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)
	resp(c, AccountResponse{Account: account})
}

// @Summary 腾讯云账户验证
// @Tags 腾讯云环境
// @Description 腾讯云账户验证接口
// @Accept  json
// @Produce  json
// @Param tencentCloudSecretId query string true "腾讯云SecretId"
// @Param tencentCloudSecretKey query string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"ok":true}}"
// @Router /cloudEnv/account/verify [get]
func (cc CloudController) AccountVerify(c *gin.Context) {
	// 对腾讯云Secret id key 进行验证，是否有效，无效直接返回腾讯云显示错误，有效 ok=true
	tencentCloudSecretId := c.Query("tencentCloudSecretId")
	tencentCloudSecretKey := c.Query("tencentCloudSecretKey")

	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	tencentCloudAccount := tc.Account{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	// 查询已有腾讯云项目
	_, err := tencentCloudAccount.GetProjects()
	if err != nil {
		log.Error(err.Error())
		fail(c, ErrQueryCloud)
		return
	}

	TefsKubeSecret.Data.SecretId = tencentCloudSecretId
	TefsKubeSecret.Data.SecretKey = tencentCloudSecretKey
	TefsKubeSecret.Data.Region = GlobalRegion
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)

	resp(c, AccountVerifyResponse{Ok: true})
}
