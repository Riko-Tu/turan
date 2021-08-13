package latex

import (
	"TEFS-BE/pkg/tencentCloud"
	"github.com/spf13/viper"
)

var (
	LatexBaseDir string
	LatexTemplateDir string
	LatexDefaultTemplateDir string
	LatexDirFormat = "%s/user/%d/latex/%d"
	// url secret
	Secret string
	// 文档分享url pattern
	UrlPattern string
	// 文档owner+协作者总和限制
	ShareLimit int64

	// latex创建项目时copy dir
	BlankProject string // 空白项目
	ExampleProject string // 样例项目
	UploadProject string // 上传项目
	// 模板目录
	TemplateDir string
	// 腾讯云cos服务
	CosService tencentCloud.Cos
	region 	string
	bucket 	string
	ImageUrlPrefix string // 封面图的url前缀
	ImageCosPathPrefix string // cos桶内存放封面图的目录
)

func Setup() {
	LatexBaseDir = viper.GetString("latex.path")
	LatexTemplateDir = viper.GetString("latex.template")
	LatexDefaultTemplateDir = viper.GetString("latex.defaultTemplate")
	Secret = viper.GetString("latex.secret")
	UrlPattern = viper.GetString("latex.url")
	ShareLimit = viper.GetInt64("latex.limit")
	BlankProject = viper.GetString("latex.blankProject")
	ExampleProject = viper.GetString("latex.exampleProject")
	UploadProject = viper.GetString("latex.uploadProject")
	TemplateDir = viper.GetString("latex.templates")

	region = viper.GetString("tencentCloud.region")
	bucket = viper.GetString("tencentCloud.cos.bucket")
	CosService = tencentCloud.Cos{
		Credential: &tencentCloud.Credential{
			AppId:     viper.GetString("tencentCloud.appId"),
			SecretId:  viper.GetString("tencentCloud.secretId"),
			SecretKey: viper.GetString("tencentCloud.secretKey"),
		},
		Region: region,
		Bucket: bucket,
	}
	ImageUrlPrefix = viper.GetString("latex.imageUrlPrefix")
	ImageCosPathPrefix = viper.GetString("latex.imageCosPathPrefix")
}