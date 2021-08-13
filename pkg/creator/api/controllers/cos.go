package controllers

import (
	"TEFS-BE/pkg/log"
	tc "TEFS-BE/pkg/tencentCloud"
	"fmt"
	"github.com/gin-gonic/gin"
	"regexp"
)

// cos响应体
type CosResponse struct {
	CosBucketName string `json:"cos_bucket_name"`
}

// cos腾讯云限额(限制创建个数)
const CosBucketLimit = 200

// 腾讯云appid正则格式
var cloudAppIdRe = regexp.MustCompile(`^\d{10}$`)

// @Summary 创建腾讯云对象存储COS seq:7
// @Tags 腾讯云环境
// @Description 创建腾讯云对象存储COS接口
// @Accept  multipart/form-data
// @Produce  json
// @Param tencentAppId formData string true "腾讯云AppId"
// @Param tencentCloudSecretId formData string true "腾讯云SecretId"
// @Param tencentCloudSecretKey formData string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"cos_bucket_name":"tefs-cos-asd134s"}}"
// @Router /cloudEnv/cos [post]
func (cc CloudController) Cos(c *gin.Context) {
	// 接收腾讯云 Secret id，key,创建cos存储桶。
	// 当存储桶名字传入时(cosBucketName),会先查询存储桶是否存在。存在：直接返回存储桶名，不存在：创建新存储桶，并返回新的存储桶名
	tencentAppId := c.PostForm("tencentAppId")
	tencentCloudSecretId := c.PostForm("tencentCloudSecretId")
	tencentCloudSecretKey := c.PostForm("tencentCloudSecretKey")
	cosBucketName := TefsKubeSecret.Data.CosBucket

	if !cloudAppIdRe.MatchString(tencentAppId) {
		fail(c, ErrParamAppId)
		return
	}

	// 腾讯云vpc client
	cosClient := tc.Cos{
		Credential: &tc.Credential{
			AppId:     tencentAppId,
			SecretId:  tencentCloudSecretId,
			SecretKey: tencentCloudSecretKey,
		},
		Region: GlobalRegion,
	}

	// 查询存储桶
	buckets, err := cosClient.GetBuckets()
	if err != nil {
		log.Error(err.Error())
		fail(c, ErrQueryCloud)
		return
	}

	// 如果参数cosBucketName传入，判断cosBucketName是否创建
	if len(cosBucketName) > 0 {
		var bucketIsCreate bool = false
		for _, v := range buckets {
			if v.Name == cosBucketName && v.Region == GlobalRegion {
				bucketIsCreate = true
				break
			}
		}
		if bucketIsCreate {
			// 跨域访问CORS设置
			cosClient.Bucket = cosBucketName
			if err := cosClient.PutBucketCors(); err != nil {
				fail(c, ErrCosBucketSetCorsFailed)
				return
			}
			resp(c, CosResponse{CosBucketName: cosBucketName})
			return
		}
	}

	// 判断cos配额
	if len(buckets) >= CosBucketLimit {
		fail(c, ErrCosBucketInsufficientQuota)
		return
	}

	// 创建存储桶
	notAvailableNames := []string{}
	for _, v := range buckets {
		notAvailableNames = append(notAvailableNames, v.Name)
	}
	newBucketName := generateRandomName("cos", notAvailableNames)
	newBucketName += "-"
	newBucketName += cosClient.Credential.AppId
	cosClient.Bucket = newBucketName
	if err := cosClient.CreateBucket(); err != nil {
		log.Error(err.Error())
		failCloudMsg(c, ErrSubmitCloud, err.Error())
		return
	}
	TefsKubeSecret.Data.CosBucket = newBucketName
	TefsKubeSecret.Data.AppId = tencentAppId
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)

	// 跨域访问CORS设置
	if err := cosClient.PutBucketCors(); err != nil {
		fmt.Println(err.Error())
		log.Error(err.Error())
		fail(c, ErrCosBucketSetCorsFailed)
		return
	}
	resp(c, CosResponse{CosBucketName: newBucketName})
}
