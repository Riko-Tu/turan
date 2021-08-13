package server

import (
	"TEFS-BE/pkg/admin/model"
	"TEFS-BE/pkg/log"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strconv"
)


// 对参数进行签名
func Sign(s, secret string) string {
	hashed := hmac.New(sha256.New, []byte(secret))
	hashed.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(hashed.Sum(nil))
}

// 连接参数,进行排序,用来进行签名
func JoinParamsToSign(params map[string]string) string {
	var buf bytes.Buffer
	keys := make([]string, 0, len(params))
	for k, _ := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := range keys {
		k := keys[i]
		if params[k] == "" {
			continue
		}
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(params[k])
		buf.WriteString("&")
	}
	buf.Truncate(buf.Len() - 1)
	return buf.String()
}

func GetParamsMap(r *http.Request) (map[string]string, error) {
	params := make(map[string]string)
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	for k, v := range r.Form {
		if len(v) == 0 {
			params[k] = ""
		} else {
			params[k] = v[0]
		}
	}
	return params, nil
}

// 获取id参数
func getIdFromParams(params map[string]string, key string) (int64, error) {
	projectIdStr, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("not found params %s", key)
	}
	targetId, err := strconv.ParseInt(projectIdStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("params %s type err, %s is not int", key, key)
	}
	return targetId, nil
}

// 验证签名
func VerifySign(params map[string]string, secret, endpoint string) bool {
	reqSign, ok := params["sign"]
	if !ok {
		return false
	}
	delete(params, "sign")

	paramsJoin := JoinParamsToSign(params)
	s := endpoint + "?" + paramsJoin
	sign := Sign(s, secret)
	if sign != reqSign {
		return false
	}
	return true
}

// shell console 请求通用验证
// 签名验证
func GetRequestParams(r *http.Request) (params map[string]string, userId, projectId int64, err error) {
	// 获取传入参数
	params, err = GetParamsMap(r)
	if err != nil {
		return
	}

	// 获取项目id
	projectId, err = getIdFromParams(params, "project_id")
	if err != nil {
		return
	}

	// 获取userId
	userId, err = getIdFromParams(params, "user_id")
	if err != nil {
		return
	}

	// 获取数据库存储的secret
	shell := model.ConsoleShell{}
	if err = shell.Get(userId, projectId); err != nil {
		log.Error(err.Error())
		err = fmt.Errorf("query db field")
		return
	}

	// 验证签名
	if !VerifySign(params, shell.Secret, r.URL.Path) {
		err = fmt.Errorf("sign error")
		return
	}
	return
}