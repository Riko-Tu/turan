package router

import (
	"TEFS-BE/pkg/admin/auth"
	"TEFS-BE/pkg/latex/controllers"
	_ "TEFS-BE/pkg/latex/docs"
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/latex/utils"
	"TEFS-BE/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func StaticMiddle(ctx *gin.Context) {
	params := ctx.Query("params")
	if len(params) == 0 {
		ctx.Abort()
		ctx.Writer.Write([]byte("params not found"))
		return
	}
	var err error
	privatePath := viper.GetString("latex.privatePath")

	params, err = utils.RsaDecrypt(params, privatePath)
	if err != nil {
		ctx.Abort()
		ctx.Writer.Write([]byte("decrypt failed"))
		return
	}
	var token, latex, createUserIdStr, filePath string
	for _, v := range strings.Split(params, "&") {
		items := strings.Split(v, "=")
		if len(items) != 2 {
			continue
		}
		key := items[0]
		value := items[1]
		switch key {
		case "token":
			token = value
		case "user":
			createUserIdStr = value
		case "latex":
			latex = value
		case "filePath":
			filePath = value
		}
	}
	if strings.HasPrefix(filePath, "/") {
		filePath = filePath[1:]
	}
	user, _, isExpired, err := auth.ParseJwtToken(token)
	if err != nil || isExpired || user.Id <= 0 {
		ctx.Abort()
		ctx.Writer.Write([]byte("token error"))
		return
	}
	latexId, err := strconv.ParseInt(latex, 10, 64)
	if err != nil {
		ctx.Abort()
		ctx.Writer.Write([]byte("latexId error"))
		return
	}
	createUserId, err := strconv.ParseInt(createUserIdStr, 10, 64)
	if err != nil {
		ctx.Abort()
		ctx.Writer.Write([]byte("createUserID type error"))
		return
	}
	count, err := models.GetLatexCount(user.Id, createUserId, latexId)
	if err != nil {
		log.Error(err.Error())
	}
	if err != nil || count <= 0 {
		ctx.Abort()
		ctx.Writer.Write([]byte("record not found"))
		return
	}

	latexSavePath := viper.GetString("latex.path")
	basePath := filepath.Join(latexSavePath, "user", createUserIdStr, "latex", latex)
	targetPath := filepath.Join(basePath, filePath)
	if !strings.HasPrefix(targetPath, basePath) {
		ctx.Abort()
		ctx.Writer.Write([]byte("base path error"))
		return
	}

	staticUrl := filepath.Join("/static", "user", createUserIdStr, "latex", latex, filePath)
	if ctx.Request.URL.Path != staticUrl {
		ctx.Abort()
		ctx.Writer.Write([]byte("url unequal"))
		return
	}
}

func Setup(e *gin.Engine) {
	// 添加swagger生成接口文档。
	e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	docStatic := e.Group("/static")
	docStatic.Use(StaticMiddle)
	httpDir := viper.GetString("latex.path")
	docStatic.StaticFS("", http.Dir(httpDir))

	controller := controllers.Controller{}
	latexApi := e.Group("/api/doc")

	// latex
	latexApi.POST("/latex", controller.CreateLatex)
	latexApi.GET("/latex", controller.LatexList)
	latexApi.DELETE("/latex/:id", controller.DeleteLatex)
	latexApi.PUT("/latex/:id", controller.ModifyLatex)
	latexApi.GET("/latex/:id/tree", controller.LatexTree)
	latexApi.GET("/latex/:id/mv", controllers.FileMVMiddle, controller.LatexMV)
	latexApi.POST("/latex/:id/file", controllers.FileChangeMiddle, controller.LatexFile)
	latexApi.POST("/latex/:id/pdf", controller.LatexCompile)
	latexApi.PUT("/latex", controller.PutUserLatex)
	latexApi.GET("/zips", controller.GetLatexZips)
	latexApi.GET("/latex/:id/config", controller.GetLatexConfig)
	latexApi.PUT("/latex/:id/config", controller.PutLatexConfig)

	// templates
	latexApi.GET("/latex_templates", controller.LatexTemplatesInfo)
	latexApi.POST("/latex_templates", controller.UploadLatexTemplateInfo)
	latexApi.PUT("/latex_templates/:id", controller.UpdateLatexTemplateInfo)
	latexApi.DELETE("/latex_templates/:id", controller.DeleteLatexTemplateInfo)

	// share
	latexApi.GET("/share", controller.WaitingShareList)
	latexApi.POST("/share", controller.CollaborateLatex)
	latexApi.DELETE("/share/:id", controller.CancelShareUrl)
	latexApi.POST("/latex/:id/share", controller.EmailLatexURL)
	latexApi.GET("/latex/:id/share", controller.CollaboratorsList)
	latexApi.PUT("/latex/:id/share", controller.EditCollaboratorPower)
	latexApi.DELETE("/latex/:id/share", controller.DeleteCollaborator)

	// tag
	latexApi.POST("/tag", controller.CreateLatexTag)
	latexApi.GET("/tag", controller.GetLatexTagList)
	latexApi.PUT("/tag/:id", controller.ModifyLatexTag)
	latexApi.DELETE("/tag/:id", controller.DeleteLatexTag)

	// latex tag
	latexApi.POST("/tag/:id/latex/:latex_id", controller.TagAddLatex)
	latexApi.DELETE("/tag/:id/latex/:latex_id", controller.LatexDeleteTag)

	// leaps
	latexApi.GET("/latex/:id/leaps", controller.Leaps)
}
