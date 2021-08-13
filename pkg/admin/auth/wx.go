package auth

import (
	"TEFS-BE/pkg/log"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

var (
	//wxLoginUrl       = "https://open.weixin.qq.com/connect/qrconnect"
	wxAccessTokenUrl = "https://api.weixin.qq.com/sns/oauth2/access_token"
	wxUserInfoUrl    = "https://api.weixin.qq.com/sns/userinfo"
)

type BaseResponse struct {
	// 错误代码
	ErrCode int64 `json:"errcode,omitempty"`
	// 错误消息
	ErrMsg string `json:"errmsg,omitempty"`
}

type wxAccessTokenResponse struct {
	BaseResponse

	// 接口调用凭证
	AccessToken string `json:"access_token,omitempty"`

	// 	access_token接口调用凭证超时时间，单位（秒）
	ExpiresIn int64 `json:"expires_in,omitempty"`

	// 用户刷新access_token
	RefreshToken string `json:"refresh_token,omitempty"`

	// 授权用户唯一标识
	OpenId string `json:"openid,omitempty"`

	// 用户授权的作用域，使用逗号（,）分隔
	Scope string `json:"scope,omitempty"`

	// 当且仅当该网站应用已获得该用户的userinfo授权时，才会出现该字段。
	UnionId string `json:"unionid,omitempty"`
}

type wxUserInfoResponse struct {
	BaseResponse

	// 普通用户的标识，对当前开发者帐号唯一
	OpenId string `json:"openid,omitempty"`

	// 普通用户昵称
	Nickname string `json:"nickname,omitempty"`

	// 普通用户性别，1为男性，2为女性
	Sex int `json:"sex,omitempty"`

	// 普通用户个人资料填写的省份
	Province string `json:"province,omitempty"`

	// 普通用户个人资料填写的城市
	City string `json:"city,omitempty"`

	// 国家，如中国为CN
	Country string `json:"country,omitempty"`

	// 用户头像，最后一个数值代表正方形头像大小（有0、46、64、96、132数值可选，0代表640*640正方形头像），用户没有头像时该项为空
	Headimgurl string `json:"headimgurl,omitempty"`

	// 用户特权信息，json数组，如微信沃卡用户为（chinaunicom）
	Privilege []string `json:"privilege,omitempty"`

	// 用户统一标识。针对一个微信开放平台帐号下的应用，同一用户的unionid是唯一的。
	Unionid string `json:"unionid,omitempty"`
}

//func genderStateParam() string {
//	return "test"
//}
//
//func WxLogin(w http.ResponseWriter, r *http.Request) {
//	params := url.Values{}
//	params.Add("appid", wxAppId)
//	params.Add("redirect_uri", wxRedirectUri)
//	params.Add("response_type", "code")
//	params.Add("scope", "snsapi_login")
//	params.Add("state", genderStateParam())
//
//	loginUrl := fmt.Sprintf("%s?%s#wechat_redirect", wxLoginUrl, params.Encode())
//	http.Redirect(w, r, loginUrl, http.StatusFound)
//}

func WxLoinCallBack(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	// 文档分享的key
	shareKey := r.FormValue("key")

	accessTokenResponse := &wxAccessTokenResponse{}
	// 获取微信access_token
	if err := getWxAccessToken(code, accessTokenResponse); err != nil {
		log.Error(err.Error())
		WriteResponse("", err.Error(), 1001, w)
		return
	}

	// 获取用户信息
	accessToken := accessTokenResponse.AccessToken
	openId := accessTokenResponse.OpenId
	userInfo := &wxUserInfoResponse{}
	if err := getWxUserInfo(accessToken, openId, userInfo); err != nil {
		log.Error(err.Error())
		WriteResponse("", err.Error(), 1001, w)
		return
	}

	// 注册
	dbHandle(userInfo.Unionid, "wx", userInfo.Nickname, userInfo.Headimgurl, shareKey, w)
}

func getWxAccessToken(code string, accessToken *wxAccessTokenResponse) error {
	params := url.Values{}
	params.Add("appid", wxAppId)
	params.Add("secret", wxAppSecret)
	params.Add("code", code)
	params.Add("grant_type", "authorization_code")

	url := fmt.Sprintf("%s?%s", wxAccessTokenUrl, params.Encode())
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bs, accessToken); err != nil {
		return err
	}
	if len(accessToken.ErrMsg) != 0 || accessToken.ErrCode != 0 {
		return fmt.Errorf(string(bs))
	}
	return nil
}

func getWxUserInfo(accessToken, openid string, info *wxUserInfoResponse) error {
	params := url.Values{}
	params.Add("access_token", accessToken)
	params.Add("openid", openid)

	url := fmt.Sprintf("%s?%s", wxUserInfoUrl, params.Encode())
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bs, info); err != nil {
		return err
	}
	if len(info.ErrMsg) != 0 || info.ErrCode != 0 {
		return fmt.Errorf(string(bs))
	}
	return nil
}
