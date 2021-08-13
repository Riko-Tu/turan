package controllers

import (
	"TEFS-BE/pkg/log"
	tc "TEFS-BE/pkg/tencentCloud"
	"github.com/gin-gonic/gin"
	"strconv"
)

// 项目响应体
type ProjectResponse struct {
	ProjectId int `json:"project_id"`
}

// 创新新项目时统一的详情备注信息
var newProjectDesc = "tefs任务计算使用项目,请勿停用。"

// @Summary 创建腾讯云项目 seq:2
// @Tags 腾讯云环境
// @Description 创建腾讯云项目接口
// @Accept  multipart/form-data
// @Produce  json
// @Param tencentCloudSecretId formData string true "腾讯云SecretId"
// @Param tencentCloudSecretKey formData string true "腾讯云SecretKey"
// @Success 200 {string} json "{"code":200,"data":{"project_id": 1201126}}"
// @Router /cloudEnv/project [post]
func (cc CloudController) Project(c *gin.Context) {
	// 接收腾讯云 Secret id，key,创建腾讯云项目。
	// 当项目id传入时(projectId),会先查询腾讯云项目是否存在。存在：直接返回传入项目id，不存在：创建新项目，并返回新建项目id
	tencentCloudSecretId := c.PostForm("tencentCloudSecretId")
	tencentCloudSecretKey := c.PostForm("tencentCloudSecretKey")
	projectIdStr := TefsKubeSecret.Data.ProjectId

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

	// 判断腾讯云项目是否已创建
	var projectIsCreate bool = false
	var projectId int
	if len(projectIdStr) > 0 {
		projectId, err = strconv.Atoi(projectIdStr)
		if err != nil {
			fail(c, ErrParamProjectId)
			return
		}
		for _, project := range projects {
			if project.ProjectId == projectId {
				projectIsCreate = true
			}
		}
	}

	// 项目已创建
	if projectIsCreate {
		data := ProjectResponse{
			ProjectId: projectId,
		}
		resp(c, data)
		return
	}

	// 项目未创建，创建项目
	// 生成随机可用项目名
	notAvailableNames := []string{}
	for _, project := range projects {
		notAvailableNames = append(notAvailableNames, project.ProjectName)
	}
	newProjectName := generateRandomName("project", notAvailableNames)

	// 创建项目
	projectId, err = tencentCloudAccount.CreateProject(newProjectName, newProjectDesc)
	if err != nil {
		log.Error(err.Error())
		fail(c, ErrSubmitCloud)
		return
	}
	TefsKubeSecret.Data.ProjectId = strconv.Itoa(projectId)
	_ = TefsKubeSecret.Write(TefsKubeSecretYaml)
	data := ProjectResponse{ProjectId: projectId}
	resp(c, data)
}
