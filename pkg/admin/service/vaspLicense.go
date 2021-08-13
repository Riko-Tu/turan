package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/cache"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/notifyContent"
	"TEFS-BE/pkg/tencentCloud"
	"TEFS-BE/pkg/tencentCloud/ses"
	"TEFS-BE/pkg/tencentCloud/sms"
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	licenseService           = model.LicenseService{}
	userAddVaspLicenseDirNum = "user_%d.addVaspLicense.cosDirNum"
	userAddVaspLicenseLock   = "user_%d.addVaspLicense.lock"
	licenseKeywordRe         = regexp.MustCompile(`(?i)vasp|(?i)source|(?i)download|(?i)instructions|(?i)link|(?i)portal|(?i)register|(?i)license|(?i)account|(?i)document|(?i)version`)
)

func verifyCreateVaspLicenseParams(organization, domain, bindEmail string) error {
	if len(organization) == 0 || utf8.RuneCountInString(organization) > 20 {
		return InvalidParams.ErrorParam("organization",
			"organization is not null and max len 20")
	}
	if len(domain) > 0 && utf8.RuneCountInString(domain) > 200 {
		return InvalidParams.ErrorParam("domain", "max len 200")
	}
	if !emailRe.MatchString(bindEmail) {
		return InvalidParams.ErrorParam("email", "format error")
	}
	return nil
}

// 验证vasp license 图片
func verifyVaspLicenseImage(email string, objectList []cos.Object) error {
	imageUrls := []string{}
	for _, object := range objectList {
		imageUrls = append(imageUrls,
			fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", bucket, region, object.Key))
	}
	characterRetItems := tencentCloud.Character(imageUrls)
	textItems := tencentCloud.GetDetectedText(characterRetItems)
	count := ""
	for _, v := range textItems {
		count += *v
	}

	var isValid bool
	emails := emailRe.FindAllString(count, -1)
	re := regexp.MustCompile(email)
	for _, email := range emails {
		if re.MatchString(email) {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("image not found bind email")
	}

	keyword := licenseKeywordRe.FindAllString(count, -1)
	tmpMap := make(map[string]byte)
	for _, v := range keyword {
		tmpMap[strings.ToLower(v)] = 0
	}
	if len(tmpMap) < 2 {
		return fmt.Errorf("keywords less than 2")
	}
	return nil
}

// 用户添加vaspLicense
// 支持图片格式 .jpg
// 图片由前端直传到cos,然后后端验证是否上传图片
// 参数op_type=1为获取上传cos临时密钥和指定cos路径
// 参数op_type=2为用户添加vaspLicense记录
func (s Service) CreateVaspLicense(ctx context.Context,
	in *pb.CreateVaspLicenseRequest) (*pb.CreateVaspLicenseReply, error) {

	user := ctx.Value("user").(*model.User)
	opType := in.GetOpType()
	switch opType {
	case 1: // 获取上传license权限
		// 获取上传cos临时密钥，指定上传路径
		secret, cosKeys, err := cosService.GetVaspLicenseTmpSecret(user.Id)
		if err != nil {
			log.Error(err.Error())
			return nil, GetCosTmpSercretFailed.Error()
		}
		dirNum := strings.Split(cosKeys[0], "/")[2]
		redisCli := cache.GetRedis()
		redisKey := fmt.Sprintf(userAddVaspLicenseDirNum, user.Id)
		if err := redisCli.Set(redisKey, dirNum, time.Second*60*30).Err(); err != nil {
			log.Error(err.Error())
			return nil, SetVaspLicenseCosDirFailed.Error()
		}

		return &pb.CreateVaspLicenseReply{
			Message:    "ok",
			UploadPath: cosKeys,
			Credential: &pb.CosCredential{
				TmpSecretID:  secret.Credentials.TmpSecretID,
				TmpSecretKey: secret.Credentials.TmpSecretKey,
				SessionToken: secret.Credentials.SessionToken,
				Bucket:       cosService.Bucket,
				Region:       cosService.Region,
				StartTime:    int64(secret.StartTime),
				ExpiredTime:  int64(secret.ExpiredTime),
				Expiration:   secret.Expiration,
			},
		}, nil
	case 2: // 生成记录
		// redis 客户端
		redisCli := cache.GetRedis()

		// 设置锁防止重复提交
		lockKey := fmt.Sprintf(userAddVaspLicenseLock, user.Id)
		setRet := redisCli.SetNX(lockKey, 1, time.Second*60*3)
		if setRet.Err() != nil {
			return nil, SetLockFailed.Error()
		}
		if !setRet.Val() {
			return nil, SubmitErr.Error()
		}
		defer func() {
			redisCli.Del(lockKey)
		}()

		// 参数获取验证
		organization := in.GetOrganization()
		domain := in.GetDomain()
		bindEmail := in.GetBindEmail()
		vaspLicenseDirNum := in.GetLicenseDirNum()
		if err := verifyCreateVaspLicenseParams(organization, domain, bindEmail); err != nil {
			return nil, err
		}

		// 参数验证：vaspLicens文件名
		redisKey := fmt.Sprintf(userAddVaspLicenseDirNum, user.Id)
		defer func() {
			redisCli.Del(redisKey)
		}()
		ret := redisCli.Get(redisKey)
		if ret.Err() != nil {
			log.Error(ret.Err().Error())
			return nil, vaspLicenseDirNumErr.Error()
		}
		storageVaspLicenseDirNum, err := ret.Int64()
		if err != nil || storageVaspLicenseDirNum != vaspLicenseDirNum {
			return nil, InvalidParams.ErrorParam("vaspLicenseDirNum",
				"not equal to storageVaspLicenseDirNum")
		}

		// 参数验证：邮箱验证码
		sendEmailCodeRedisKey := fmt.Sprintf("user_%d.sendEmailCode_%s", user.Id, bindEmail)
		storageEmailCode, err := redisCli.Get(sendEmailCodeRedisKey).Int64()
		if err == cache.RedisNilError {
			return nil, InvalidParams.ErrorParam("email code", "")
		}
		if err != nil {
			log.Error(err.Error())
			return nil, RedisGetFailed.Error()
		}
		if storageEmailCode != in.GetEmailCode() {
			//return nil, InvalidParams.ErrorParam("email code", "")
			return nil, VerificationCodeError.Error()
		}
		defer func() {
			redisCli.Del(sendEmailCodeRedisKey)
		}()

		//// 查询上传文件
		//cosPath := fmt.Sprintf("user_%d/vasp_license/%d", user.Id, vaspLicenseDirNum)
		//imageList, err := cosService.GetObjectList(cosPath + "/0.jpg")
		//if err != nil {
		//	return nil, ErrorErr.ErrorErr(err)
		//}
		//if len(imageList) == 0 {
		//	return nil, NotUploadImage.Error()
		//}

		// 获取上传到cos的文件列表
		cosPath := fmt.Sprintf("user_%d/vasp_license/%d", user.Id, vaspLicenseDirNum)
		objectList, err := cosService.GetObjectList(cosPath)
		if err != nil {
			return nil, err
		}
		if len(objectList) == 0 {
			return nil, NotUploadImage.Error()
		}

		// 验证上传图片
		// 验证通过数据库状态为3（成功）
		// 验证失败数据库状态1（待人工审核，创建项目不会被限制）
		var status int64 = 3
		if err := verifyVaspLicenseImage(bindEmail, objectList); err != nil {
			status = 1
		}

		// 添加vaspLicense记录
		nowTime := time.Now().Unix()
		license := &model.License{
			UserId:       user.Id,
			Organization: organization,
			Domain:       domain,
			BindEmail:    bindEmail,
			CosPath:      cosPath,
			Status:       status,
			CreateAt:     nowTime,
			UpdateAt:     nowTime,
		}
		if err := licenseService.Create(license).Error; err != nil {
			log.Error(err.Error())
			return nil, CreateRecordFailed.Error()
		}
	default:
		return nil, InvalidParams.ErrorParam("op_type", "")
	}
	return &pb.CreateVaspLicenseReply{Message: "ok"}, nil
}

// 更新vasp license 研究领域和详情
func (s Service) UpdateVaspLicense(ctx context.Context,
	in *pb.UpdateVaspLicenseRequest) (*pb.UpdateVaspLicenseReply, error) {

	user := ctx.Value("user").(*model.User)

	// 参数验证
	licenseId := in.GetId()
	domain := in.GetDomain()
	details := in.GetDetails()
	if licenseId <= 0 {
		return nil, InvalidId.Error()
	}
	if utf8.RuneCountInString(domain) > 200 {
		return nil, InvalidParams.ErrorParam("domain", "max len 200")
	}
	if utf8.RuneCountInString(details) > 200 {
		return nil, InvalidParams.ErrorParam("details", "max len 200")
	}
	if len(domain) == 0 && len(details) == 0 {
		return nil, InvalidParams.ErrorParam("domain, details",
			"params domain details at least one is required")
	}

	// 数据库获取license记录
	license := licenseService.Get(licenseId)
	if user.Id != license.UserId {
		return nil, NotFoundRecord.Error()
	}

	// 更新记录
	var ups = make(map[string]interface{})
	ups["domain"] = domain
	ups["details"] = details
	if err := licenseService.Update(license, ups).Error; err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}
	return &pb.UpdateVaspLicenseReply{Message: "ok"}, nil
}

// 逻辑删除vasp license
func (s Service) DelVaspLicense(ctx context.Context, in *pb.DelVaspLicenseRequest) (*pb.DelVaspLicenseReply, error) {
	user := ctx.Value("user").(*model.User)
	licenseId := in.GetId()
	if licenseId <= 0 {
		return nil, InvalidId.Error()
	}
	license := licenseService.Get(licenseId)
	if user.Id != license.UserId {
		return nil, NotFoundRecord.Error()
	}
	// 只要审核未通过license才可删除
	if license.Status != 2 {
		return nil, LicenseCannotDeleted.Error()
	}
	var ups = make(map[string]interface{})
	ups["delete_at"] = time.Now().Unix()
	if err := licenseService.Update(license, ups).Error; err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}
	return &pb.DelVaspLicenseReply{Message: "ok"}, nil
}

// 获取vasp license 详情
func (s Service) GetVaspLicense(ctx context.Context, in *pb.GetVaspLicenseRequest) (*pb.GetVaspLicenseReply, error) {
	user := ctx.Value("user").(*model.User)
	licenseId := in.GetId()
	if licenseId <= 0 {
		return nil, InvalidId.Error()
	}

	license := licenseService.Get(licenseId)
	if license.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}

	isAdmin := settingService.UserIsAdmin(user)
	if !isAdmin {
		if user.Id != license.UserId {
			return &pb.GetVaspLicenseReply{
				License: &pb.License{Status:license.Status},
			},nil
		}
	}

	cosBasePath := license.CosPath
	cosObjects, err := cosService.GetObjectList(cosBasePath)
	if err != nil {
		return nil, GetCosObjectFailed.Error()
	}
	var licenseImages []string
	for _, v := range cosObjects {
		licenseImages = append(licenseImages, v.Key)
	}
	return &pb.GetVaspLicenseReply{
		License: &pb.License{
			Id:           license.Id,
			UserId:       license.UserId,
			Organization: license.Organization,
			Domain:       license.Domain,
			BindEmail:    license.BindEmail,
			CosPath:      license.CosPath,
			Status:       license.Status,
			CreateAt:     license.CreateAt,
			UpdateAt:     license.UpdateAt,
		},
		Images: licenseImages,
	}, nil
}

// 获取vasp license列表
func (s Service) GetVaspLicenseList(ctx context.Context,
	in *pb.GetVaspLicenseListRequest) (*pb.GetVaspLicenseListReply, error) {

	user := ctx.Value("user").(*model.User)
	offset := in.GetOffset()
	limit := in.GetLimit()
	status := in.GetStatus()
	if status > 4 || status < 0 {
		return nil, InvalidParams.ErrorParam("status", "0 < status < 4")
	}
	var userId int64
	userId = user.Id
	if settingService.UserIsAdmin(user) {
		userId = 0
	}
	licenses, total := licenseService.GetList(offset, limit, userId, status)
	var licenseItems []*pb.License
	for _, license := range licenses {
		licenseItems = append(licenseItems, &pb.License{
			Id:           license.Id,
			UserId:       license.UserId,
			Organization: license.Organization,
			Domain:       license.Domain,
			BindEmail:    license.BindEmail,
			CosPath:      license.CosPath,
			Status:       license.Status,
			CreateAt:     license.CreateAt,
			UpdateAt:     license.UpdateAt,
		})
	}
	return &pb.GetVaspLicenseListReply{
		License: licenseItems,
		Total:   total,
	}, nil
}

// 后台审核vasp license,修改记录状态
// 审核完成短信邮件通知
func (s Service) ReviewVaspLicense(ctx context.Context,
	in *pb.ReviewVaspLicenseRequest) (*pb.ReviewVaspLicenseReply, error) {

	admin := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(admin) {
		return nil, NoAuthority.Error()
	}
	reviewRet := in.GetReviewRet()
	if reviewRet != 2 && reviewRet != 3 {
		return nil, InvalidParams.ErrorParam("reviewRet", "reviewRet in 2 3")
	}
	license := licenseService.Get(in.GetId())
	if license.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}

	ups := make(map[string]interface{})
	ups["status"] = reviewRet
	if err := licenseService.Update(license, ups).Error; err != nil {
		return nil, UpdateDbFailed.Error()
	}

	var review, handle, emailHtml, notifyValue string
	licenseAddTime := time.Unix(license.CreateAt, 0).Format(timeLayout)
	switch reviewRet {
	case 2:
		review = "未通过"
		officialEmail := settingService.GetOfficialEmail()
		emailHtml = fmt.Sprintf(notifyContent.EmailVaspLicenseVerifyFailed,
			license.Id, license.Organization, licenseAddTime)
		handle = fmt.Sprintf("如有需要请重新申请，或邮件%s联系我们", officialEmail)
		notifyValue = fmt.Sprintf(notifyContent.LicenseVerifyFailed, license.Organization,
			"VASP license 绑定邮箱与提交邮箱不匹配。")
	case 3:
		review = "已通过"
		officialLink := settingService.GetOfficialLink()
		emailHtml = fmt.Sprintf(notifyContent.EmailVaspLicenseVerifySuccess,
			license.Id, license.Organization, licenseAddTime)
		handle = fmt.Sprintf("快去创建项目吧，官网：%s", officialLink)
		notifyValue = fmt.Sprintf(notifyContent.LicenseVerifySuccess, license.Organization)
	}

	user := userService.Get(license.UserId)
	if user.Id <= 0 {
		return &pb.ReviewVaspLicenseReply{Message: "ok"}, nil
	}

	nowTime := time.Now().Unix()
	notify := &model.Notify{
		NotifyType: 4,
		UserId:     user.Id,
		Title:      "VASP license",
		Content:    notifyValue,
		CreateAt:   nowTime,
		UpdateAt:   nowTime,
	}
	if err := notifyService.Create(notify).Error; err != nil {
		log.Error(err.Error())
	}

	// 短信通知
	templateParamSet := []*string{&license.Organization, &review, &handle}
	phone := fmt.Sprintf("+86 %s", user.Phone)
	smsTemplate := &sms.GetSms().VaspLicenseTemplateId
	sms.SendSms(&phone, smsTemplate, templateParamSet)

	// 邮件通知
	email := user.Email
	if len(email) > 0 {
		subject := "VASP license"
		ses.SendEmail(email, "", emailHtml, subject)
	}
	return &pb.ReviewVaspLicenseReply{Message: "ok"}, nil
}
