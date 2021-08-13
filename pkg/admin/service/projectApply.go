package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/notifyContent"
	"TEFS-BE/pkg/tencentCloud/ses"
	"TEFS-BE/pkg/tencentCloud/sms"
	"context"
	"fmt"
	"github.com/jinzhu/gorm"
	"time"
	"unicode/utf8"
)

var projectApplyService = model.ProjectApplyService{}

// 创建项目申请记录
func (s Service) CreateProjectApply(ctx context.Context,
	in *pb.CreateProjectApplyRequest) (*pb.CreateProjectApplyReply, error) {

	user := ctx.Value("user").(*model.User)
	projectInvitationCode := in.GetProjectInvitationCode()
	applyDetails := in.GetDetails()
	if len(projectInvitationCode) <= 0 {
		return nil, InvalidParams.ErrorParam("projectInvitationCode", "format error")
	}
	if len(applyDetails) <= 0 || utf8.RuneCountInString(applyDetails) > 200 {
		return nil, InvalidParams.ErrorParam("applyDetails",
			"invalid applyDetails,not null and max len 200")
	}

	project, err := projectService.GetByInvitationCode(projectInvitationCode)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if project.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}

	// 查询用户是否已经存在项目中
	userProject, err := userProjectService.GetByUserIdAndProjectId(user.Id, project.Id)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if userProject.Id > 0 && userProject.Status != 2 {
		// 用户被禁用，可以在重新申请加入，反之不能申请加入
		return nil, UserDisabled.Error()
	}

	// 查询是否存在申请记录，只能申请一次
	projectApply, err := projectApplyService.GetNotReviewRecord(user.Id, project.Id)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if projectApply.Id > 0 && projectApply.Status == 1 {
		return nil, NearApplication.Error()
	}

	nowTime := time.Now().Unix()
	newProjectApply := &model.ProjectApply{
		UserId:    user.Id,
		ProjectId: project.Id,
		Status:    1,
		Details:   applyDetails,
		CreateAt:  nowTime,
		UpdateAt:  nowTime,
	}
	if err := projectApplyService.Create(newProjectApply).Error; err != nil {
		log.Error(err.Error())
		return nil, CreateRecordFailed.Error()
	}

	// 创建申请记录
	content := fmt.Sprintf(notifyContent.MembersVerify, user.Name, project.Name)
	notify := &model.Notify{
		NotifyType: 2,
		ProjectId:  project.Id,
		Title:      "新人申请",
		Content:    content,
		CreateAt:   nowTime,
		UpdateAt:   nowTime,
	}
	if err := notifyService.Create(notify).Error; err != nil {
		log.Error(err.Error())
	}

	return &pb.CreateProjectApplyReply{
		Message: "ok",
	}, nil
}

func getRecordsForUpdate(userId, projectApplyId int64, tx *gorm.DB) (projectApply *model.ProjectApply,
	adminProject *model.UserProject, err error) {

	projectApply = &model.ProjectApply{}
	// 行锁
	err = tx.Set("gorm:query_option", "FOR UPDATE").First(projectApply, projectApplyId).Error
	if err != nil {
		tx.Rollback()
		log.Error(err.Error())
		return nil, nil, UpdateDbFailed.Error()
	}

	if projectApply.Id <= 0 {
		return nil, nil, NotFoundRecord.Error()
	}
	if projectApply.Status != 1 {
		return nil, nil, RecordIsUsed.Error()
	}

	adminProject, err = userProjectService.GetByUserIdAndProjectId(userId, projectApply.ProjectId)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, QueryDbFailed.Error()
	}
	if adminProject.Id <= 0 {
		return nil, nil, ProjectNotFoundUser.Error()
	}
	if adminProject.Role != 1 && adminProject.Role != 2 {
		return nil, nil, NoAuthority.Error()
	}
	return
}

func adoptHandle(tx *gorm.DB, projectApply *model.ProjectApply, role, vaspLicenseId int64,
	targetUser *model.User, targetProject *model.Project, reviewRetNotify *model.Notify) error {
	// 查询用户项目记录是否存在
	userProject, err := userProjectService.GetByUserIdAndProjectId(projectApply.UserId, projectApply.ProjectId)
	if err != nil {
		log.Error(err.Error())
		return QueryDbFailed.Error()
	}
	nowTime := time.Now().Unix()
	if userProject.Id > 0 {
		// 用户已存在项目中
		if userProject.Status != 2 {
			return UserAlreadyExists.Error()
		} else {
			ups := make(map[string]interface{})
			ups["status"] = 1
			ups["role"] = role
			ups["update_at"] = nowTime
			if err := tx.Model(userProject).Update(ups).Error; err != nil {
				log.Error(err.Error())
				return UpdateDbFailed.Error()
			}
		}
	} else {
		newUserProject := &model.UserProject{
			UserId:        projectApply.UserId,
			ProjectId:     projectApply.ProjectId,
			VaspLicenseId: vaspLicenseId,
			Role:          role,
			Status:        1,
			CreateAt:      nowTime,
			UpdateAt:      nowTime,
		}
		if err := tx.Create(newUserProject).Error; err != nil {
			log.Error(err.Error())
			return CreateRecordFailed.Error()
		}
	}

	// 添加通知信息
	notifys := []*model.Notify{}
	var roleName string
	switch role {
	case 2:
		roleName = "管理员"
	default:
		roleName = "普通用户"
	}
	notify := &model.Notify{
		NotifyType: 2,
		UserId:     0,
		ProjectId:  projectApply.ProjectId,
		Title:      "成员变更",
		Content:    fmt.Sprintf(notifyContent.MembersChangeAdd, targetUser.Name, targetProject.Name, roleName),
		CreateAt:   nowTime,
		UpdateAt:   nowTime,
	}
	notifys = append(notifys, notify)
	reviewRetNotify.Content = fmt.Sprintf(notifyContent.AddProjectSuccess, targetProject.Name, roleName)
	notifys = append(notifys, reviewRetNotify)

	for _, v := range notifys {
		if err := notifyService.Create(v).Error; err != nil {
			log.Error(err.Error())
			return CreateRecordFailed.Error()
		}
	}
	return nil
}

func reviewDoneNotify(targetUser *model.User, projectApply *model.ProjectApply, isAccept bool) {
	// 审核完成 短信邮箱通知用户审核结果
	// 用户需要开启通知
	if targetUser.IsNotify == 1 {
		project := projectService.Get(projectApply.ProjectId)
		// 邮件提醒
		if len(targetUser.Email) > 0 {
			subject := "项目提醒"
			var emailHtml string
			if isAccept {
				officialEmail := settingService.GetOfficialEmail()
				emailHtml = fmt.Sprintf(notifyContent.EmailAddProjectSuccess, project.Name, officialEmail)
				ses.SendEmail(targetUser.Email, "", emailHtml, subject)
			}
		}
		// 短信提醒
		var template string
		if isAccept {
			template = sms.GetSms().AddProjectSuccessNotifyTemplateId
		} else {
			template = sms.GetSms().AddProjectFailedNotifyTemplateId
		}
		templateParamSet := []*string{&project.Name}
		phone := fmt.Sprintf("+86 %s", targetUser.Phone)
		sms.SendSms(&phone, &template, templateParamSet)
	}
}

// 审核项目申请
func (s Service) ReviewProjectApply(ctx context.Context,
	in *pb.ReviewProjectApplyRequest) (*pb.ReviewProjectApplyReply, error) {

	user := ctx.Value("user").(*model.User)
	projectApplyId := in.GetProjectApplyId()
	reviewRet := in.GetReviewRet().String()

	if projectApplyId <= 0 {
		return nil, InvalidId.Error()
	}

	// 开启事务
	db := database.GetDb()
	tx := db.Begin()
	projectApply, adminProject, err := getRecordsForUpdate(user.Id, projectApplyId, tx)
	if err != nil {
		tx.Rollback()
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}

	updateSuccess := false
	defer func() {
		if updateSuccess {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	var ups = make(map[string]interface{})
	var isAccept bool = true
	var role int64 = 3 // 普通用户
	switch reviewRet {
	case "reject":
		ups["status"] = 2
		isAccept = false
	case "accept":
		ups["status"] = 3
	case "set_admin":
		if adminProject.Role != 1 {
			return nil, NoAuthority.Error()
		}
		ups["status"] = 3
		role = 2 // 管理员
	default:
		return nil, InvalidParams.ErrorParam("reviewRet", "not in range")
	}

	// 更新 项目申请记录状态
	if err := tx.Model(projectApply).Update(ups).Error; err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}

	nowTime := time.Now().Unix()
	reviewRetNotify := &model.Notify{
		NotifyType: 2,
		UserId:     projectApply.UserId,
		ProjectId:  projectApply.ProjectId,
		Title:      "加入项目",
		CreateAt:   nowTime,
		UpdateAt:   nowTime,
	}
	targetProject := projectService.Get(projectApply.ProjectId)

	// 申请加入项目的目标用户
	targetUser := userService.Get(projectApply.UserId)
	if targetUser.Id <= 0 {
		return nil, NotFoundRecord.ErrorRecord("targetUser")
	}

	// 项目申请通过，添加用户项目记录
	if isAccept {
		err = adoptHandle(tx, projectApply, role, targetProject.VaspLicenseId,
			targetUser, targetProject, reviewRetNotify)
		if err != nil {
			return nil, err
		}
	} else {
		reviewRetNotify.Content = fmt.Sprintf(notifyContent.AddProjectFailed, targetProject.Name)
		if err := notifyService.Create(reviewRetNotify).Error; err != nil {
			log.Error(err.Error())
			return nil, CreateRecordFailed.Error()
		}
	}
	tx.Commit()

	reviewDoneNotify(targetUser, projectApply, isAccept)
	return &pb.ReviewProjectApplyReply{
		Message: "ok",
	}, nil
}

// 项目申请记录列表
func (s Service) GetProjectApplyList(ctx context.Context,
	in *pb.GetProjectApplyListRequest) (*pb.GetProjectApplyListReply, error) {

	user := ctx.Value("user").(*model.User)

	offset := in.GetOffset()
	limit := in.GetLimit()

	// status = 1 待审核
	projectApplyList, total, err := projectApplyService.GetProjectApplyAndUserInfoList(offset, limit, user.Id)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	var data []*pb.ProjectApply
	for _, v := range projectApplyList {
		data = append(data, &pb.ProjectApply{
			Id:          v.Id,
			UserName:    v.UserName,
			ProjectName: v.ProjectName,
			UserPhone:   v.UserPhone,
			Details:     v.Details,
			CreateAt:    v.CreateAt,
		})
	}
	return &pb.GetProjectApplyListReply{
		ProjectApplyList: data,
		Total:            total,
	}, nil
}
