package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/log"
	"context"
	"time"
)

var (
	settingService = model.SettingService{}
)

// 创建系统后台配置
func (s Service) CreateSetting(ctx context.Context, in *pb.CreateSettingRequest) (*pb.CreateSettingReply, error) {
	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}

	key := in.GetKey()
	value := in.GetValue()
	describe := in.GetDescribe()
	if len(key) == 0 || len(key) > 56 {
		return nil, InvalidParams.ErrorParam("key", "key is not null and max len 56")
	}
	if len(value) == 0 || len(value) > 5000 {
		return nil, InvalidParams.ErrorParam("value", "value is not null max len 5000")
	}
	if len(describe) > 0 && len(describe) > 255 {
		return nil, InvalidParams.ErrorParam("describe", "max len 225")
	}

	nowTime := time.Now().Unix()
	setting := &model.Setting{
		Title:    key,
		Value:    value,
		Describe: describe,
		CreateAt: nowTime,
		UpdateAt: nowTime,
	}
	if err := settingService.Create(setting).Error; err != nil {
		log.Error(err.Error())
		return nil, CreateRecordFailed.Error()
	}
	return &pb.CreateSettingReply{Message: "ok"}, nil
}

// 更新系统后台配置
func (s Service) UpdateSetting(ctx context.Context, in *pb.UpdateSettingRequest) (*pb.UpdateSettingReply, error) {
	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}

	settingId := in.GetId()
	value := in.GetValue()
	describe := in.GetDescribe()
	if settingId <= 0 {
		return nil, InvalidId.Error()
	}
	if len(value) == 0 || len(value) > 5000 {
		return nil, InvalidParams.ErrorParam("value", "value is not null max len 5000")
	}
	if len(describe) > 0 && len(describe) > 255 {
		return nil, InvalidParams.ErrorParam("describe", "max len 255")
	}

	setting := settingService.Get(settingId)
	if setting.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}
	ups := make(map[string]interface{})
	ups["value"] = value
	if len(describe) > 0 {
		ups["describe"] = describe
	}
	if err := settingService.Update(setting, ups).Error; err != nil {
		return nil, UpdateDbFailed.Error()
	}
	return &pb.UpdateSettingReply{Message: "ok"}, nil
}

// 逻辑删除系统后台配置。
func (s Service) DelSetting(ctx context.Context, in *pb.DelSettingRequest) (*pb.DelSettingReply, error) {
	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}
	settingId := in.GetId()
	if settingId <= 0 {
		return nil, InvalidId.Error()
	}
	setting := settingService.Get(settingId)
	if setting.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}
	nowTime := time.Now().Unix()
	ups := make(map[string]interface{})
	ups["delete_at"] = nowTime
	if err := settingService.Update(setting, ups).Error; err != nil {
		return nil, UpdateDbFailed.Error()
	}
	return &pb.DelSettingReply{Message: "ok"}, nil
}

// 获取系统后台配置详情
func (s Service) GetSetting(ctx context.Context, in *pb.GetSettingRequest) (*pb.GetSettingReply, error) {
	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}
	settingId := in.GetId()
	if settingId <= 0 {
		return nil, InvalidId.Error()
	}
	setting := settingService.Get(settingId)
	if setting.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}
	return &pb.GetSettingReply{
		Setting: &pb.Setting{
			Id:       setting.Id,
			Title:    setting.Title,
			Value:    setting.Value,
			Describe: setting.Describe,
			CreateAt: setting.CreateAt,
			UpdateAt: setting.UpdateAt,
		},
	}, nil
}

// 获取系统后台配置列表
func (s Service) GetSettingList(ctx context.Context, in *pb.GetSettingListRequest) (*pb.GetSettingListReply, error) {
	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}
	offset := in.GetOffset()
	limit := in.GetLimit()
	settings, total := settingService.GetList(offset, limit)
	var settingItems []*pb.Setting
	for _, setting := range settings {
		settingItems = append(settingItems, &pb.Setting{
			Id:       setting.Id,
			Title:    setting.Title,
			Value:    setting.Value,
			Describe: setting.Describe,
			CreateAt: setting.CreateAt,
			UpdateAt: setting.UpdateAt,
		})
	}
	return &pb.GetSettingListReply{Setting: settingItems, Total: total}, nil
}
