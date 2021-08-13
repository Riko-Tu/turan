package tencentCloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

var domain = "account.api.qcloud.com/v2/index.php"

// 腾讯云账户
type Account struct {
	Credential *Credential
	Region     string
}

// 基础响应体
type BaseResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	CodeDesc string `json:"codeDesc,omitempty"`
}

// 获取项目信息响应
type GetProjectInfoResponse struct {
	BaseResponse
	Data []Project `json:"data,omitempty"`
}

// 创建项目响应体
type CreateProjectResponse struct {
	BaseResponse
	ProjectId int `json:"projectId,omitempty"`
}

// 项目结构体
type Project struct {
	ProjectName string `json:"projectName"`
	ProjectId   int  `json:"projectId"`
	CreateTime  string `json:"createTime"`
	CreatorUin  int  `json:"creatorUin"`
	ProjectInfo string `json:"projectInfo"`
}

// 获取签名字符串
// todo:(v_vwwwang)函数单元测试
func getStringToSign(method, domain, path string, params map[string]string) string {
	var buf bytes.Buffer
	buf.WriteString(method)
	buf.WriteString(domain)
	buf.WriteString(path)
	buf.WriteString("?")

	// sort params
	keys := make([]string, 0, len(params))
	for k := range params {
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

func (a Account) request(params map[string]string, action string) ([]byte, error) {
	params["Region"] = a.Region
	params["SignatureMethod"] = "HmacSHA256"
	params["Nonce"] = strconv.Itoa(rand.Int())[:5]
	params["Action"] = action
	params["SecretId"] = a.Credential.SecretId
	params["Timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	HttpMethod := "GET"
	s := getStringToSign(HttpMethod, domain, "", params)
	signStr := common.Sign(s, a.Credential.SecretKey, "HmacSHA256")
	signature := url.QueryEscape(signStr)
	requestUrl := "https://" + s[3:] + "&Signature=" + signature
	resp, err := http.Get(requestUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// 创建项目
func (a Account) CreateProject(projectName, projectDesc string) (projectId int, err error) {
	params := make(map[string]string)
	params["projectName"] = projectName
	params["projectDesc"] = projectDesc

	action := "AddProject"
	respBody, err := a.request(params, action)
	if err != nil {
		return 0, err
	}
	createProjectResponse := &CreateProjectResponse{}
	if err := json.Unmarshal(respBody, createProjectResponse); err != nil {
		return 0, err
	}
	if createProjectResponse.Code > 0 {
		return 0, fmt.Errorf(string(respBody))
	}
	return createProjectResponse.ProjectId, nil
}

// 查询项目列表,可用于获取腾讯云主id
func (a Account) GetProjects() ([]Project, error) {
	params := make(map[string]string)
	action := "DescribeProject"
	respBody, err := a.request(params, action)
	if err != nil {
		return nil, err
	}
	projectInfoResponse := &GetProjectInfoResponse{}
	if err := json.Unmarshal(respBody, projectInfoResponse); err != nil {
		return nil, err
	}
	if projectInfoResponse.Code != 0 {
		return nil, fmt.Errorf(string(respBody))
	}
	return projectInfoResponse.Data, nil
}
