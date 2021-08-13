package auth

import (
	"TEFS-BE/pkg/admin/model"
	"TEFS-BE/pkg/cache"
	"TEFS-BE/pkg/log"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
	"math"
	"net/http"
	"time"
)

const tokenEffectiveTime = 60 * 60 * 24 * 7 // token有效期1周

var (
	// qq
	qqAuthAppId   string
	qqAuthAppKey  string
	qqRedirectUri string

	// 微信
	wxAppId       string
	wxAppSecret   string
	//wxRedirectUri string

	// jwt secret
	secret           string
	jwtTokenRedisKey = "auth.user.%d.jwtToken"

	userService = model.UserService{}
	settingService = model.SettingService{}
	experimentService = model.ExperimentService{}
	userProjectService = model.UserProjectService{}
)

// api 返回json结构体
type ApiJson struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// 初始化设置
func Setup() {
	qqAuthAppId = viper.GetString("auth.qq.appId")
	qqAuthAppKey = viper.GetString("auth.qq.appKey")
	qqRedirectUri = viper.GetString("auth.qq.redirectUri")
	secret = viper.GetString("auth.secret")
	wxAppId = viper.GetString("auth.wx.appId")
	wxAppSecret = viper.GetString("auth.wx.AppSecret")
	//wxRedirectUri = viper.GetString("auth.wx.redirectUri")
}

// 写入响应json
func WriteResponse(data interface{}, msg string, code int, w http.ResponseWriter) {
	ret := &ApiJson{}
	ret.Code = code
	ret.Data = data
	ret.Msg = msg
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	_ = jsonEncoder.Encode(ret)
	_, _ = w.Write(bf.Bytes())
}

// 创建token
func CreateJwtToken(nickname, figureUrl, account, way string, id int64) (string, error) {
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":         id,
		"account":    account,
		"way":        way,
		"nickname":   nickname,
		"figure_url": figureUrl,
		"exp":        time.Now().Add(tokenEffectiveTime * time.Second).Unix(),
	})
	token, err := at.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return token, nil
}

// 解析token
func ParseJwtToken(token string) (user *model.User, tokenClaims map[string]string, isExpired bool, err error) {
	claim, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	var tokenIsInvalid bool
	if err != nil {
		// token过期
		isExpired = true
		if err.Error() == "Token is expired" {
			tokenIsInvalid = false
		} else {
			tokenIsInvalid = true
		}
	}
	if tokenIsInvalid || claim == nil {
		return nil, nil, isExpired, fmt.Errorf("invalid token")
	}
	jwtClaims := claim.Claims.(jwt.MapClaims)
	userId := int64(jwtClaims["id"].(float64) * math.Pow10(0))

	var userInfo = make(map[string]string)
	userInfo["account"] = jwtClaims["account"].(string)
	userInfo["way"] = jwtClaims["way"].(string)
	userInfo["nickname"] = jwtClaims["nickname"].(string)
	userInfo["figure_url"] = jwtClaims["figure_url"].(string)
	user = userService.Get(userId)
	return user, userInfo, isExpired, nil
}

// 获取token在redis的key
func GetTokenRedisKey(userId int64) string {
	return fmt.Sprintf(jwtTokenRedisKey, userId)
}

// 保存token到redis
func SaveToken(userId int64, jwtToken string) error {
	redisCli := cache.GetRedis()
	key := GetTokenRedisKey(userId)
	return redisCli.Set(key, jwtToken, tokenEffectiveTime*time.Second).Err()
}

// 刷新token在redis剩余时间
func RefreshTokenTime(userId int64) error {
	redisCli := cache.GetRedis()
	key := GetTokenRedisKey(userId)
	return redisCli.Expire(key, tokenEffectiveTime*time.Second).Err()
}

// 删除登录token
func DelToken(userId int64) (int64, error) {
	redisCli := cache.GetRedis()
	key := GetTokenRedisKey(userId)
	return redisCli.Del(key).Result()
}

// qq 微信登录授权成功后的处理
func dbHandle(account, way, nickName, headPortrait, shareKey string, w http.ResponseWriter) {
	data := make(map[string]interface{})
	// db 查询user
	user := userService.GetUserByAccount(account)
	data["account"] = account
	data["way"] = way
	if user.Id <= 0 {
		data["is_register"] = false
		user.Account = account
		user.Name = nickName
		user.Way = way
		nowTime := time.Now().Unix()
		user.CreateAt = nowTime
		user.UpdateAt = nowTime
		user.IsNotify = 1
		if err := userService.Create(user).Error; err != nil {
			log.Error(err.Error())
			WriteResponse("", "create user failed", 1003, w)
			return
		}
	} else {
		ups := make(map[string]interface{})
		ups["last_login_at"] = time.Now().Unix()
		if len(user.Email) > 0 && len(user.Phone) > 0 {
			data["is_register"] = true
		} else {
			data["is_register"] = false
		}
		if len(user.Name) == 0 {
			ups["name"] = nickName
		}

		// 更新用户信息
		userService.Update(user, ups)
	}
	if settingService.UserIsAdmin(user) {
		data["is_admin"] = true
	} else {
		data["is_admin"] = false
	}

	// 生成jwt
	jwtToken, err := CreateJwtToken(nickName, headPortrait, account, way, user.Id)
	if err != nil {
		WriteResponse("", "create token failed", 1004, w)
		return
	}
	// redis保存token
	if err := SaveToken(user.Id, jwtToken); err != nil {
		WriteResponse("", "save token failed", 1003, w)
		return
	}
	data["token"] = jwtToken
	data["id"] = user.Id
	data["agreement_version"] = user.AgreementVersion
	// 获取用户最近实验项目id
	project := experimentService.GetLastUpdateRecord(user.Id)
	if project.Id <= 0 {
		project = userProjectService.GetLastUserProject(user.Id)
	}
	if project.Id > 0 {
		projectByte, err := json.Marshal(project)
		if err != nil {
			log.Error(err.Error())
		} else {
			data["project"] = string(projectByte)
		}
	}
	// 文档分享的key
	data["shareKey"] = shareKey
	WriteResponse(data, "ok", 0, w)
}