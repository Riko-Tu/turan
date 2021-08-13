package controllers

import (
	"TEFS-BE/pkg/admin/model"
	latexServer "TEFS-BE/pkg/latex"
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/latex/utils"
	"TEFS-BE/pkg/log"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"mime/multipart"
	"os"
	path2 "path"
	"regexp"
	"strings"
)
var zipCharRe = regexp.MustCompile("[[-_a-zA-Z0-9\u4e00-\u9fcc]")

var settingService = model.SettingService{}
var latexTemplateType = []string{
	"paper", // 论文模板目录
	"notes", // 笔记模板目录
}

// @Summary 获取latex模板
// @Tags template
// @Security ApiKeyAuth
// @Description 获取latex模板
// @Accept  json
// @Produce  json
// @Param offset query int64 true "从多少条开始"
// @Param limit query int64 true "返回的条数"
// @Param type query string true "获取模板类型：paper=论文模板，notes=笔记模板"
// @Success 200 {string} json
// @Router /latex_templates [get]
func (c Controller) LatexTemplatesInfo(ctx *gin.Context) {
	_, token, err := getUserForToken(ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrToken, "")
		return
	}

	offset, limit, controllerError := verifyOffsetLimit(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, token)
		return
	}

	templateType := ctx.Query("type")
	latexTemplateList, total, err := models.GetLatexTemplates(templateType, offset, limit)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, token)
		return
	}

	data := make(map[string]interface{})
	data["total"] = total
	data[templateType] = latexTemplateList

	resp(ctx, data, token)
	return
}

// @Summary 上传latex模板(需要管理员权限)
// @Tags template
// @Security ApiKeyAuth
// @Description 上传latex模板(需要管理员权限)
// @Accept  json
// @Produce  json
// @Param author query string true "模板作者"
// @Param license query string true "模板license"
// @Param abstract query string true "模板摘要"
// @Param image formData file true "模板封面图"
// @Param file formData file true "要操作的文件"
// @Param type query string true "上传模板类型：paper=论文模板，notes=笔记模板"
// @Success 200 {string} json
// @Router /latex_templates [post]
func (c Controller) UploadLatexTemplateInfo(ctx *gin.Context) {
	user, token, err := getUserForToken(ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrToken, "")
		return
	}
	templateType := ctx.Query("type")

	if !settingService.UserIsAdmin(user) {
		fail(ctx, ErrNotAuthority, token)
		return
	}

	author := ctx.Query("author")
	license := ctx.Query("license")
	abstract := ctx.Query("abstract")
	image, err := ctx.FormFile("image")
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUploadFile, token)
		return
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUploadFile, token)
		return
	}

	fileName := file.Filename
	name := strings.Split(fileName, ".zip")[0]
	// 模板存储文件夹路径 删除上传模板文件名字中的特殊字符，用"_"连接
	//folderName := strings.Replace(name, " ", "", -1)
	strs := strings.FieldsFunc(name, templatePathFormat)
	folderName := strings.Join(strs, "_")

	templatesDir := latexServer.TemplateDir
	// 添加子目录，论文模板or笔记模板
	templatesDir = path2.Join(templatesDir, templateType)
	zipPath := path2.Join(templatesDir, folderName + ".zip")
	if err := ctx.SaveUploadedFile(file, zipPath); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrSaveFile, token)
		return
	}
	dest := strings.Split(zipPath, ".zip")[0]
	rawDest := path2.Join(dest, "raw")
	outDest := path2.Join(dest, "out")
	// 创建模板源文件所在文件夹{template_name}/raw
	if err := os.MkdirAll(rawDest, 0755); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUploadZip, token)
		return
	}
	// 创建模板编译后文件所在文件夹{template_name}/out
	if err := os.MkdirAll(outDest, 0755); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUploadZip, token)
		return
	}
	if err := utils.Unzip(zipPath, rawDest); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUploadZip, token)
		return
	}

	if err := os.Remove(zipPath); err != nil {
		log.Error(err.Error())
	}

	// 写入数据库
	template := &models.LatexTemplate{}
	template.Title = name
	template.Author = author
	template.License = license
	template.Abstract = abstract
	template.Type = templateType
	template.Path = folderName
	template.Image, err = saveImage(image, template.Type, template.Path, ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrSaveImage, token)
		return
	}

	if err = template.Create(); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrCreateDB, token)
		return
	}

	resp(ctx, "ok", token)
	return
}

// @Summary 更新latex模板(需要管理员权限)
// @Tags template
// @Security ApiKeyAuth
// @Description 更新latex模板(需要管理员权限)
// @Accept  json
// @Produce  json
// @Param id path int true "id"
// @Param author query string false "模板作者"
// @Param license query string false "模板license"
// @Param abstract query string false "模板摘要"
// @Param image formData file true "模板封面图"
// @Success 200 {string} json
// @Router /latex_templates/{id} [put]
func (c Controller) UpdateLatexTemplateInfo(ctx *gin.Context) {
	_, token, err := getUserForToken(ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrToken, "")
		return
	}

	templateId, err := conversionId(ctx.Param("id"))
	template := &models.LatexTemplate{}
	if err = template.Get(templateId); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrId, token)
		return
	}

	ups := make(map[string]interface{})
	if ctx.Query("author") != "" {
		ups["author"] = ctx.Query("author")
	}
	if ctx.Query("abstract") != "" {
		ups["abstract"] = ctx.Query("abstract")
	}
	if ctx.Query("license") != "" {
		ups["license"] = ctx.Query("license")
	}

	image, err := ctx.FormFile("image")
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUploadFile, token)
		return
	}
	// 用户上传了图片才执行以下代码
	if image.Size > 0 {
		ups["image"], err = saveImage(image, template.Type, template.Title, ctx)
		if err != nil {
			log.Error(err.Error())
			fail(ctx, ErrSaveFile, token)
			return
		}
	}

	if err = template.Update(ups); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, token)
		return
	}

	resp(ctx, "ok", token)
	return
}

// @Summary 删除latex模板(需要管理员权限)
// @Tags template
// @Security ApiKeyAuth
// @Description 删除latex模板(需要管理员权限)
// @Accept  json
// @Produce  json
// @Param id path int true "要操作的文件"
// @Success 200 {string} json
// @Router /latex_templates/{id} [delete]
func (c Controller) DeleteLatexTemplateInfo(ctx *gin.Context) {
	_, token, err := getUserForToken(ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrToken, "")
		return
	}

	templateId, err := conversionId(ctx.Param("id"))
	template := &models.LatexTemplate{}
	if err = template.Get(templateId); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrId, token)
		return
	}

	if err = template.Delete(); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, token)
		return
	}

	resp(ctx, "ok", token)
	return
}

// 保存封面图
func saveImage(image *multipart.FileHeader, templateType, folder string, ctx *gin.Context) (url string, err error) {
	// 封面图所在路径:模板目录/模板类型(paper/notes)/模板名字/封面图
	imageDir := path2.Join(latexServer.TemplateDir, templateType)
	imageDir = path2.Join(imageDir, folder)
	imagePath := path2.Join(imageDir, image.Filename)
	// 覆盖封面图
	if err = ctx.SaveUploadedFile(image, imagePath); err != nil {
		return
	}
	// 上传cos
	cosPath := latexServer.ImageCosPathPrefix + templateType + "/" + folder + "/" + image.Filename
	if err = latexServer.CosService.Upload(cosPath, imagePath); err != nil {
		return
	}
	url = latexServer.ImageUrlPrefix + cosPath
	return
}

// 获取当前目录下所有目录的名字（即当前目录下所有模板）
func getTemplateNameList(dir string) (templates []string, err error) {
	fileInfoList, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	for _, v := range fileInfoList {
		if v.IsDir() {
			templates = append(templates, v.Name())
		}
	}
	return
}

func templatePathFormat(c rune)bool{
	return !zipCharRe.MatchString(string(c))
}