package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/notifyContent"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var userProjectService = model.UserProjectService{}

// 获取用户组织详情列表
func (s Service) GetOrganizationList(ctx context.Context,
	in *pb.GetOrganizationListRequest) (*pb.GetOrganizationListReply, error) {

	user := ctx.Value("user").(*model.User)

	userOrganizationProjects, err := userProjectService.GetOrganizationProjectList(user.Id)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}

	var data = make(map[int64]*pb.UserOrganization)
	for _, v := range userOrganizationProjects {
		vaspLicenseId := v.VaspLicenseId
		_, ok := data[vaspLicenseId]
		if ok {
			continue
		} else {
			data[vaspLicenseId] = &pb.UserOrganization{
				VaspLicenseId:       vaspLicenseId,
				Organization:        v.Organization,
				Domain:              v.Domain,
				OrganizationProject: []*pb.UserProject{},
			}
		}
	}

	for _, v := range userOrganizationProjects {
		userProject := &pb.UserProject{
			Id:          v.Id,
			ProjectId:   v.ProjectId,
			ProjectName: v.ProjectName,
			Role:        v.Role,
			CreateAt:    v.CreateAt,
			UpdateAt:    v.UpdateAt,
		}
		if userProject.Role < 3 {
			userProject.TencentCloudAccount = v.TencentCloudAccount
		} else {
			userProject.TencentCloudAccount = ""
		}
		data[v.VaspLicenseId].OrganizationProject = append(data[v.VaspLicenseId].OrganizationProject, userProject)
	}

	reply := []*pb.UserOrganization{}
	for _, v := range data {
		reply = append(reply, v)
	}

	return &pb.GetOrganizationListReply{
		UserOrganization: reply,
		User: &pb.User{
			Name:     user.Name,
			Email:    user.Email,
			Phone:    user.Phone,
			IsNotify: user.IsNotify,
		},
	}, nil
}

// 获取项目组成员
func (s Service) GetProjectUserList(ctx context.Context,
	in *pb.GetProjectUserListRequest) (*pb.GetProjectUserListReply, error) {

	user := ctx.Value("user").(*model.User)

	projectId := in.GetProjectId()
	project := projectService.Get(projectId)
	if project.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}

	userProject, err := userProjectService.GetByUserIdAndProjectId(user.Id, projectId)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if userProject.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}

	sortField := in.GetSortField().String()
	sort := in.GetSort().String()
	offset := in.GetOffset()
	limit := in.GetLimit()

	projectUserList, total, err := userProjectService.GetProjectUsers(sortField, sort, projectId, offset, limit)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}

	var data []*pb.ProjectUser
	for _, v := range projectUserList {
		data = append(data, &pb.ProjectUser{
			Id:             v.Id,
			UserId:         v.UserId,
			UserName:       v.UserName,
			ProjectRole:    v.Role,
			LastLoginTime:  v.LastLoginAt,
			AddProjectTime: v.CreateAt,
		})
	}

	return &pb.GetProjectUserListReply{
		ProjectUser: data,
		Total:       total,
	}, nil
}

// 变更用户项目记录。（禁用，变更为管理员，变更为普通用户）
func (s Service) ChangeUserProject(ctx context.Context,
	in *pb.ChangeUserProjectRequest) (*pb.ChangeUserProjectReply, error) {

	user := ctx.Value("user").(*model.User)
	userProjectId := in.GetUserProjectId()
	option := in.GetOptions().String()
	if userProjectId <= 0 {
		return nil,  NotFoundRecord.ErrorRecord("userProject")
	}

	userProject, err := userProjectService.Get(userProjectId)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	// 记录不存在，或被禁用
	if userProject.Id <= 0 || userProject.Status != 1 {
		return nil, NotFoundRecord.ErrorRecord("userProject")
	}

	projectId := userProject.ProjectId
	adminUserProject, err := userProjectService.GetByUserIdAndProjectId(user.Id, projectId)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if adminUserProject.Id <= 0 {
		return nil, NotFoundRecord.ErrorRecord("adminUserProject")
	}

	targetUser := userService.Get(userProject.UserId)
	if targetUser.Id <= 0 {
		return nil, NotFoundRecord.ErrorRecord("targetUser")
	}

	var ups = make(map[string]interface{})
	var notifyTitle, notifyValue string
	switch option {
	case "disable": // 禁用用户
		if adminUserProject.Role > 2 || adminUserProject.Role >= userProject.Role {
			return nil, NoAuthority.Error()
		}
		ups["status"] = 2
		notifyTitle = "成员变更"
		notifyValue = fmt.Sprintf(notifyContent.MembersChangeReduce, targetUser.Name)
	case "role_change_admin": // 变更为管理员
		if adminUserProject.Role != 1 {
			return nil, NoAuthority.Error()
		}
		ups["role"] = 2
		notifyTitle = "权限变更"
		notifyValue = fmt.Sprintf(notifyContent.AuthorityChange, targetUser.Name, "普通用户", "管理员")
	case "role_change_common": // 变更为普通用户
		if adminUserProject.Role != 1 {
			return nil, NoAuthority.Error()
		}
		ups["role"] = 3
		notifyTitle = "权限变更"
		notifyValue = fmt.Sprintf(notifyContent.AuthorityChange, targetUser.Name, "管理员", "普通用户")
	default:
		return nil, InvalidParams.ErrorParam("option", "")
	}

	if err := userProjectService.Update(userProject, ups).Error; err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}

	// 通知
	nowTime := time.Now().Unix()
	notify := &model.Notify{
		NotifyType: 2,
		ProjectId:  userProject.ProjectId,
		Title:      notifyTitle,
		Content:    notifyValue,
		CreateAt:   nowTime,
		UpdateAt:   nowTime,
	}
	if err := notifyService.Create(notify).Error; err != nil {
		log.Error(err.Error())
	}

	return &pb.ChangeUserProjectReply{
		Message: "ok",
	}, nil
}

// 按用户获取项目列表
func (s Service) GetUserProjectList(ctx context.Context,
	in *pb.GetUserProjectListRequest) (*pb.GetUserProjectListReply, error) {

	user := ctx.Value("user").(*model.User)
	offset := in.GetOffset()
	limit := in.GetLimit()
	projects, total, err := userProjectService.GetUserProjectList(user.Id, offset, limit)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}

	response := make(map[string]interface{})
	response["my_project"] = projects
	response["total"] = total

	responseByte, err := json.Marshal(response)
	if err != nil {
		log.Error(err.Error())
		return nil, DataToJsonErr.Error()
	}

	return &pb.GetUserProjectListReply{
		Response: string(responseByte),
	}, nil
}
