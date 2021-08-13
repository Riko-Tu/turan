package router

import (
	// swaggo 文档初始化导入
	_ "TEFS-BE/docs"
	"TEFS-BE/pkg/creator/api/controllers"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(e *gin.Engine) {
	// 添加swagger生成接口文档。
	e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 用户本地部署应用程序， 创建用户自己腾讯云TKE web服务，需要用到用户腾讯云账户信息
	cloudEnvApi := e.Group("/api/cloudEnv")
	controller := controllers.CloudController{}
	cloudEnvApi.GET("/version", controller.Version)
	cloudEnvApi.GET("/region", controller.Region)
	cloudEnvApi.POST("/project", controller.Project)
	cloudEnvApi.GET("/account", controller.Account)
	cloudEnvApi.GET("/account/verify", controller.AccountVerify)
	cloudEnvApi.POST("/vpc", controller.Vpc)
	cloudEnvApi.POST("/securityGroup", controller.SecurityGroup)
	cloudEnvApi.POST("/securityGroup/policies", controller.SecurityGroupPolicies)
	cloudEnvApi.POST("/cos", controller.Cos)
	cloudEnvApi.POST("/tke", controller.Tke)
	cloudEnvApi.GET("/tke/status", controller.TkeStatus)
	cloudEnvApi.GET("/tke/instance/status", controller.TkeInstanceStatus)
	cloudEnvApi.GET("/tke/instance/publicIpAddress", controller.TkeInstancePublicIpAddress)
}
