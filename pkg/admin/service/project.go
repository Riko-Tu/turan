package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/notifyContent"
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"
)

var (
	projectService = model.ProjectService{}

	projectNameMaxLen     int = 60
	projectDescribeMaxLen int = 200
)

func createPbCloudEnv(cloudEnv *model.CloudEnv) *pb.CloudEnv {
	return &pb.CloudEnv{
		Id:              cloudEnv.Id,
		UserId:          cloudEnv.UserId,
		VaspLicenseId:   cloudEnv.VaspLicenseId,
		CloudSecretId:   cloudEnv.CloudSecretId,
		CloudAppId:      cloudEnv.CloudAppId,
		Region:          cloudEnv.Region,
		Zone:            cloudEnv.Zone,
		CloudAccountId:  cloudEnv.CloudAccountId,
		CloudProjectId:  cloudEnv.CloudProjectId,
		VpcId:           cloudEnv.VpcId,
		SubnetId:        cloudEnv.SubnetId,
		SecurityGroupId: cloudEnv.SecurityGroupId,
		CosBucket:       cloudEnv.CosBucket,
		ClusterId:       cloudEnv.ClusterId,
		InstanceId:      cloudEnv.InstanceId,
		InstanceIp:      cloudEnv.InstanceIp,
		Status:          cloudEnv.Status,
		ErrMsg:          cloudEnv.ErrMsg,
		CreateAt:        cloudEnv.CreateAt,
		UpdateAt:        cloudEnv.UpdateAt,
	}
}

func generateCreateProjectReply(project *model.Project, cloudEnv *model.CloudEnv) *pb.CreateProjectReply {
	projectDate := &pb.Project{
		Id:             project.Id,
		UserId:         project.UserId,
		CloudEnvId:     project.CloudEnvId,
		VaspLicenseId:  project.VaspLicenseId,
		Name:           project.Name,
		InvitationCode: project.InvitationCode,
		Status:         project.Status,
		Describe:       project.Describe,
		Link:           project.Link,
		CreateAt:       project.CreateAt,
		UpdateAt:       project.UpdateAt,
	}
	cloudEnvData := createPbCloudEnv(cloudEnv)

	return &pb.CreateProjectReply{
		Message:  "ok",
		Project:  projectDate,
		CloudEnv: cloudEnvData,
	}
}

func verifyCreateProjectParams(name, describe, cloudSecretId, cloudAppId string) error {
	if len(name) == 0 {
		return InvalidParams.ErrorParam("name", "name is not null")
	}
	if utf8.RuneCountInString(name) > projectNameMaxLen {
		return InvalidParams.ErrorParam("name", "name max len 60")
	}
	if len(describe) > 0 && utf8.RuneCountInString(describe) > projectDescribeMaxLen {
		return InvalidParams.ErrorParam("describe", "describe max len 200")
	}
	if !cloudSecretIdRe.MatchString(cloudSecretId) {
		return InvalidParams.ErrorParam("CloudSecretId", "CloudSecretId format error")
	}
	if !cloudAppIdRe.MatchString(cloudAppId) {
		return InvalidParams.ErrorParam("cloudAppId", "cloudAppId format error")
	}
	return nil
}

func getDbRecord(vaspLicenseId, userId int64, cloudSecretId, name string) (license *model.License,
	cloudEnv *model.CloudEnv, project *model.Project, err error) {
	// 查询license
	license = licenseService.Get(vaspLicenseId)
	if license.Id <= 0 || license.UserId != userId || license.Status == 2 {
		return nil, nil, nil, NotFoundRecord.ErrorRecord("vasp license")
	}

	// 查询cloudEnv
	cloudEnv, err = cloudEnvService.GetByCloudSecretId(cloudSecretId)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, nil, QueryDbFailed.Error()
	}

	// 查询project
	project, err = projectService.GetByUserAndName(userId, name)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, nil, QueryDbFailed.Error()
	}
	return
}

func verifyCloudEnv(cloudEnv *model.CloudEnv, licenseId, projectId int64) (cloudEnvStatus *int64, err error) {
	if cloudEnv.Id > 0 && cloudEnv.VaspLicenseId != licenseId {
		return nil, CloudIdIsBind.Error()
	}
	if (cloudEnv.Id == 3 || cloudEnv.Id == 0) && projectId > 0 {
		return nil, NameRepeat.Error()
	}
	return &cloudEnv.Status, nil
}

func createProjectRecord(user *model.User, cloudEnv *model.CloudEnv, license *model.License,
	name, describe string) (project *model.Project, err error) {
	// 创建新项目
	nowTime := time.Now().Unix()
	project = &model.Project{
		UserId:        user.Id,
		CloudEnvId:    cloudEnv.Id,
		VaspLicenseId: license.Id,
		Name:          name,
		Status:        3,
		Describe:      describe,
		CreateAt:      nowTime,
		UpdateAt:      nowTime,
	}

	// 用户项目记录
	userProject := &model.UserProject{
		UserId:        user.Id,
		VaspLicenseId: project.VaspLicenseId,
		Role:          1,
		Status:        1,
		CreateAt:      nowTime,
		UpdateAt:      nowTime,
	}

	// 创建项目成功的通知
	notify := &model.Notify{
		NotifyType: 0,
		UserId:     user.Id,
		Title:      "创建项目",
		Content:    fmt.Sprintf(notifyContent.CreateProjectSuccess, project.Name),
		CreateAt:   nowTime,
		UpdateAt:   nowTime,
	}

	// 开启事务
	db := database.GetDb()
	tx := db.Begin()
	opSuccess := false
	defer func() {
		if opSuccess {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	// 插入记录，邀请码唯一 数据库字段设置唯一，邀请link根据邀请码生成
	for i := 0; i <= 10; i++ {
		if i == 10 {
			return nil, CreateRecordFailed.Error()
		}
		if err := projectService.CreateInvitationCode(project); err != nil {
			log.Error(err.Error())
			return nil, CreateInvitationCodeFailed.Error()
		}
		projectService.CreateLink(project)

		if err = tx.Create(project).Error; err == nil {
			userProject.ProjectId = project.Id
			notify.ProjectId = project.Id
			err = tx.Create(userProject).Error
			if err != nil {
				log.Error(err.Error())
			}
			err = tx.Create(notify).Error
			if err != nil {
				log.Error(err.Error())
			}
		}
		if err != nil {
			project.Id = 0
			userProject.Id = 0
			log.Error(err.Error())
		} else {
			opSuccess = true
			break
		}
	}
	return
}

func createProjectCloudEnvRecord(user *model.User, license *model.License, name,
	describe, cloudSecretId, cloudAppId string) (project *model.Project, cloudEnv *model.CloudEnv, err error) {
	nowTime := time.Now().Unix()
	project = &model.Project{
		UserId:        user.Id,
		VaspLicenseId: license.Id,
		Name:          name,
		Status:        1, // 创建中
		Describe:      describe,
		CreateAt:      nowTime,
		UpdateAt:      nowTime,
	}
	cloudEnv = &model.CloudEnv{
		UserId:        user.Id,
		VaspLicenseId: license.Id,
		CloudSecretId: cloudSecretId,
		CloudAppId:    cloudAppId,
		Status:        1, // 创建中
		CreateAt:      nowTime,
		UpdateAt:      nowTime,
	}

	// 插入记录，邀请码唯一 数据库字段设置唯一，邀请link根据邀请码生成
	for i := 0; i <= 10; i++ {
		if i == 10 {
			return nil, nil, CreateRecordFailed.Error()
		}
		if err := projectService.CreateInvitationCode(project); err != nil {
			log.Error(err.Error())
			return nil, nil, CreateInvitationCodeFailed.Error()
		}
		projectService.CreateLink(project)

		// 开启事务同时插入project和cloudEnv记录
		if err := projectService.TxCreateProjectAndCloudEnv(project, cloudEnv); err == nil {
			break
		} else {
			log.Error(err.Error())
		}
	}
	return
}

// 创建项目
func (s Service) CreateProject(ctx context.Context, in *pb.CreateProjectRequest) (*pb.CreateProjectReply, error) {
	user := ctx.Value("user").(*model.User)
	name := in.GetName()
	describe := in.GetDescribe()
	vaspLicenseId := in.GetVaspLicenseId()
	cloudSecretId := in.GetCloudSecretId()
	cloudAppId := in.GetCloudAppId()

	if err := verifyCreateProjectParams(name, describe, cloudSecretId, cloudAppId); err != nil {
		return nil, err
	}

	license, cloudEnv, project, err := getDbRecord(vaspLicenseId, user.Id, cloudSecretId, name)
	if err != nil {
		return nil, err
	}

	cloudEnvStatusPrt, err := verifyCloudEnv(cloudEnv, license.Id, project.Id)
	if err != nil {
		return nil, err
	}

	var newProject *model.Project
	switch *cloudEnvStatusPrt {
	case 3:
		newProject, err = createProjectRecord(user, cloudEnv, license, name, describe)
		if err != nil {
			return nil, err
		}
	case 0:
		newProject, cloudEnv, err = createProjectCloudEnvRecord(user, license, name, describe, cloudSecretId, cloudAppId)
		if err != nil {
			return nil, err
		}
	default:
		newProject = projectService.GetByCloudEnvId(cloudEnv.Id)
		if newProject.Id <= 0 {
			return nil, QueryDbFailed.Error()
		}
		if newProject.Name != name {
			ups := make(map[string]interface{})
			ups["name"] = name
			if err := projectService.Update(newProject, ups).Error; err != nil {
				return nil, UpdateDbFailed.Error()
			}
		}
	}

	return generateCreateProjectReply(newProject, cloudEnv), nil
}

// 获取项目详情
func (s Service) GetProject(ctx context.Context, in *pb.GetProjectRequest) (*pb.GetProjectReply, error) {
	user := ctx.Value("user").(*model.User)
	id := in.GetId()
	if id <= 0 {
		return nil, InvalidId.Error()
	}

	project := projectService.Get(id)
	if project.Id <= 0 || project.UserId != user.Id {
		return nil, NotFoundRecord.Error()
	}

	data := &pb.Project{
		Id:             project.Id,
		UserId:         project.UserId,
		CloudEnvId:     project.CloudEnvId,
		VaspLicenseId:  project.VaspLicenseId,
		Name:           project.Name,
		InvitationCode: project.InvitationCode,
		Status:         project.Status,
		Describe:       project.Describe,
		Link:           project.Link,
		CreateAt:       project.CreateAt,
		UpdateAt:       project.UpdateAt,
	}
	return &pb.GetProjectReply{
		Project: data,
	}, nil
}

// 获取项目列表
func (s Service) GetProjectList(ctx context.Context, in *pb.GetProjectListRequest) (*pb.GetProjectListReply, error) {
	user := ctx.Value("user").(*model.User)

	offset := in.GetOffset()
	limit := in.GetLimit()
	status := in.GetStatus()

	var userId int64
	if settingService.UserIsAdmin(user) {
		userId = 0
	} else {
		userId = user.Id
	}

	projectList, total := projectService.GetList(offset, limit, userId, status)
	var data []*pb.Project
	for _, project := range projectList {
		data = append(data, &pb.Project{
			Id:             project.Id,
			UserId:         project.UserId,
			CloudEnvId:     project.CloudEnvId,
			VaspLicenseId:  project.VaspLicenseId,
			Name:           project.Name,
			InvitationCode: project.InvitationCode,
			Status:         project.Status,
			Describe:       project.Describe,
			Link:           project.Link,
			CreateAt:       project.CreateAt,
			UpdateAt:       project.UpdateAt,
		})
	}
	return &pb.GetProjectListReply{
		ProjectList: data,
		Total:       total,
	}, nil
}

// 更新项目
// 只能更新名字和详情
func (s Service) UpdateProject(ctx context.Context, in *pb.UpdateProjectRequest) (*pb.UpdateProjectReply, error) {
	user := ctx.Value("user").(*model.User)
	projectId := in.GetId()
	newName := in.GetNewName()
	newDescribe := in.GetNewDescribe()

	if projectId <= 0 {
		return nil, NotFoundRecord.Error()
	}
	if len(newName) == 0 && len(newDescribe) == 0 {
		return nil, InvalidParams.ErrorParam("newName, newDescribe",
			"params name and describe at least one is needed")
	}
	if utf8.RuneCountInString(newName) > projectNameMaxLen {
		return nil, InvalidParams.ErrorParam("newName", "name max len 60")
	}
	if utf8.RuneCountInString(newDescribe) > projectDescribeMaxLen {
		return nil, InvalidParams.ErrorParam("newDescribe", "describe max len 200")
	}

	project := projectService.Get(projectId)
	if project.UserId != user.Id {
		return nil, NotFoundRecord.Error()
	}

	var ups = make(map[string]interface{})
	ups["name"] = newName
	ups["describe"] = newDescribe

	if err := projectService.Update(project, ups).Error; err != nil {
		return nil, UpdateDbFailed.Error()
	}
	return &pb.UpdateProjectReply{Message: "ok"}, nil
}

// 获取用户最后使用的项目
func (s Service) GetLastProject(ctx context.Context, in *pb.GetLastProjectRequest) (*pb.GetLastProjectReply, error) {
	user := ctx.Value("user").(*model.User)
	project := experimentService.GetLastUpdateRecord(user.Id)
	if project.Id <= 0 {
		project = userProjectService.GetLastUserProject(user.Id)
	}
	var data string
	if project.Id > 0 {
		projectByte, err := json.Marshal(project)
		if err != nil {
			log.Error(err.Error())
			data = "{}"
		} else {
			data = string(projectByte)
		}
	}
	return &pb.GetLastProjectReply{ProjectJson:data}, nil
}