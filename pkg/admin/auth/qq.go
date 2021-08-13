package auth

import (
	"TEFS-BE/pkg/cache"
	"TEFS-BE/pkg/log"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type QQToken struct {
	AccessToken  string // 授权令牌
	ExpiresIn    int    // 该access token的有效期，单位为秒。
	RefreshToken string // 在授权自动续期步骤中，获取新的Access_Token时需要提供的参数。refresh_token仅一次有效
	OpenId       string // qq 唯一标识
}

type QQUser struct {
	Ret             int    `json:"ret"`
	Msg             string `json:"msg"`
	IsLost          int    `json:"is_lost"`
	Nickname        string `json:"nickname"`
	Gender          string `json:"gender"`
	GenderType      int    `json:"gender_type"`
	Province        string `json:"province"`
	City            string `json:"city"`
	Year            string `json:"year"`
	Constellation   string `json:"constellation"`
	Figureurl       string `json:"figureurl"` // 下面url全是头像链接。只是分辨率不同
	Figureurl1      string `json:"figureurl_1"`
	Figureurl2      string `json:"figureurl_2"`
	FigureurlQq1    string `json:"figureurl_qq_1"`
	FigureurlQq2    string `json:"figureurl_qq_2"`
	FigureurlQq     string `json:"figureurl_qq"`
	FigureurlType   string `json:"figureurl_type"`
	IsYellowVip     string `json:"is_yellow_vip"`
	Vip             string `json:"vip"`
	YellowVipLevel  string `json:"yellow_vip_level"`
	Level           string `json:"level"`
	IsYellowYearVip string `json:"is_yellow_year_vip"`
}

func QQLogin(w http.ResponseWriter, r *http.Request) {
	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", qqAuthAppId)
	params.Add("state", "test")
	str := fmt.Sprintf("%s&redirect_uri=%s", params.Encode(), qqRedirectUri)
	loginURL := fmt.Sprintf("%s?%s", "https://graph.qq.com/oauth2.0/authorize", str)
	http.Redirect(w, r, loginURL, http.StatusFound)
}

func QQLoginCallback(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	redirectUri := r.FormValue("redirect_uri")
	// 文档分享的key
	shareKey := r.FormValue("key")

	qqToken := &QQToken{}
	// 获取accessToken
	if err := GetQQToken(code, qqAuthAppId, qqAuthAppKey, redirectUri, qqToken); err != nil {
		log.Error(err.Error())
		WriteResponse("", err.Error(), 1001, w)
		return
	}
	// 获取openId
	if err := GetQQOpenId(qqToken); err != nil {
		log.Error(err.Error())
		WriteResponse("", err.Error(), 1002, w)
		return
	}
	// 获取用户qq信息
	qqUser, err := GetQQUser(qqToken)
	if err != nil {
		WriteResponse("", err.Error(), 1003, w)
		return
	}
	dbHandle(qqToken.OpenId, "qq", qqUser.Nickname, qqUser.Figureurl2, shareKey, w)
}

func GetQQToken(code, appId, appKey, redirectUri string, qqToken *QQToken) error {
	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("client_id", appId)
	params.Add("client_secret", appKey)
	params.Add("code", code)
	loginUrl := fmt.Sprintf("https://graph.qq.com/oauth2.0/token?%s&redirect_uri=%s", params.Encode(), redirectUri)
	resp, err := http.Get(loginUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(bs)
	re := regexp.MustCompile(`^access_token=.*&expires_in=.*&refresh_token=.*$`)
	if !re.MatchString(body) {
		return fmt.Errorf(body)
	}
	restMap := convertToMap(body)
	qqToken.AccessToken = restMap["access_token"]
	expiresIn, _ := strconv.Atoi(restMap["expires_in"])
	qqToken.ExpiresIn = expiresIn
	qqToken.RefreshToken = restMap["refresh_token"]
	return nil
}

func GetQQOpenId(qqToken *QQToken) error {
	accessToken := qqToken.AccessToken
	uri := fmt.Sprintf("https://graph.qq.com/oauth2.0/me?access_token=%s", accessToken)
	response, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	bs, _ := ioutil.ReadAll(response.Body)
	body := string(bs)
	openId := body[45:77]
	qqToken.OpenId = openId
	return nil
}

func GetQQUser(qqToken *QQToken) (*QQUser, error) {
	user := &QQUser{}

	// 缓存获取用户qq信息
	redisClient := cache.GetRedis()
	key := qqToken.OpenId + ".userInfo"
	userByte, err := redisClient.Get(key).Bytes()
	if err == nil {
		_ = json.Unmarshal(userByte, &user)
		return user, nil
	}

	// 获取失败，请求qqAPI获取,数据缓存到redis
	params := url.Values{}
	params.Add("access_token", qqToken.AccessToken)
	params.Add("openid", qqToken.OpenId)
	params.Add("oauth_consumer_key", qqAuthAppId)
	sendUrl := fmt.Sprintf("https://graph.qq.com/user/get_user_info?%s", params.Encode())
	response, err := http.Get(sendUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	userByte, _ = ioutil.ReadAll(response.Body)
	err = json.Unmarshal(userByte, &user)
	if err != nil || user.Ret != 0 {
		return nil, fmt.Errorf("get qq user failed")
	}
	redisClient.SetNX(key, userByte, time.Hour*24)
	return user, nil
}

func convertToMap(str string) map[string]string {
	var resultMap = make(map[string]string)
	values := strings.Split(str, "&")
	for _, value := range values {
		vs := strings.Split(value, "=")
		resultMap[vs[0]] = vs[1]
	}
	return resultMap
}
