package controllers

import (
	adminModels "TEFS-BE/pkg/admin/model"
	latexServer "TEFS-BE/pkg/latex"
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/latex/utils"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"path"
	"path/filepath"
)

func getModels(ctx *gin.Context) (token string, user *adminModels.User, latex *models.Latex, userLatex *models.UserLatex, controllerError *ControllerError) {
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
	if err := latex.Get(id); err != nil {
		log.Error(err.Error())
		controllerError = ErrNotFoundRecord
		return
	}
	userLatex = &models.UserLatex{}
	if err := userLatex.GetForUserAndLatex(user.Id, latex.Id); err != nil {
		log.Error(err.Error())
		controllerError = ErrNotFoundRecord
		return
	}
	return
}

// @Summary 获取latex目录树
// @Tags latex
// @Security ApiKeyAuth
// @Description 获取latex目录树
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Success 200 {string} json
// @Router /latex/{id}/tree [get]
func (c Controller) LatexTree(ctx *gin.Context) {
	newToken, _, latex, _, ctxErr := getModels(ctx)
	if ctxErr != nil {
		fail(ctx, ctxErr, newToken)
		return
	}
	latexPath := fmt.Sprintf(latexServer.LatexDirFormat, latexServer.LatexBaseDir, latex.UserId, latex.Id)
	latexPath = path.Join(latexPath, LatexRaw)

	tree := &utils.Dir{
		Name: ".",
	}

	if err := utils.DirTree(latexPath, tree); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrGetTree, newToken)
		return
	}

	resp(ctx, tree, newToken)
	return
}

// @Summary 移动latex文件或目录
// @Tags latex
// @Security ApiKeyAuth
// @Description 移动latex文件或目录
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Param source_path query string true "源路径"
// @Param target_path query string true "目标路径"
// @Success 200 {string} json
// @Router /latex/{id}/mv [get]
func (c Controller) LatexMV(ctx *gin.Context) {
	newToken, _, latex, userLatex, ctxErr := getModels(ctx)
	if ctxErr != nil {
		fail(ctx, ctxErr, newToken)
		return
	}

	if userLatex.Power != ReadAndWriteAccess {
		fail(ctx, ErrNotAuthority, newToken)
		return
	}

	latexPath := fmt.Sprintf(latexServer.LatexDirFormat, latexServer.LatexBaseDir, latex.UserId, latex.Id)
	latexPath = path.Join(latexPath, LatexRaw)

	ErrMsg := ErrParam
	sourcePath := ctx.Query("source_path")
	if len(sourcePath) == 0 {
		ErrMsg.Message = "miss params source_path"
		fail(ctx, ErrMsg, newToken)
		return
	}
	targetPath := ctx.Query("target_path")
	if len(targetPath) == 0 {
		ErrMsg.Message = "miss params target_path"
		fail(ctx, ErrMsg, newToken)
		return
	}

	sourcePath = filepath.Join(sourcePath)
	targetPath = filepath.Join(targetPath)
	ctx.Set("fileType", utils.GetFileType(filepath.Join(latexPath, sourcePath)))

	if err := os.Rename(filepath.Join(latexPath, sourcePath), filepath.Join(latexPath, targetPath)); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrMV, newToken)
		return
	}

	ctx.Set(operationIsSuccess, true)
	ctx.Set("userId", userLatex.UserId)
	ctx.Set("latexId", userLatex.LatexId)
	ctx.Set("filePath", sourcePath)
	ctx.Set("targetPath", targetPath)

	resp(ctx, "ok", newToken)
	return
}