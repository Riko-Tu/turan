package controllers

import (
	latexServer "TEFS-BE/pkg/latex"
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/latex/utils"
	"TEFS-BE/pkg/log"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"path/filepath"
)

type LatexConfig struct {
	// 编译文件
	MainDocument string `json:"main_document"`
}

// @Summary 获取latex配置信息
// @Tags latex
// @Security ApiKeyAuth
// @Description 获取latex配置信息
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Success 200 {string} json
// @Router /latex/{id}/config [get]
func (c Controller) GetLatexConfig(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}
	latexId, err := conversionId(ctx.Param("id"))
	if err != nil {
		fail(ctx, ErrId, newToken)
		return
	}

	userLatex := &models.UserLatex{}
	if err := userLatex.GetForUserAndLatex(user.Id, latexId); err != nil {
		fail(ctx, ErrNotFoundRecord, newToken)
		return
	}
	data := &LatexConfig{}
	if err := json.Unmarshal([]byte(userLatex.ConfigJson), data); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrConfigFormat, newToken)
		return
	}
	resp(ctx, data, newToken)
	return
}

// @Summary 修改latex配置信息
// @Tags latex
// @Security ApiKeyAuth
// @Description 修改latex配置信息
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Param main_document query string false "编译文件路径"
// @Success 200 {string} json
// @Router /latex/{id}/config [put]
func (c Controller) PutLatexConfig(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}
	latexId, err := conversionId(ctx.Param("id"))
	if err != nil {
		fail(ctx, ErrId, newToken)
		return
	}

	userLatex := &models.UserLatex{}
	if err := userLatex.GetForUserAndLatex(user.Id, latexId); err != nil {
		fail(ctx, ErrNotFoundRecord, newToken)
		return
	}
	latexConfig := &LatexConfig{}
	if err := json.Unmarshal([]byte(userLatex.ConfigJson), latexConfig); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrConfigFormat, newToken)
		return
	}

	latex := &models.Latex{}
	if err := latex.Get(latexId); err != nil {
		fail(ctx, ErrNotFoundRecord, newToken)
		return
	}

	mainDocument := ctx.Query("main_document")
	latexBasePath := fmt.Sprintf(latexServer.LatexDirFormat, latexServer.LatexBaseDir, latex.UserId, latex.Id)
	mainDocumentPath := filepath.Join(latexBasePath, LatexRaw, mainDocument)
	if !utils.PathExists(mainDocumentPath) {
		fail(ctx, ErrPath, newToken)
		return
	}

	latexConfig.MainDocument = mainDocument
	latexJsonByte, err := json.Marshal(latexConfig)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	ups := make(map[string]interface{})
	ups["config_json"] = string(latexJsonByte)
	if err := userLatex.Update(ups); err != nil {
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	resp(ctx, "ok", newToken)
	return
}
