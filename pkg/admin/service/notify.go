package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/log"
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"
)

var notifyService = model.NotifyService{}

// 获取消息通知列表
func (s Service) GetNotifyList(ctx context.Context, in *pb.GetNotifyListRequest) (*pb.GetNotifyListReply, error) {
	user := ctx.Value("user").(*model.User)
	offset := in.GetOffset()
	limit := in.GetLimit()

	var userId int64 = user.Id
	records, unreadTotal, err := notifyService.GetList(offset, limit, userId)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	response := make(map[string]interface{})
	response["unread_total"] = unreadTotal
	response["data"] = records

	responseJson, err := json.Marshal(response)
	if err != nil {
		return nil, DataToJsonErr.Error()
	}
	return &pb.GetNotifyListReply{
		Response: string(responseJson),
	}, nil
}

// 改变通知消息状态（未读 到 已读）
func (s Service) ChangeNotifyStatus(ctx context.Context,
	in *pb.ChangeNotifyStatusRequest) (*pb.ChangeNotifyStatusReply, error) {

	user := ctx.Value("user").(*model.User)
	projectId := in.GetProjectId()
	notifyId := in.GetNotifyId()

	notify := notifyService.Get(notifyId)
	if notify.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}

	if notify.UserId != 0 && notify.UserId != user.Id {
		return nil, NotFoundRecord.Error()
	}

	//if notify.NotifyType == 2 {
	//	userProject, err := userProjectService.GetByUserIdAndProjectId(user.Id, projectId)
	//	if err != nil {
	//		log.Error(err.Error())
	//		return nil, fmt.Errorf("get userProject record failed")
	//	}
	//	if userProject.Role > 2 {
	//		return nil, fmt.Errorf("err notify record")
	//	}
	//}

	if err := notifyService.CreateNotifyStatus(user.Id, projectId, notifyId).Error; err != nil {
		log.Error(err.Error())
		return nil, CreateRecordFailed.Error()
	}
	return &pb.ChangeNotifyStatusReply{Message: "ok"}, nil
}

// 创建系统通知消息记录
func (s Service) CreateSystemNotify(ctx context.Context,
	in *pb.CreateSystemNotifyRequest) (*pb.CreateSystemNotifyReply, error) {

	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}
	title := in.GetTitle()
	content := in.GetContent()

	if utf8.RuneCountInString(title) > 10 {
		return nil, InvalidParams.ErrorParam("title", "title max len 10")
	}
	if utf8.RuneCountInString(content) > 500 {
		return nil, InvalidParams.ErrorParam("content", "content max len 500")
	}
	nowTime := time.Now().Unix()
	notify := &model.Notify{
		NotifyType: 1,
		Title:      title,
		Content:    content,
		CreateAt:   nowTime,
		UpdateAt:   nowTime,
	}
	if err := notifyService.Create(notify).Error; err != nil {
		return nil, CreateRecordFailed.Error()
	}
	return &pb.CreateSystemNotifyReply{Message: "ok"}, nil
}

// 删除系统通知记录
func (s Service) DeleteSystemNotify(ctx context.Context,
	in *pb.DeleteSystemNotifyRequest) (*pb.DeleteSystemNotifyReply, error) {

	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}
	notifyId := in.GetNotifyId()
	if notifyId <= 0 {
		return nil, InvalidId.Error()
	}

	notify := notifyService.Get(notifyId)
	if notify.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}
	if notify.ProjectId != 0 || notify.UserId != 0 {
		return nil, NotifyTypeErr.Error()
	}

	ups := make(map[string]interface{})
	ups["delete_at"] = time.Now().Unix()
	if err := notifyService.Update(notify, ups).Error; err != nil {
		return nil, UpdateDbFailed.Error()
	}
	return &pb.DeleteSystemNotifyReply{Message: "ok"}, nil
}
