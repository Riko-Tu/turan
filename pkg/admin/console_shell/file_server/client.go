package file_server

import (
	"TEFS-BE/pkg/admin/auth"
	"TEFS-BE/pkg/admin/model"
	"TEFS-BE/pkg/log"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	urlUtil "net/url"
	"strconv"
	"strings"
	"time"
)

func getTkeUrl(r *http.Request) (baseUrl string, err error) {
	projectIdStr := r.FormValue("project_id")
	projectId, err := strconv.ParseInt(projectIdStr, 10, 64)
	if err != nil {
		err = fmt.Errorf(`{"code":"5003", "msg":"project id is not int"}`)
		return
	}

	token := r.Header.Get("Authorization")
	var user *model.User
	user, _, isExpired, err := auth.ParseJwtToken(token)
	if err != nil {
		log.Error(err.Error())
		err = fmt.Errorf(`{"code":"5004", "msg":"parse token failed"}`)
		return
	}
	if isExpired {
		err = fmt.Errorf(`{"code":"5004", "msg":"token expired"}`)
		return
	}

	shell := &model.ConsoleShell{}
	if err = shell.Get(user.Id, projectId); err != nil {
		log.Error(err.Error())
		err = fmt.Errorf(`{"code":"5005", "msg":"not found console shell record"}`)
		return
	}
	address := shell.Address
	tmp := strings.Split(address, ":")
	if len(tmp) < 3 {
		log.Error(fmt.Sprintf("shell addres err,id:%d,address:%s", shell.Id, shell.Address))
		err = fmt.Errorf(`{"code":"5006", "msg":"record address err"}`)
		return
	}
	baseUrl = fmt.Sprintf("http://%s:%s", tmp[0], tmp[2])
	return
}

const uploadEndpoint = "uploadFile"
const downLoadEndpoint = "downFile"

// 接收文件上传到用户tke
func ReceiveFileToShellServer(w http.ResponseWriter, r *http.Request) () {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "authorization, access-control-allow-origin, content-type, x-requested-with")
	file, _, err := r.FormFile("file")
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte(`{"code":"5001", "msg":"get upload file failed"}`))
		return
	}

	remotePath := r.FormValue("remote_path")
	if len(remotePath) <= 0 {
		w.Write([]byte(`{"code":"5002", "msg":"remotePath err"}`))
		return
	}

	baseUrl, err := getTkeUrl(r)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	url := fmt.Sprintf("%s/%s?path=%s", baseUrl, uploadEndpoint, urlUtil.QueryEscape(remotePath))

	bodyBuffer := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuffer)
	fileWriter, err := bodyWriter.CreateFormFile("file", "file.txt")
	if _, err := io.Copy(fileWriter, file); err != nil {
		log.Error(err.Error())
		w.Write([]byte(`{"code":"5007", "msg":"copy upload file failed"}`))
		return
	}
	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	httpCli := http.Client{
		Timeout: time.Duration(3600) * time.Second,
	}
	resp, err := httpCli.Post(url, contentType, bodyBuffer)
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte(`{"code":"5008", "msg":"post upload file err"}`))
		return
	}
	if resp.StatusCode != http.StatusOK {
		w.Write([]byte(`{"code":"5009", "msg":"post upload file failed"}`))
		return
	}
	defer resp.Body.Close()
	w.Write([]byte(`{"code":"200", "msg":"upload file success"}`))
}

// 从用户TKE下载文件
func DownFileShellServer(w http.ResponseWriter, r *http.Request) () {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "authorization, access-control-allow-origin, content-type, x-requested-with")
	baseUrl, err := getTkeUrl(r)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	url := fmt.Sprintf("%s/%s?file=%s", baseUrl, downLoadEndpoint, r.FormValue("file"))
	httpCli := http.Client{
		Timeout: time.Duration(3600) * time.Second,
	}
	res, err := httpCli.Get(url)
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte(`{"code":"5010", "msg":"request file server failed"}`))
		return
	}

	ret, err := ioutil.ReadAll(res.Body)
	if err != nil {
		w.Write([]byte(`{"code":"5011", "msg":"read file failed"}`))
		return
	}

	w.Header().Set("Content-Disposition", res.Header.Get("Content-Disposition"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Accept-Ranges", "bytes")

	if _, err := w.Write(ret); err != nil {
		w.Write([]byte(`{"code":"5011", "msg":"download file failed"}`))
		return
	}
	return
}
