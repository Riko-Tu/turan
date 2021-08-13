package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/log"
	"context"
	"time"
)

var userService = model.UserService{}

// 注册
func (s Service) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterReply, error) {
	user := ctx.Value("user").(*model.User)
	ups := make(map[string]interface{})
	// 参数验证，手机，邮箱，手机验证码
	phone := in.GetPhone()
	name := in.GetName()
	email := in.GetEmail()
	if ok := phoneRe.MatchString(phone); !ok {
		return nil, InvalidParams.ErrorParam("phone", "format error")
	}
	if !userNameRe.MatchString(name) {
		return nil, InvalidParams.ErrorParam("name", "format error")
	}
	if ok := emailRe.MatchString(email); !ok {
		return nil, InvalidParams.ErrorParam("email", "format error")
	}
	ups["email"] = email
	ups["phone"] = phone
	ups["name"] = name

	// todo 腾讯云的短信功能存在风险,整改过后再开放。后续开放短信功能，放开下面注释代码即可
	//smsCode := in.GetSmsCode()
	//if ok := smsCodeRe.MatchString(smsCode); !ok {
	//	return nil, fmt.Errorf("invalid sms code")
	//}
	//key := phone + "_1"
	//redis := cache.GetRedis()
	//value, err := redis.Get(key).Result()
	//if err != nil && err != cache.RedisNilError {
	//	log.Error(err.Error())
	//	return nil, fmt.Errorf("redis get sms code failed")
	//}
	//if err == cache.RedisNilError || value != smsCode {
	//	return nil, fmt.Errorf("invalid sms code")
	//}

	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	ups["last_login_at"] = nowTime
	if err := userService.Update(user, ups).Error; err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}
	return &pb.RegisterReply{Message: "ok"}, nil
}

// 获取用户信息
func (s Service) GetUser(ctx context.Context, in *pb.GetUserRequest) (*pb.GetUserReply, error) {
	user := ctx.Value("user").(*model.User)
	userInfo := pb.User{
		Id:          user.Id,
		Account:     user.Account,
		Way:         user.Way,
		Name:        user.Name,
		Email:       user.Email,
		Phone:       user.Phone,
		IsNotify:    user.IsNotify,
		CreateAt:    user.CreateAt,
		UpdateAt:    user.UpdateAt,
		LastLoginAt: user.LastLoginAt,
	}
	return &pb.GetUserReply{User: &userInfo}, nil
}

// 批量查询用户
func (s Service) GetUserList(ctx context.Context, in *pb.GetUserListRequest) (*pb.GetUserListReply, error) {
	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}
	offset := in.GetOffset()
	limit := in.GetLimit()

	users, total := userService.GetList(offset, limit)
	var usersInfo []*pb.User
	for _, user := range users {
		usersInfo = append(usersInfo, &pb.User{
			Id:          user.Id,
			Account:     user.Account,
			Way:         user.Way,
			Name:        user.Name,
			Email:       user.Email,
			Phone:       user.Phone,
			IsNotify:    user.IsNotify,
			CreateAt:    user.CreateAt,
			UpdateAt:    user.UpdateAt,
			LastLoginAt: user.LastLoginAt,
		})
	}
	return &pb.GetUserListReply{Users: usersInfo, Total: total}, nil
}

// 用户信息修改
func (s Service) UserEdit(ctx context.Context, in *pb.UserEditRequest) (*pb.UserEditReply, error) {
	user := ctx.Value("user").(*model.User)
	var ups = make(map[string]interface{})
	email := in.GetEmail()
	name := in.GetName()
	agreementVersion := in.GetAgreementVersion()

	isNotify := in.GetIsNotify()
	if isNotify != 0 && isNotify != 1 && isNotify != 2 {
		return nil, InvalidParams.ErrorParam("isNotify", "isNotify in 0 1 2")
	}

	if len(email) > 0 {
		if emailRe.MatchString(email) {
			ups["email"] = email
		} else {
			return nil, InvalidParams.ErrorParam("email", "format error")
		}
	}
	if len(name) > 0 {
		if userNameRe.MatchString(name) {
			ups["name"] = name
		} else {
			return nil, InvalidParams.ErrorParam("name", "format error")
		}
	}
	if isNotify != 0 && isNotify != user.IsNotify {
		ups["is_notify"] = isNotify
	}
	if len(agreementVersion) > 0 {
		if !agreementVersionRe.MatchString(agreementVersion) {
			return nil, InvalidParams.ErrorParam("agreementVersion", "format error")
		}
		ups["agreement_version"] = agreementVersion
	}

	if err := userService.Update(user, ups).Error; err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}
	return &pb.UserEditReply{Message: "ok"}, nil
}

// 重新设置手机号 type=1验证手机验证码，type=2设置新手机
func (s Service) ResetPhone(ctx context.Context, in *pb.ResetPhoneRequest) (*pb.ResetPhoneReply, error) {
	// todo 短信功能限制,短信功能开放后，放开下面注释代码
	user := ctx.Value("user").(*model.User)
	//opType := in.GetType()
	//phone := in.GetPhone()
	//smsCode := in.GetSmsCode()
	newPhone := in.GetNewPhone()
	//newSmsCode := in.GetNewSmsCode()
	//if !phoneRe.MatchString(phone) {
	//	return nil, fmt.Errorf("phone invalid")
	//}
	if !phoneRe.MatchString(newPhone) {
		return nil, InvalidParams.ErrorParam("newPhone", "format error")
	}

	// todo 短信功能限制,短信功能开放后删除下面代码
	var ups = make(map[string]interface{})
	ups["phone"] = newPhone
	if err := userService.Update(user, ups).Error; err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}

	// todo 短信功能限制,短信功能开放后，放开下面注释代码
	//key := phone + "_2"
	//redisClient := cache.GetRedis()
	//storageSmsCode, err := RedisGetSmsCode(redisClient, key)
	//if err != nil {
	//	return nil, err
	//}
	//if storageSmsCode != smsCode {
	//	return nil, fmt.Errorf("sms code invalid")
	//}
	//switch opType {
	//case 1:
	//	return &pb.ResetPhoneReply{Message: "ok"}, nil
	//case 2:
	//	if !phoneRe.MatchString(newPhone) {
	//		return nil, fmt.Errorf("new phone invalid")
	//	}
	//	key2 := newPhone + "_2"
	//	storageSmsCode2, err := RedisGetSmsCode(redisClient, key2)
	//	if err != nil {
	//		return nil, err
	//	}
	//	if storageSmsCode2 != newSmsCode {
	//		return nil, fmt.Errorf("new phone sms code invalid")
	//	}
	//	ret := userService.Update(user, map[string]interface{}{
	//		"phone": newPhone,
	//	})
	//	if err := ret.Error; err != nil {
	//		return nil, fmt.Errorf("reset phone failed")
	//	}
	//	return &pb.ResetPhoneReply{Message: "ok"}, nil
	//default:
	//	return nil, fmt.Errorf("type invalid")
	//}

	// todo:短信功能限制,短信功能开放后删除下面代码
	return &pb.ResetPhoneReply{Message: "ok"}, nil
}
