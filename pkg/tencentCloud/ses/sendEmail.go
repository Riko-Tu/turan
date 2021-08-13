package ses

import (
	"TEFS-BE/pkg/admin/model"
	"TEFS-BE/pkg/log"
	"encoding/json"
	"fmt"
)

var (
	defaultEmailAlias = "TEFS"
	settingService = model.SettingService{}
)

func SendEmail(toEmail, message, html, subject string) bool {
	// 发送邮件验证码
	fromEmail := GetSendEmail()
	alias := settingService.GetEmailAlias()
	if len(alias) == 0 {
		alias = defaultEmailAlias
	}
	sendEmailInfo := &SendParams{
		From:       fmt.Sprintf("%s <%s>", alias, fromEmail),
		To:         toEmail,
		Subject:    fmt.Sprintf("TEFS【%s】", subject),
		Text:       message,
		Html: 		html,
	}
	resp, err := Send(sendEmailInfo)
	if err != nil {
		logMsg := fmt.Sprintf("send email(%s) to (%s) err:%s", subject, toEmail, err.Error())
		log.Error(logMsg)
		return false
	}
	if resp.Message != "SUCCESS" {
		respJson, _ := json.Marshal(resp)
		logMsg := fmt.Sprintf("send email(%s) to (%s) failed:%s", subject, toEmail, string(respJson))
		log.Error(logMsg)
		return false
	}
	logMsg := fmt.Sprintf("send email(%s) to (%s) success", subject, toEmail)
	log.Info(logMsg)
	return true
}

func SendEmailRespCode(toEmail, message, html, subject string) (isSuccess bool, code int) {
	// 发送邮件验证码
	fromEmail := GetSendEmail()
	alias := settingService.GetEmailAlias()
	if len(alias) == 0 {
		alias = defaultEmailAlias
	}
	sendEmailInfo := &SendParams{
		From:       fmt.Sprintf("%s <%s>", alias, fromEmail),
		To:         toEmail,
		Subject:    fmt.Sprintf("TEFS【%s】", subject),
		Text:       message,
		Html: 		html,
	}
	resp, err := Send(sendEmailInfo)
	if err != nil {
		logMsg := fmt.Sprintf("send email(%s) to (%s) err:%s", subject, toEmail, err.Error())
		log.Error(logMsg)
		return
	}
	if resp.Message != "SUCCESS" {
		respJson, _ := json.Marshal(resp)
		logMsg := fmt.Sprintf("send email(%s) to (%s) failed:%s", subject, toEmail, string(respJson))
		log.Error(logMsg)
		code = resp.Code
		return
	}
	logMsg := fmt.Sprintf("send email(%s) to (%s) success", subject, toEmail)
	log.Info(logMsg)
	isSuccess = true
	return
}