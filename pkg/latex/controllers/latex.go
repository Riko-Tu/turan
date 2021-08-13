package controllers

import (
	adminModels "TEFS-BE/pkg/admin/model"
	latexServer "TEFS-BE/pkg/latex"
	"TEFS-BE/pkg/latex/localcommand"
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/latex/utils"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const LatexNameMaxLen int = 200

func verifyLatexName(name string) bool {
	nameLen := utf8.RuneCountInString(name)
	if nameLen > LatexNameMaxLen || nameLen == 0 {
		return false
	}
	return true
}

const (
	blankCreate    = "1"
	exampleCreate  = "2"
	uploadCreate   = "3"
	templateCreate = "4"
)

// @Summary 创建latex
// @Tags latex
// @Security ApiKeyAuth
// @Description 创建latex
// @Accept  json
// @Produce  json
// @Param name query string false "latex 名字"
// @Param create_latex_type query string true "创建latex类型 1=空白创建(只会生成main.txt文件);2=样例创建;3=上传创建(zip文件);4=模板创建"
// @Param zip_file formData file false "create_latex_type=3(上传创建)时的zip文件(只支持zip文件)。"
// @Param template_name query string false "create_latex_type=4(模板创建)时的模板名称"
// @Success 200 {string} json
// @Router /latex [post]
func (c Controller) CreateLatex(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}

	var latexName string

	createLatexType := ctx.Query("create_latex_type")
	var zipFile *multipart.FileHeader
	if createLatexType == uploadCreate {
		zipFile, err = ctx.FormFile("zip_file")
		if err != nil {
			log.Error(err.Error())
			fail(ctx, ErrUploadZip, newToken)
			return
		}
		zipName := zipFile.Filename
		if len(zipName) >= 5 {
			latexName = zipName[0 : len(zipName)-4]
		} else {
			latexName = zipName
		}
		if !verifyLatexName(latexName) {
			latexName = latexName[0:LatexNameMaxLen-1]
		}
	} else {
		latexName = ctx.Query("name")
		if !verifyLatexName(latexName) {
			fail(ctx, ErrLatexTagName, newToken)
			return
		}
	}

	latex := &models.Latex{}
	latex.UserId = user.Id
	latex.LastModifiedUserId = user.Id
	latex.Name = latexName

	err = latex.TxCreate(func(latexDir string) error {
		if err := localcommand.CreateDir(latexDir); err != nil {
			return err
		}
		var copyPath string
		switch createLatexType {
		case blankCreate:
			copyPath = latexServer.BlankProject
		case exampleCreate:
			copyPath = latexServer.ExampleProject
		case uploadCreate:
			copyPath = latexServer.UploadProject
		case templateCreate:
			templateName := ctx.Query("template_name")
			copyPath = filepath.Join(latexServer.TemplateDir, templateName)
			if !utils.IsDir(copyPath) {
				return fmt.Errorf("template name not found")
			}
		default:
			return fmt.Errorf("create latex type error")
		}
		if err := localcommand.CopyDir(copyPath, latexDir); err != nil {
			return err
		}
		if createLatexType == uploadCreate {

			if zipFile.Size > LatexDirMaxSize {
				return fmt.Errorf("zip file size go beyond")
			}
			zipFileName := zipFile.Filename
			saveZipPath := filepath.Join(latexDir, LatexRaw, zipFileName)
			if err := ctx.SaveUploadedFile(zipFile, saveZipPath); err != nil {
				return err
			}
			rawPath := filepath.Join(latexDir, LatexRaw)
			if err := utils.Unzip(saveZipPath, rawPath); err != nil {
				return err
			}
			if err := os.RemoveAll(saveZipPath); err != nil {
				return err
			}
			unzipSize, err := utils.GetDirSize(rawPath)
			if err != nil {
				return err
			}
			if unzipSize > LatexDirMaxSize {
				return fmt.Errorf("unzip size go beyond")
			}
		}
		return nil
	})
	if err != nil {
		log.Error(err.Error())
		ErrMsg := ErrCreateLatex
		ErrMsg.Message = err.Error()
		fail(ctx, ErrMsg, newToken)
		return
	}

	data := make(map[string]int64)
	data["latex_id"] = latex.Id
	resp(ctx, data, newToken)
	return
}

// @Summary 获取latex列表
// @Tags latex
// @Security ApiKeyAuth
// @Description 获取latex列表
// @Accept  json
// @Produce  json
// @Param status query int true "状态 1=正常 2=归档 3=删除"
// @Param offset query int true "从第多少条起始"
// @Param limit query int true "获取条数"
// @Param sort_field query string true "创建时间 create_at; 最后更新时间update_at; 当前用户操作时间current_user_operation_time;"
// @Param sort query string true "asc升序 desc降序"
// @Param id query int false "latex文档ID(非必填项)"
// @Success 200 {string} json
// @Router /latex [get]
func (c Controller) LatexList(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}

	status, err := strconv.ParseInt(ctx.Query("status"), 10, 64)
	if err != nil || status > 3 || status < 1 {
		fail(ctx, ErrLatexStatus, newToken)
		return
	}
	sortField := ctx.Query("sort_field")
	if sortField != "create_at" && sortField != "update_at" && sortField != "current_user_operation_time" {
		fail(ctx, ErrSortField, newToken)
		return
	}
	offset, limit, sort, controllerError := verifyOffsetLimitSort(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}

	var latexId *int64
	// id字段是否为空
	idStr := ctx.Query("id")
	if idStr != "" {
		id, err := conversionId(idStr)
		if err != nil {
			fail(ctx, ErrLatexId, newToken)
			return
		}
		latexId = &id
	}

	// 若传入id字段为空，latexId传nil即可
	records, total, err := models.GetLatexList(user.Id, offset, limit, status, sort, sortField, latexId)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}
	for _, record := range records {
		if record.CreateUserId == user.Id {
			record.LatexIsCurrentUser = true
		}
	}

	data := make(map[string]interface{})
	data["total"] = total
	data["records"] = records
	resp(ctx, data, newToken)
	return
}

// api ctx get db user model and latex model
func getUserAndLatex(ctx *gin.Context) (token string, user *adminModels.User, latex *models.Latex, controllerError *ControllerError) {
	user, token, err := getUserForToken(ctx)
	if err != nil {
		controllerError = ErrToken
		return
	}
	id, err := conversionId(ctx.Param("id"))
	if err != nil {
		controllerError = ErrId
		return
	}
	latex = &models.Latex{}
	if err := latex.Get(id); err != nil || latex.UserId != user.Id {
		controllerError = ErrNotFoundRecord
		return
	}
	return
}

// @Summary 删除latex
// @Tags latex
// @Security ApiKeyAuth
// @Description 删除latex
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Success 200 {string} json
// @Router /latex/{id} [delete]
func (c Controller) DeleteLatex(ctx *gin.Context) {
	newToken, _, latex, controllerError := getUserAndLatex(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}

	ups := map[string]interface{}{"delete_at": time.Now().Unix()}
	err := latex.TxDelete(ups, func(latexDir string) error {
		if !utils.PathExists(latexDir) {
			return nil
		}
		return localcommand.Rm(latexDir)
	})
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrDeleteLatex, newToken)
		return
	}

	resp(ctx, "ok", newToken)
	return
}

// @Summary 修改latex
// @Tags latex
// @Security ApiKeyAuth
// @Description 修改latex
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Param name query string true "latex 名字"
// @Success 200 {string} json
// @Router /latex/{id} [put]
func (c Controller) ModifyLatex(ctx *gin.Context) {
	newToken, _, latex, controllerError := getUserAndLatex(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}

	latexNewName := ctx.Query("name")
	if !verifyLatexName(latexNewName) {
		fail(ctx, ErrLatexName, newToken)
		return
	}
	ups := map[string]interface{}{"name": latexNewName}
	if err := latex.Update(ups); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	resp(ctx, "ok", newToken)
	return
}

const createZipDIR = "create_zip_dir"

// @Summary 下载latex zips
// @Tags latex
// @Security ApiKeyAuth
// @Description 下载latex zips
// @Accept  json
// @Produce  json
// @Param latex_ids query string true "需要下载的latex ID 可多个，例如1,2,3"
// @Success 200 {string} json
// @Router /zips [get]
func (c Controller) GetLatexZips(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}

	latexIdsStr := ctx.Query("latex_ids")
	latexIds := strings.Split(latexIdsStr, ",")
	for _, v := range latexIds {
		_, err := conversionId(v)
		if err != nil {
			fail(ctx, ErrId, newToken)
			return
		}
	}

	latexInfoItems, err := models.GetLatexJoinUserLatex(user.Id, latexIds)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}
	if len(latexIds) != len(latexInfoItems) {
		fail(ctx, ErrId, newToken)
		return
	}

	createZipDir := filepath.Join(latexServer.LatexBaseDir, "user", strconv.FormatInt(user.Id, 10), createZipDIR)
	if !utils.PathExists(createZipDir) {
		if err := localcommand.CreateDir(createZipDir); err != nil {
			log.Error(err.Error())
			fail(ctx, ErrCreateFile, newToken)
			return
		}
	}

	zipPaths := []string{}
	for _, v := range latexInfoItems {
		latexDir := fmt.Sprintf(latexServer.LatexDirFormat, latexServer.LatexBaseDir, v.UserId, v.Id)
		rawPath := filepath.Join(latexDir, LatexRaw)
		ul := uuid.NewV4().String()
		tmpPath := filepath.Join(createZipDir, ul)
		if err := localcommand.CreateDir(tmpPath); err != nil {
			log.Error(err.Error())
			fail(ctx, ErrCreateFile, newToken)
			return
		}
		defer localcommand.Rm(tmpPath)
		zipPathDir := filepath.Join(createZipDir, ul)
		zipPaths = append(zipPaths, zipPathDir)
		zipPath := filepath.Join(zipPathDir, v.Name + ".zip")
		if err := utils.ZipCompressor(zipPath, "", rawPath); err != nil {
			log.Error(err.Error())
			fail(ctx, ErrCompress, newToken)
			return
		}
	}

	var tmpZipPath, zipName string
	if len(latexInfoItems) == 1 {
		zipName = latexInfoItems[0].Name
		tmpZipPath = filepath.Join(zipPaths[0], latexInfoItems[0].Name + ".zip")
	} else {
		zipName = fmt.Sprintf("doc-%s", strings.Join(latexIds, "-"))
		tmpZipPath = filepath.Join(createZipDir, zipName+".zip")
		if err := utils.ZipCompressor(tmpZipPath, "", zipPaths...); err != nil {
			log.Error(err.Error())
			fail(ctx, ErrCompress, newToken)
			return
		}
		defer localcommand.Rm(tmpZipPath)
	}

	ctx.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", zipName))
	ctx.File(tmpZipPath)
	return
}