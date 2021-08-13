package sms

import (
	"github.com/spf13/viper"
)

var sms *Sms

// 腾讯云SMS短信服务
type Sms struct {
	region                            string
	secretId                          string
	secretKey                         string
	sdkAppId                          string
	sign                              string
	RegisterTemplateId                string
	EditUserTemplateId                string
	VaspLicenseTemplateId             string
	ExperimentNotifyTemplateId        string
	AddProjectSuccessNotifyTemplateId string
	AddProjectFailedNotifyTemplateId  string
}

func Setup() {
	sms = &Sms{
		region:                            viper.GetString("tencentCloud.region"),
		secretId:                          viper.GetString("tencentCloud.secretId"),
		secretKey:                         viper.GetString("tencentCloud.secretKey"),
		sdkAppId:                          viper.GetString("tencentCloud.sms.smsSdkAppid"),
		sign:                              viper.GetString("tencentCloud.sms.sign"),
		RegisterTemplateId:                viper.GetString("tencentCloud.sms.registerTemplateId"),
		EditUserTemplateId:                viper.GetString("tencentCloud.sms.editUserTemplateId"),
		VaspLicenseTemplateId:             viper.GetString("tencentCloud.sms.vaspLicenseTemplateId"),
		ExperimentNotifyTemplateId:        viper.GetString("tencentCloud.sms.experimentNotifyTemplateId"),
		AddProjectSuccessNotifyTemplateId: viper.GetString("tencentCloud.sms.addProjectSuccessNotifyTemplateId"),
		AddProjectFailedNotifyTemplateId:  viper.GetString("tencentCloud.sms.addProjectFailedNotifyTemplateId"),
	}
}

func GetSms() *Sms {
	return sms
}

// todo:(v_vwwwang) 短信合规后开放
//// 获取sms client
//func getClient() (*tcSms.Client, error) {
//	credential := common.NewCredential(
//		sms.secretId,
//		sms.secretKey,
//	)
//	cpf := profile.NewClientProfile()
//	cpf.HttpProfile.ReqMethod = "POST"
//	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"
//	cpf.SignMethod = "HmacSHA1"
//	return tcSms.NewClient(credential, sms.region, cpf)
//}

// 发送短信
func SendSms(phone, template *string, templateParamSet []*string) bool {
	// todo 腾讯云的短信功能存在风险,整改过后再开放，删除下面一行代码即可
	return false

	//client, err := getClient()
	//if err != nil {
	//	return false
	//}
	//request := tcSms.NewSendSmsRequest()
	//request.Sign = &sms.sign
	//request.SmsSdkAppid = &sms.sdkAppId
	//request.TemplateID = template
	//request.TemplateParamSet = templateParamSet
	//request.PhoneNumberSet = []*string{phone}
	//response, err := client.SendSms(request)
	//if err != nil {
	//	log.Error(fmt.Sprintf("send sms(%s) err:%s", *phone, err.Error()))
	//	return false
	//}
	//sendStatus := response.Response.SendStatusSet[0]
	//if *sendStatus.Code != "Ok" {
	//	sendStatusByte, _ := json.Marshal(sendStatus)
	//	log.Error(fmt.Sprintf("send sms(%s) failed:%s", *phone, string(sendStatusByte)))
	//	return false
	//}
	//log.Info(fmt.Sprintf("send sms(%s) success", *phone))
	//return true
}
