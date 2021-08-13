package ses

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var sesCli *Ses

const (
	sesSendApi         = "https://ses.myqcloud.com/Qses/prod/api/send"
	sesTemplateSendAPI = "https://ses.myqcloud.com/Qses/prod/api/templateSend"
	sesQueryAPI        = "https://ses.myqcloud.com/Qses/prod/api/query"
)

func Setup() {
	sesCli = &Ses{
		User:     viper.GetString("tencentCloud.ses.user"),
		Password: viper.GetString("tencentCloud.ses.password"),
		Email:    viper.GetString("tencentCloud.ses.email"),
	}
}

func GetSendEmail() string {
	return sesCli.Email
}

func (s Ses) getAuthorization() string {
	input := []byte(fmt.Sprintf("%s:%s", s.User, s.Password))
	authorization := "Basic " + base64.StdEncoding.EncodeToString(input)
	return authorization
}

func (s Ses) request(method, url, contentType string, body io.Reader, response interface{}) error {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", s.getAuthorization())
	if len(contentType) > 0 {
		request.Header.Add("Content-Type", contentType)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(result, response); err != nil {
		return err
	}
	return nil
}

func Send(params *SendParams) (*SendResponse, error) {
	// 附件,可多个
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if len(params.Attachment) > 0 {
		for _, path := range params.Attachment {
			file, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			defer file.Close()
			strItmes := strings.Split(path, "/")
			fileName := strItmes[len(strItmes)-1]
			part, err := writer.CreateFormFile(fileName, filepath.Base(path))
			if err != nil {
				return nil, err
			}
			_, _ = io.Copy(part, file)
		}
	}

	// 设置参数
	var err error
	_ = writer.WriteField("from", params.From)
	_ = writer.WriteField("to", params.To)
	_ = writer.WriteField("subject", params.Subject)
	// html优先
	if len(params.Html) > 0 {
		_ = writer.WriteField("html", params.Html)
	} else {
		_ = writer.WriteField("text", params.Text)
	}
	// 用户回复接受邮箱
	if len(params.ReplyTo) > 0 {
		_ = writer.WriteField("replyTo", params.ReplyTo)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	response := &SendResponse{}
	if err := sesCli.request("POST", sesSendApi, writer.FormDataContentType(), body, response); err != nil {
		return nil, err
	}
	return response, nil
}

func TemplateSend(params *TemplateSendParams) (*SendResponse, error) {
	paramsByte, _ := json.Marshal(params)
	contentType := "application/json"
	response := &SendResponse{}
	body := bytes.NewBuffer(paramsByte)
	if err := sesCli.request("POST", sesTemplateSendAPI, contentType, body, response); err != nil {
		return nil, err
	}
	return response, nil
}

func Query(params *QueryParams) (*QueryResponse, error) {
	response := &QueryResponse{}
	url := sesQueryAPI + "?bulkId=" + params.BulkId
	if err := sesCli.request("GET", url, "", nil, response); err != nil {
		return nil, err
	}
	return response, nil
}

