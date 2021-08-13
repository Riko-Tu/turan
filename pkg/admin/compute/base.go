package compute

import (
	"TEFS-BE/pkg/admin/model"
	laboratoryCli "TEFS-BE/pkg/laboratory/client"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/notifyContent"
	"TEFS-BE/pkg/tencentCloud/ses"
	"TEFS-BE/pkg/tencentCloud/sms"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"strconv"
	"time"
)

var (
	notifyService  = model.NotifyService{}
	userService    = model.UserService{}
	projectService = model.ProjectService{}
	settingServer  = model.SettingService{}
)

// 删除实验cos
func DeleteExperimentCos(cosClient *cos.Client, dir string) error {
	files, err := laboratoryCli.GetExperimentCosFiles(cosClient, dir)
	if err != nil {
		return nil
	}
	count := len(files)
	if count == 0 {
		return nil
	}
	var startIndex, endIndex int
	if count > 1000 {
		endIndex = 1000
	} else {
		endIndex = count
	}
	for startIndex < endIndex {
		if err = laboratoryCli.DeleteExperimentCos(cosClient, files[startIndex:endIndex]); err != nil {
			return err
		}
		startIndex += 1000
		endIndex += 1000
		if endIndex > count {
			endIndex = count
		}
	}
	return nil
}

// 实验通知
func ExperimentNotify(success bool, experiment *model.Experiment) {
	// 实验室通知
	user := userService.Get(experiment.UserId)
	project := projectService.Get(experiment.ProjectId)

	var content, ret string
	if success {
		ret = "计算成功"
	} else {
		ret = "计算失败"
	}
	content = fmt.Sprintf(notifyContent.ExperimentRet, project.Name, experiment.Name, ret)

	nowTime := time.Now().Unix()
	notify := &model.Notify{
		NotifyType: 3,
		UserId:     experiment.UserId,
		ProjectId:  experiment.ProjectId,
		Title:      "实验通知",
		Content:    content,
		CreateAt:   nowTime,
		UpdateAt:   nowTime,
	}
	if err := notifyService.Create(notify).Error; err != nil {
		log.Error(err.Error())
	}

	if user.IsNotify != 1 {
		return
	}
	if len(user.Email) > 0 {
		//message := fmt.Sprintf("您的(%s)实验已完成，实验结果：%s，实验所属项目：%s", name, ret, project.Name)
		subject := "实验提醒"
		startTime := time.Unix(experiment.CreateAt, 0).Format("2006-01-02 15:04:05")
		officialEmail := settingService.GetOfficialEmail()
		computeSecond := experiment.DoneAt - experiment.StartAt
		var hour int64 = 3600
		var runTime string
		if computeSecond < hour {
			runTime = strconv.FormatFloat(float64(computeSecond)/60, 'f', 2, 64) + "m"
		} else {
			runTime = strconv.FormatFloat(float64(computeSecond)/3600, 'f', 2, 64) + "h"
		}
		emailHtml := fmt.Sprintf(notifyContent.EmailExperimentDone,
			project.Name, experiment.Name, ret, experiment.Name, startTime, runTime, officialEmail)
		ses.SendEmail(user.Email, "", emailHtml, subject)
	}
	template := sms.GetSms().ExperimentNotifyTemplateId
	templateParamSet := []*string{&experiment.Name, &ret, &project.Name}
	phone := fmt.Sprintf("+86 %s", user.Phone)
	sms.SendSms(&phone, &template, templateParamSet)
}

// 删除实验环境失败
// 邮件通知
func DelEnvFailedEmailNotify(experimentId int64, envId, jid, labAddress string) (sendSuccess bool) {
	emailAddress := settingServer.GetOfficialEmail()
	fmt.Println(emailAddress)
	message := fmt.Sprintf("删除环境失败，请手动删除。experimentId:%d, envId:%s, jid:%s, labAddress:%s", experimentId, envId, jid, labAddress)
	return ses.SendEmail(emailAddress, message, "", "删除实验环境失败提醒")
}