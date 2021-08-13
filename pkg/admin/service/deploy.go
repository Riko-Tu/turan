package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	labClient "TEFS-BE/pkg/laboratory/client"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/notifyContent"
	"context"
	"fmt"
	"time"
)

func (s Service) DeployUserWeb(ctx context.Context, in *pb.DeployUserWebRequest) (
	*pb.DeployUserWebReply, error) {

	user := ctx.Value("user").(*model.User)
	cloudEnvId := in.GetCloudEnvId()
	projectId := in.GetProjectId()
	if cloudEnvId <= 0 {
		return nil, InvalidId.ErrorParam("cloudEnvId", "")
	}
	if projectId <= 0 {
		return nil, InvalidId.ErrorParam("projectId", "")
	}

	cloudEnv := cloudEnvService.Get(cloudEnvId)
	if cloudEnv.Id <= 0 || cloudEnv.UserId != user.Id {
		return nil, NotFoundRecord.ErrorRecord("cloudEnv")
	}

	project := projectService.Get(projectId)
	if project.Id <= 0 || cloudEnv.Id != project.CloudEnvId || project.UserId != user.Id {
		return nil, NotFoundRecord.ErrorRecord("project")
	}

	ip := cloudEnv.InstanceIp
	if len(ip) == 0 {
		return nil, CloudEnvNotCreate.Error()
	}

	// 验证用户部署服务是否能访问
	address := fmt.Sprintf("%s:32500", ip)
	client, err := labClient.GetClient(address)
	if err != nil {
		log.Error(err.Error())
		return nil, GetClientError.Error()
	}

	requestSuccess := false
	for i := 0; i < 6; i++ {
		_, err = labClient.AvailableZoneList(client)
		if err != nil {
			log.Error(err.Error())
			time.Sleep(time.Second * 10)
		} else {
			requestSuccess = true
			break
		}
	}
	if !requestSuccess {
		return nil, RequestError.Error()
	}

	// 使用事务更新腾讯云环境和项目记录，并创建一条用户项目记录
	var cloudEnvUps = make(map[string]interface{})
	var projectUps = make(map[string]interface{})
	nowTime := time.Now().Unix()
	cloudEnvUps["status"] = 3
	projectUps["status"] = 3
	projectUps["update_at"] = nowTime
	cloudEnvUps["update_at"] = nowTime

	userProject := &model.UserProject{
		UserId:        user.Id,
		ProjectId:     project.Id,
		VaspLicenseId: project.VaspLicenseId,
		Role:          1,
		Status:        1,
		CreateAt:      nowTime,
		UpdateAt:      nowTime,
	}

	if err := cloudEnvService.TxUpdateProjectAndCloudEnv(project,
		cloudEnv, userProject, projectUps, cloudEnvUps); err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}

	// 创建项目成功的通知
	notify := &model.Notify{
		NotifyType: 0,
		ProjectId:  project.Id,
		UserId:     user.Id,
		Title:      "创建项目",
		Content:    fmt.Sprintf(notifyContent.CreateProjectSuccess, project.Name),
		CreateAt:   nowTime,
		UpdateAt:   nowTime,
	}
	if err := notifyService.Create(notify).Error; err != nil {
		log.Error(err.Error())
	}

	return &pb.DeployUserWebReply{
		Message: "ok",
	}, nil
}
