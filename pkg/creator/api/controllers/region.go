package controllers

import (
	"TEFS-BE/pkg/log"
	tc "TEFS-BE/pkg/tencentCloud"
	"github.com/gin-gonic/gin"
)

// 腾讯云地区响应
type RegionResponse struct {
	Region string `json:"region"`
	Zone   string `json:"zone"`
}

// @Summary 设置腾讯云地区信息 seq:1
// @Tags 腾讯云环境
// @Description 设置讯云地区信(region, zone)息接口
// @Accept  json
// @Produce  json
// @Param tencentCloudSecretId query string true "腾讯云SecretId"
// @Param tencentCloudSecretKey query string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"region":"ap-nanjing","zone":"ap-nanjing-3"}}"
// @Router /cloudEnv/region [get]
func (cc CloudController) Region(c *gin.Context) {
	// 接收腾讯云 Secret id，key, 获取创建腾讯云web环境的region和zone。
	// region 默认南京地区
	// zone 查询南京地区zone, 由于有些zone的cvm会存在售竭情况，故选择最新的zone(返回结果列表最后一个)
	// 当zone存在时，查询zone是否存在，存在：直接返回，不存在：查询最新zone
	tencentCloudSecretId := c.Query("tencentCloudSecretId")
	tencentCloudSecretKey := c.Query("tencentCloudSecretKey")
	zone := TefsKubeSecret.Data.Zone

	// 腾讯云cvm client
	credential := tc.Credential{
		SecretId:  tencentCloudSecretId,
		SecretKey: tencentCloudSecretKey,
	}
	cvm := tc.Cvm{
		Credential: &credential,
		Region:     GlobalRegion,
	}

	// zone参数传入，zone查询是否有效
	if len(zone) > 0 {
		allZone, err := cvm.GetZone()
		if err != nil {
			log.Error(err.Error())
			fail(c, ErrQueryCloud)
			return
		}
		var zoneIsSet bool = false
		for _, v := range allZone.Response.ZoneSet {
			if *v.Zone == zone {
				zoneIsSet = true
			}
		}
		if zoneIsSet {
			data := RegionResponse{
				Region: GlobalRegion,
				Zone:   zone,
			}
			resp(c, data)
			return
		}
	}

	// zone参数未传入，或传入无效。设置有效zone
	zones, err := cvm.GetAvailableZone()
	if err != nil {
		log.Error(err.Error())
		fail(c, ErrQueryCloud)
		return
	}
	if len(zones) < 1 {
		fail(c, ErrNotAvailableZone)
		return
	}
	zone = *zones[0].Zone
	data := RegionResponse{
		Region: GlobalRegion,
		Zone:   zone,
	}
	TefsKubeSecret.Data.SecretId = tencentCloudSecretId
	TefsKubeSecret.Data.SecretKey = tencentCloudSecretKey
	TefsKubeSecret.Data.Zone = zone
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)
	resp(c, data)
}
