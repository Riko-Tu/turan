package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/cache"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/notifyContent"
	"TEFS-BE/pkg/tencentCloud/ses"
	"context"
	"fmt"
	"time"
)

// 发送邮件验证码
func (s Service) SendEmailCode(ctx context.Context, in *pb.SendEmailCodeRequest) (*pb.SendEmailCodeReply, error) {
	user := ctx.Value("user").(*model.User)
	toEmail := in.GetEmail()
	if !emailRe.MatchString(toEmail) {
		return nil, EmailFormatError.Error()
	}
	emailCode := GenValidateCode(6)
	var sendSuccess = false

	// redis存储邮箱验证码，设置有效期10分钟,如果发送失败删除redis key
	redisCli := cache.GetRedis()
	key := fmt.Sprintf("user_%d.sendEmailCode_%s", user.Id, toEmail)
	if err := redisCli.SetNX(key, emailCode, time.Second * 60 * 10).Err(); err != nil {
		return nil, fmt.Errorf("save email code failed")
	}
	defer func() {
		if !sendSuccess {
			if err := redisCli.Del(key).Err(); err != nil {
				log.Error(err.Error())
			}
		}
	}()

	subject := "验证码"
	html := fmt.Sprintf(notifyContent.EmailVerifyCode, emailCode)
	if !ses.SendEmail(toEmail, "",html, subject) {
		return nil, SendEmailFailed.Error()
	}
	sendSuccess = true
	return &pb.SendEmailCodeReply{
		Message:              "ok",
	}, nil
}