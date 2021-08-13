package tencentCloud

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 腾讯云cos存储
type Cos struct {
	Credential *Credential
	Region     string
	Bucket     string
}

// 获取cos client
func (c Cos) getClient(isBaseUrl bool) *cos.Client {
	var client *cos.Client
	var baseUrl *cos.BaseURL
	if isBaseUrl {
		rawUrl := fmt.Sprintf("https://%s.cos.%s.myqcloud.com", c.Bucket, c.Region)
		u, _ := url.Parse(rawUrl)
		baseUrl = &cos.BaseURL{BucketURL: u}
	} else {
		baseUrl = nil
	}
	client = cos.NewClient(baseUrl, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  c.Credential.SecretId,
			SecretKey: c.Credential.SecretKey,
		},
	})
	return client
}

// 创建存储桶
func (c Cos) CreateBucket() error {
	client := c.getClient(true)
	_, err := client.Bucket.Put(context.Background(), nil)
	return err
}

// 查询所有存储桶
func (c Cos) GetBuckets() ([]cos.Bucket, error) {
	client := c.getClient(false)
	result, _, err := client.Service.Get(context.Background())
	if err != nil {
		return nil, err
	}
	return result.Buckets, nil
}

// 上传文件
func (c Cos) Upload(key, filePath string) error {
	client := c.getClient(true)
	// 通过本地文件上传对象
	_, err := client.Object.PutFromFile(context.Background(), key, filePath, nil)
	return err
}

// 下载文件
func (c Cos) Download(key, localPath string) error {
	client := c.getClient(true)
	// 下载到本地文件
	_, err := client.Object.GetToFile(context.Background(), key, localPath, nil)
	return err
}

// 获取路径下文件列表
func (c Cos) GetObjectList(dir string) ([]cos.Object, error) {
	client := c.getClient(true)
	opt := &cos.BucketGetOptions{
		Prefix:  dir,
		MaxKeys: 99999, // 获取cos文件名列表最大数量。因为cos接口未设置分页。这里指的一个比较高的值。用户基本上不会到达这个值
	}
	result, _, err := client.Bucket.Get(context.Background(), opt)
	if err != nil {
		return nil, err
	}
	return result.Contents, nil
}

// 获取临时秘钥，指定使用的cosAPI和cos文件路径
func (c Cos) GetTmpSecret(cosApis, cosPaths []string) (*sts.CredentialResult, error) {
	stsClient := sts.NewClient(
		c.Credential.SecretId,
		c.Credential.SecretKey,
		nil,
	)
	opt := &sts.CredentialOptions{
		DurationSeconds: int64(time.Hour.Seconds()),
		Region:          c.Region,
		Policy: &sts.CredentialPolicy{
			Statement: []sts.CredentialPolicyStatement{
				{
					Action:   cosApis,
					Effect:   "allow",
					Resource: cosPaths,
				},
			},
		},
	}
	res, err := stsClient.GetCredential(opt)
	if err != nil {
		return nil, err
	}
	if res.Error != nil {
		return nil, fmt.Errorf(res.Error.Error())
	}
	return res, nil
}

func (c Cos)JoinQcsCosPath(cosPaths []string) []string {
	cosBasePath := fmt.Sprintf("qcs::cos:%s:uid/%s:%s", c.Region, c.Credential.AppId, c.Bucket)
	for i, v := range cosPaths {
		cosPaths[i] = cosBasePath + v
	}
	return cosPaths
}

// 获取下载权限，传入可下载文件路径。
func (c Cos) GetDownloadTmpSecret(cosPaths []string) (*sts.CredentialResult, error) {
	appId := c.Credential.AppId
	bucket := c.Bucket
	region := c.Region
	cosBasePath := "qcs::cos:" + region + ":uid/" + appId + ":" + bucket
	//cosApis := []string{"name/cos:GetObject", "name/cos:GetBucket"} // cos下载api
	cosApis := []string{
		"name/cos:PostObject",
		"name/cos:PutObject",
		"name/cos:DeleteObject",
		"name/cos:HeadObject",
		"name/cos:GetObject",
		"name/cos:GetBucket",
	}
	for i, v := range cosPaths {
		cosPaths[i] = cosBasePath + v
	}
	return c.GetTmpSecret(cosApis, cosPaths)
}

// 获取在cos上vasplicense的下一文件的角标
func (c Cos) GetNextVaspLicenseDir(userId int64) (string, error) {
	userVaspLcensesDir := fmt.Sprintf("user_%d/vasp_license/", userId)
	objects, err := c.GetObjectList(userVaspLcensesDir)
	if err != nil {
		return "", err
	}
	var nextVaspLicenseDirNum int
	if objects == nil {
		nextVaspLicenseDirNum = 1
	} else {
		for _, o := range objects {
			key := o.Key
			if strings.HasPrefix(key, userVaspLcensesDir) {
				strItems := strings.Split(key, "/")
				if len(strItems) < 4 {
					continue
				} else {
					vaspLicenseIndex, err := strconv.Atoi(strItems[2])
					if err != nil {
						continue
					} else {
						if vaspLicenseIndex+1 > nextVaspLicenseDirNum {
							nextVaspLicenseDirNum = vaspLicenseIndex + 1
						}
					}
				}
			}
		}
		if nextVaspLicenseDirNum <= 0 {
			nextVaspLicenseDirNum = 1
		}
	}
	nextVaspLicenseDir := userVaspLcensesDir + strconv.Itoa(nextVaspLicenseDirNum)
	return nextVaspLicenseDir, nil
}

// 获取用户上传vasplicense到cos的临时秘钥，和指定上传路径
func (c Cos) GetVaspLicenseTmpSecret(userId int64) (*sts.CredentialResult, []string, error) {
	appId := c.Credential.AppId
	bucket := c.Bucket
	region := c.Region
	cosBasePath := "qcs::cos:" + region + ":uid/" + appId + ":" + bucket + "/"
	// 具体可使用cosAPI这里指定上传的api
	cosApis := []string{
		"name/cos:PostObject",
		"name/cos:PutObject",
		"name/cos:DeleteObject",
		"name/cos:HeadObject",
		"name/cos:GetObject",
	}

	dir, err := c.GetNextVaspLicenseDir(userId)
	if err != nil {
		return nil, nil, err
	}

	// 具体上传文件名,最多上传4个图片文件
	var cosPaths []string
	var keys []string
	for i := range [4]int{} {
		keys = append(keys, dir+"/"+strconv.Itoa(i)+".jpg")
		cosPaths = append(cosPaths, cosBasePath+dir+"/"+strconv.Itoa(i)+".jpg")
	}
	secret, err := c.GetTmpSecret(cosApis, cosPaths)
	if err != nil {
		return nil, nil, err
	}
	return secret, keys, nil
}

// 获取实验上传临时密钥
func (c Cos) GetUploadTmpSecret(paths []string) (*sts.CredentialResult, error) {
	cosBasePath := "qcs::cos:" + c.Region + ":uid/" + c.Credential.AppId + ":" + c.Bucket
	cosApis := []string{
		"name/cos:PostObject",
		"name/cos:PutObject",
		//简单上传操作
		"name/cos:PutObject",
		//表单上传对象
		"name/cos:PostObject",
		//分块上传：初始化分块操作
		"name/cos:InitiateMultipartUpload",
		//分块上传：List 进行中的分块上传
		"name/cos:ListMultipartUploads",
		//分块上传：List 已上传分块操作
		"name/cos:ListParts",
		//分块上传：上传分块块操作
		"name/cos:UploadPart",
		//分块上传：完成所有分块上传操作
		"name/cos:CompleteMultipartUpload",
		//取消分块上传操作
		"name/cos:AbortMultipartUpload",
		"name/cos:GetBucket",
		"name/cos:DeleteObject", // 删除权限
		"name/cos:HeadObject",
		"name/cos:GetObject",
	}
	for i, path := range paths {
		if strings.HasPrefix(path, "/") {
			paths[i] = cosBasePath + path
		} else {
			paths[i] = cosBasePath + "/" + path
		}
	}
	return c.GetTmpSecret(cosApis, paths)
}

// 获取删除实验文件夹临时密钥
func (c Cos) GetDeleteTmpSecret(paths []string) (*sts.CredentialResult, error) {
	cosBasePath := "qcs::cos:" + c.Region + ":uid/" + c.Credential.AppId + ":" + c.Bucket
	cosApis := []string{
		"name/cos:GetBucket",    // 查询权限
		"name/cos:DeleteObject", // 删除权限
		"name/cos:HeadObject",
	}
	for i, path := range paths {
		if strings.HasPrefix(path, "/") {
			paths[i] = cosBasePath + path
		} else {
			paths[i] = cosBasePath + "/" + path
		}
	}
	return c.GetTmpSecret(cosApis, paths)
}

// 获取实验文件夹所有权限临时密钥
func (c Cos) GetTmpSecretForAll(paths []string)  (*sts.CredentialResult, error) {
	cosBasePath := "qcs::cos:" + c.Region + ":uid/" + c.Credential.AppId + ":" + c.Bucket
	cosApis := []string{
		"*",
	}
	for i, path := range paths {
		if strings.HasPrefix(path, "/") {
			paths[i] = cosBasePath + path
		} else {
			paths[i] = cosBasePath + "/" + path
		}
	}
	fmt.Println(paths)
	return c.GetTmpSecret(cosApis, paths)
}

// 设置跨域配置
func (c Cos) PutBucketCors() error {
	client := c.getClient(true)
	opt := &cos.BucketPutCORSOptions{
		Rules: []cos.BucketCORSRule{
			{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"PUT", "GET","POST","DELETE","HEAD"},
				AllowedHeaders: []string{"*"},
				MaxAgeSeconds:  600,
				ExposeHeaders:  []string{"Etag"},
			},
		},
	}
	_, err := client.Bucket.PutCORS(context.Background(), opt)
	if err != nil {
		return err
	}
	return nil
}

// 临时秘钥获取cos客户端
func TmpSecretGetCosCli(secretId, secretKey, sessionToken, cosUrl string) (cli *cos.Client, e error) {
	u, e := url.Parse(cosUrl)
	if e != nil {
		return nil, e
	}
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:secretId,
			SecretKey:secretKey,
			SessionToken:sessionToken,
		},
	})
	return client, nil
}