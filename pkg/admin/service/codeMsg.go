package service

import "fmt"

// grpc 错误代码和消息
type ErrCodeMsg struct {
	Code int
	Msg  string
}

// grpc 错误
func (cm ErrCodeMsg) Error() error {
	return fmt.Errorf(fmt.Sprintf("code:%d,msg:%s", cm.Code, cm.Msg))
}

// 参数错误
func (cm ErrCodeMsg) ErrorParam(paramKey, tips string) error {
	msg := fmt.Sprintf("msg:invalid params %s", paramKey)
	if len(tips) > 0 {
		msg += ", "
		msg += tips
	}
	return fmt.Errorf(fmt.Sprintf("code:%d,msg:%s", cm.Code, msg))
}

// 记录错误，找不到记录
func (cm ErrCodeMsg) ErrorRecord(recordType string) error {
	return fmt.Errorf(fmt.Sprintf("code:%d,msg:not found record %s", cm.Code, recordType))
}

// 错误作为消息
func (cm ErrCodeMsg) ErrorErr(err error) error {
	return fmt.Errorf(fmt.Sprintf("code:%d,msg:%s", cm.Code, err.Error()))
}

var (
	// 数据库id无效
	InvalidId = &ErrCodeMsg{1001, "invalid id"}

	// 找不到记录
	NotFoundRecord = &ErrCodeMsg{1002, "not found record"}

	// 数据库保存失败
	DbSaveFailed = &ErrCodeMsg{1003, "db save failed"}

	// 没有权限
	NoAuthority = &ErrCodeMsg{1004, "no authority"}



	// 获取用户服务客户端失败
	GetClientError = &ErrCodeMsg{1005, "get service client failed"}

	// 请求失败
	RequestError = &ErrCodeMsg{1006, "request service failed"}

	// 更新数据库失败
	UpdateDbFailed = &ErrCodeMsg{1007, "update db failed"}

	// 发送邮件失败
	SendEmailFailed = &ErrCodeMsg{1008, "send email failed"}

	// 以错误信息返回
	ErrorErr = &ErrCodeMsg{1009, ""}

	// 查询数据失败
	QueryDbFailed = &ErrCodeMsg{1010, "query db failed"}

	// 创建记录失败
	CreateRecordFailed = &ErrCodeMsg{1011, "create db record failed"}

	// 获取cos临时密钥失败
	GetCosTmpSercretFailed = &ErrCodeMsg{1012, "get cos tmp sercret failed"}

	// 获取用户实验室错误
	GetLabCliErr = &ErrCodeMsg{1013, "get laboratory client failed"}

	// 实验计算中
	ExperimentComputing = &ErrCodeMsg{1014, "experiment is computing"}

	// 获取zone失败
	GetZoneFailed = &ErrCodeMsg{1015, "get zone failed"}

	// 获取CVM信息失败
	GetCvmFailed = &ErrCodeMsg{1016, "get cvm failed"}

	// 为找到可用zone
	NotFoundAvailableZone = &ErrCodeMsg{1017, "no found available cvm zone"}

	// 未找到计算cvm镜像
	NotFoundComputeImage = &ErrCodeMsg{1018, "not found compute image"}

	// 获取用户腾讯云账户计算镜像失败
	GetLabCvmImageFailed = &ErrCodeMsg{1019, "get lab cmv image failed"}

	// 共享计算镜像失败
	ShareComputeImageFailed = &ErrCodeMsg{1020, "share compute image failed"}

	// 查询用户batch计算环境失败
	QueryBatchEnvFailed = &ErrCodeMsg{1021, "query batch env list failed"}

	// 用户腾讯云账户batch计算环境数量限制
	BatchEnvLimit = &ErrCodeMsg{1022, "batch env limit"}

	// 用户腾讯云账户batch计算环境创建失败
	BatchEnvCreateFailed = &ErrCodeMsg{1023, "batch env create failed"}

	// 发送实验到消费者队列失败
	SendExToConsumerFailed = &ErrCodeMsg{1024, "send experiment to consumer failed"}

	// 请稍后再试
	TryAgainLater = &ErrCodeMsg{1025, "try again later"} // 稍后在试

	// 实验不在计算中
	ExIsNotComputing = &ErrCodeMsg{1026, "experiment is not computing"}

	// 实验还在终止中
	ExIsTerminating = &ErrCodeMsg{1027, "experiment is terminating"}

	// 获取cos客户端失败
	GetCosCliFailed = &ErrCodeMsg{1028, "get cos client failed"}

	// 删除实验cos目录失败
	DelExCosDirFailed = &ErrCodeMsg{1029, "delete experiment cos dir failed"}

	// 数据对象转换为json错误
	DataToJsonErr = &ErrCodeMsg{1030, "data to json failed"}

	// 通知记录类型错误
	NotifyTypeErr = &ErrCodeMsg{1031, "notify type not system notify"}

	// 腾讯云账户以绑定到其他的vaspLicense
	CloudIdIsBind = &ErrCodeMsg{1032, "cloud secret id used by other vasplicense"}

	// 名字重复
	NameRepeat = &ErrCodeMsg{1033, "name repeat"}

	// 创建项目邀请码失败
	CreateInvitationCodeFailed = &ErrCodeMsg{1034, "create invitation code failed"}

	// 用户在项目中被禁用
	UserDisabled = &ErrCodeMsg{1035, "user disabled"}

	// 申请的项目已经申请过了
	NearApplication = &ErrCodeMsg{1036, "the project near application"}

	// 记录已经操作过了
	RecordIsUsed = &ErrCodeMsg{1037, "record is used"}

	// 项目中未找到用户
	ProjectNotFoundUser = &ErrCodeMsg{1038, "project nof found user"}

	// 用户已存在项目中
	UserAlreadyExists = &ErrCodeMsg{1039, "the user already exists in the project"}

	// 短信发送限制
	SmsSendRestrict = &ErrCodeMsg{1040, "SMS send restrict"}

	// 短信内容存储失败
	StorageSmsFailed = &ErrCodeMsg{1041, "storage sms failed"}

	// 功能不可用
	FeatureNotAvailable = &ErrCodeMsg{1042, "feature not available"}

	// 总量限制
	TotalLimit = &ErrCodeMsg{1043, "total limit"}

	// tag name 重复
	DuplicateTagName = &ErrCodeMsg{1044, "duplicate tag name"}

	// 设置Vasp license COS目录失败
	SetVaspLicenseCosDirFailed = &ErrCodeMsg{1045, "set vasp license dir number failed"}

	// redis设置锁失败
	SetLockFailed = &ErrCodeMsg{1046, "cache db error"}

	// 提交错误
	SubmitErr = &ErrCodeMsg{1047, "please do not submit again"}

	// vasp license 目录数错误
	vaspLicenseDirNumErr = &ErrCodeMsg{1048, "get vaspLicenseDirNum failed"}

	// 查询redis失败
	RedisGetFailed = &ErrCodeMsg{1049, "cache get failed"}

	// 为上传图片
	NotUploadImage = &ErrCodeMsg{1050, "not upload image file"}

	// vasp license 只能在未审核通过情况下删除
	LicenseCannotDeleted = &ErrCodeMsg{1051, "the approved license cannot be deleted"}

	// cos下载失败
	GetCosObjectFailed = &ErrCodeMsg{1052, "get cos objetts failed"}

	// 获取redis锁超时
	GetLockTimeOut = &ErrCodeMsg{1052, "get lock timeout"}

	// vasplicense被禁用，请联系创建人重新提交资料
	VaspDisable = &ErrCodeMsg{
		1053,
		"vasp license disabled, please contact the creator to submit the information again",
	}

	// 无效参数
	InvalidParams = &ErrCodeMsg{1054, "invalid params"}

	// 用户腾讯云环境创建失败
	CloudEnvNotCreate = &ErrCodeMsg{1055, "cloudEnv not create done"}

	// 收件邮箱格式错误
	EmailFormatError = &ErrCodeMsg{1056, "to email format error"}

	// copy实验只能在一个项目内
	CopyExErr = &ErrCodeMsg{1057, "you can only copy experiments in the same project"}

	// 链接失败
	ConnErr = &ErrCodeMsg{1058, "link vaspkit tefs_file_server failure"}

	// 链接失败
	MissParam = &ErrCodeMsg{1059, "miss param"}

	// to json err
	ToJsonFailed = &ErrCodeMsg{1060, "data to json failed"}

	// 存在子实验
	ExistSonExperiment = &ErrCodeMsg{1061, "exist son experiment"}
	// 验证码错误
	VerificationCodeError = &ErrCodeMsg{1060, "verification code error"}

	// 获取cos文件失败
	GetCosFileFailed = &ErrCodeMsg{1061, "get cos file error"}

	// 实验里不存在数据文件
	ExpDataNotFound = &ErrCodeMsg{1062, "experiment data not found"}

	// 获取数据失败
	GetDataFailed = &ErrCodeMsg{1063, "get data failed"}

	// getEnergy参数错误
	ErrKey = &ErrCodeMsg{1064, "param key error"}

	// 参数错误
	ErrDataType = &ErrCodeMsg{1064, "param dataType error"}

	// 解析文件错误
	ParseXmlFailed = &ErrCodeMsg{1065, "parse vasprun.xml failed"}

	// 创建路径错误
	PathErr = &ErrCodeMsg{1066, "mkdir error"}

	// 用户不匹配
	ErrUser = &ErrCodeMsg{1067, "user not match"}

	// 登录验证失败,code为 2xxx
	// 获取token失败
	GetTokenFailed = &ErrCodeMsg{2001, "not login:get token failed"}

	// token解析失败
	ParseTokenFailed = &ErrCodeMsg{2002, "not login:parse token failed"}

	// 找不到用户
	NotFoundUser = &ErrCodeMsg{2003, "not login:not found user"}

	// 获取token缓存失败
	GetStorageTokenFailed = &ErrCodeMsg{2004, "not login:get storage token failed"}

	// token过期
	InvalidToken = &ErrCodeMsg{2005, "not login:invalid token"}

	// 刷新token失败
	RefreshToken1 = &ErrCodeMsg{2006, "not login:refresh token create failed"}

	RefreshToken2 = &ErrCodeMsg{2007, "not login:refresh token save failed"}
)
