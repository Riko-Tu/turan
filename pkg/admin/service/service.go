package service

import (
	"TEFS-BE/pkg/admin/auth"
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/cache"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/tencentCloud"
	"TEFS-BE/pkg/tencentCloud/sms"
	"context"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

var (
	//phoneRe = regexp.MustCompile(`^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$`)
	phoneRe            = regexp.MustCompile(`^1\d{10}$`)
	userNameRe         = regexp.MustCompile("^[a-zA-Z\u4e00-\u9fcc]{2,30}$")
	emailRe            = regexp.MustCompile(`\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`)
	//smsCodeRe          = regexp.MustCompile(`^[0-9]{6}$`)
	cloudSecretIdRe    = regexp.MustCompile(`^[a-zA-z0-9]{36}$`)
	cloudAppIdRe       = regexp.MustCompile(`^\d{10}$`)
	agreementVersionRe = regexp.MustCompile(`^\d+(\.\d+)+$`)

	cosService                  tencentCloud.Cos
	cvmService                  tencentCloud.Cvm
	tefsDeployCosPathDir        string
	tefsPotcarCosPathDir		string
	tefsDemoExPathDir		    string
	tefsDeployForWindowsCosPath string
	tefsDeployForMacCosPath     string
	rsaPrivateKey               string
	region 						string
	bucket 						string
	rsaBegin                    = "-----BEGIN RSA PRIVATE KEY-----"
	rsaEnd                      = "-----END RSA PRIVATE KEY-----"
	timeLayout                  = "2006-01-02 15:04:05"

	chartDataFilePathDir 		string
	chartDemoExPathDir			string
	chartDemoExBucket			string
	chartDemoExRegion			string
	tencentCloudAppID			string
	tencentCloudSecretID		string
	tencentCloudSecretKey		string
	tmpChartDataFilePathDir  	string
)

func Setup() {

	region = viper.GetString("tencentCloud.region")
	bucket = viper.GetString("tencentCloud.cos.bucket")
	cosService = tencentCloud.Cos{
		Credential: &tencentCloud.Credential{
			AppId:     viper.GetString("tencentCloud.appId"),
			SecretId:  viper.GetString("tencentCloud.secretId"),
			SecretKey: viper.GetString("tencentCloud.secretKey"),
		},
		Region: region,
		Bucket: bucket,
	}

	cvmService = tencentCloud.Cvm{
		Credential: &tencentCloud.Credential{
			AppId:     viper.GetString("tencentCloud.appId"),
			SecretId:  viper.GetString("tencentCloud.secretId"),
			SecretKey: viper.GetString("tencentCloud.secretKey"),
		},
		Region: viper.GetString("tencentCloud.region"),
	}
	rsaPrivateKey = viper.GetString("privateKey")
	rsaPrivateKey = strings.Split(rsaPrivateKey, rsaBegin)[1]
	rsaPrivateKey = strings.Split(rsaPrivateKey, rsaEnd)[0]
	rsaPrivateKey = strings.ReplaceAll(rsaPrivateKey, " ", "\n")
	rsaPrivateKey = rsaBegin + rsaPrivateKey + rsaEnd

	tefsDeployCosPathDir = viper.GetString("tencentCloud.cos.tefsDeployCosPathDir")
	tefsPotcarCosPathDir = viper.GetString("tencentCloud.cos.tefsPotcarCosPathDir")
	tefsDemoExPathDir = viper.GetString("tencentCloud.cos.tefsDemoExPathDir")
	tefsDeployForWindowsCosPath = viper.GetString("tencentCloud.cos.tefsDeployForWindowsCosPath")
	tefsDeployForMacCosPath = viper.GetString("tencentCloud.cos.tefsDeployForMacCosPath")
	if len(tefsDeployCosPathDir) == 0 && len(tefsDeployForWindowsCosPath) == 0 && len(tefsDeployForMacCosPath) ==0 {
		log.Fatal("config tefsDeployCosPath error")
	}

	tencentCloudAppID = viper.GetString("tencentCloud.appId")
	tencentCloudSecretID = viper.GetString("tencentCloud.secretId")
	tencentCloudSecretKey = viper.GetString("tencentCloud.secretKey")
	chartDataFilePathDir = viper.GetString("chart.dataFilePathDir")
	chartDemoExPathDir = viper.GetString("chart.tefsDemoExPathDir")
	chartDemoExBucket = viper.GetString("chart.bucket")
	chartDemoExRegion = viper.GetString("chart.region")
	tmpChartDataFilePathDir = viper.GetString("chart.tmpDataFilePathDir")
}

type Service struct {
}

// 生成短信验证码
func GenValidateCode(width int) string {
	numeric := [10]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	r := len(numeric)
	rand.Seed(time.Now().UnixNano())

	var sb strings.Builder
	for i := 0; i < width; i++ {
		fmt.Fprintf(&sb, "%d", numeric[ rand.Intn(r) ])
	}
	return sb.String()
}

// redis获取验证手机验证码
func RedisGetSmsCode(redisClient *redis.Client, key string) (smsCode string, err error) {
	smsCode, err = redisClient.Get(key).Result()
	if err == cache.RedisNilError {
		return "", nil
	}
	if err != nil {
		log.Error(err.Error())
		return "", fmt.Errorf("query failed")
	}
	return smsCode, nil
}

// 发送短信验证码
func (s Service) SendSms(ctx context.Context, in *pb.SendSmsRequest) (*pb.SendSmsReply, error) {
	// 手机号码格式验证
	if ok := phoneRe.MatchString(in.GetPhone()); !ok {
		return nil, InvalidParams.ErrorParam("phone", "format error")
	}

	// 发送短信类型
	var smsTemplate *string
	switch in.GetType() {
	case 1:
		smsTemplate = &sms.GetSms().RegisterTemplateId
	case 2:
		smsTemplate = &sms.GetSms().EditUserTemplateId
	default:
		return nil, InvalidParams.ErrorParam("type", "type in 1,2")
	}

	// 生成短信验证码
	validateCode := GenValidateCode(6)

	// redis存储
	redisClient := cache.GetRedis()
	key := fmt.Sprintf("%s_%d", in.GetPhone(), in.GetType())
	// 1.查询短信是否已发送
	storageSmsCode, err := RedisGetSmsCode(redisClient, key)
	if err != nil {
		return nil, ErrorErr.ErrorErr(err)
	}
	if len(storageSmsCode) != 0 {
		return nil, SmsSendRestrict.Error()
	}
	// 2.redis设置lock，时间2分钟,2分钟内无法再次发送
	if err := redisClient.SetNX(key, validateCode, time.Second*60*2).Err(); err != nil {
		log.Error(err.Error())
		return nil, StorageSmsFailed.Error()
	}

	// 发送短信失败，删除redis lock
	sendSuccess := false
	defer func() {
		if !sendSuccess {
			redisClient.Del(key)
		}
	}()

	// 发送短信
	phone := "+86" + in.GetPhone()
	effectiveTime := "2" // 用户实际收到短信的有效时间提示，单位分钟
	templateParamSet := []*string{&validateCode, &effectiveTime}
	if sms.SendSms(&phone, smsTemplate, templateParamSet) {
		sendSuccess = true
	} else {
		return nil, FeatureNotAvailable.Error()
	}
	return &pb.SendSmsReply{Message: "send success"}, nil
}

// 获取cos下载临时密钥
func (s Service) GetDownloadTmpCredential(ctx context.Context,
	in *pb.GetDownloadTmpCredentialRequest) (*pb.GetDownloadTmpCredentialReply, error) {

	user := ctx.Value("user").(*model.User)
	var cosPaths []string
	var cosApis []string
	if settingService.UserIsAdmin(user) {
		cosPaths = []string{"/*"}
	} else {
		opType := in.GetOpType()
		switch opType {
		case 1:
			cosPaths = []string{fmt.Sprintf("/user_%d/*", user.Id)}
			cosApis = []string{"name/cos:GetObject", "name/cos:GetBucket"}
		case 2:
			cosApis = []string{"name/cos:GetObject", "name/cos:GetBucket"}
			cosPaths = []string{tefsDeployCosPathDir, tefsPotcarCosPathDir, tefsDemoExPathDir}
		case 3:
			fmt.Println(1234)
			cosApis = []string{
				"name/cos:PostObject",
				"name/cos:PutObject",
				"name/cos:DeleteObject",
				"name/cos:HeadObject",
				"name/cos:GetObject",
				"name/cos:GetBucket",
			}
			cosPaths = []string{fmt.Sprintf("/user_%d/Echar/*", user.Id)}
		default:
			return nil, InvalidParams.ErrorParam("opType", "opType in 1 2 3")
		}
	}
	secret, err := cosService.GetTmpSecret(cosApis, cosService.JoinQcsCosPath(cosPaths))
	if err != nil {
		return nil, err
	}
	reply := &pb.GetDownloadTmpCredentialReply{
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
	}
	if in.GetOpType() == 2 {
		tefsDeployCosPath := &pb.TefsDeployCosPath{
			Windows: tefsDeployForWindowsCosPath,
			Mac:     tefsDeployForMacCosPath,
		}
		reply.TefsDeployCosPath = tefsDeployCosPath
	}
	return reply, nil
}

// 登出
func (s Service) LoginOut(ctx context.Context, in *pb.LoginOutRequest) (*pb.LoginOutReply, error) {
	user := ctx.Value("user").(*model.User)
	_, err := auth.DelToken(user.Id)
	if err != nil {
		log.Error(err.Error())
	}
	return &pb.LoginOutReply{Message: "ok"}, nil
}

// 处理token
func HandleToken(token string) (user *model.User, newToken string, err error) {

	if strings.HasPrefix(token, "Bearer ") {
		s := strings.Split(token, "Bearer ")
		if len(s) != 2 {
			err = ParseTokenFailed.Error()
			return
		}
		token = s[1]
	}

	user, tokenClaims, isExpired, err := auth.ParseJwtToken(token)
	if err != nil {
		log.Error(err.Error())
		err = ParseTokenFailed.Error()
		return
	}
	if user.Id <= 0 {
		err = NotFoundUser.Error()
		return
	}

	// redis token json
	redisCli := cache.GetRedis()
	tokenRedisKey := auth.GetTokenRedisKey(user.Id)
	oldTokenRedisKey := tokenRedisKey + ".old"
	storageTokens, err := redisCli.MGet(tokenRedisKey, oldTokenRedisKey).Result()
	if err != nil {
		log.Error(err.Error())
		err = GetStorageTokenFailed.Error()
		return
	}
	curStorageToken := storageTokens[0]
	oldStorageToken := storageTokens[1]

	// 旧token
	if oldStorageToken != nil && token == oldStorageToken.(string) {
		return
	}

	// redis未找到token
	if curStorageToken == nil || curStorageToken.(string) != token  {
		err = InvalidToken.Error()
		return
	}

	// token未过期，刷新redis存在时间
	if !isExpired {
		if refreshErr := auth.RefreshTokenTime(user.Id); refreshErr != nil {
			log.Error(refreshErr.Error())
		}
		return
	}

	// token过期，redis存储token未过期，redis设置一个旧token
	ok, setErr := redisCli.SetNX(oldTokenRedisKey, token, time.Second * 30).Result()
	if setErr != nil {
		log.Error(setErr.Error())
	}
	if ok {
		newToken, err = auth.CreateJwtToken(tokenClaims["nickname"], tokenClaims["figure_url"],
			tokenClaims["account"], tokenClaims["way"], user.Id)
		if err != nil {
			log.Error(err.Error())
			err = RefreshToken1.Error()
		}
		if err := auth.SaveToken(user.Id, newToken); err != nil {
			log.Error(err.Error())
			err = RefreshToken2.Error()
		}
	}
	return
}