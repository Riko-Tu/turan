package controllers

import (
	adminModels "TEFS-BE/pkg/admin/model"
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/log"
	"github.com/gin-gonic/gin"
	"time"
)

func getTagUserLatex(ctx *gin.Context) (token string, user *adminModels.User, tagId, latexId int64, controllerError *ControllerError) {
	user, token, err := getUserForToken(ctx)
	if err != nil {
		controllerError = ErrToken
		return
	}
	tagId, err = conversionId(ctx.Param("id"))
	if err != nil {
		controllerError = ErrId
		return
	}
	latexId, err = conversionId(ctx.Param("latex_id"))
	if err != nil {
		controllerError = ErrId
		return
	}
	tag := &models.UserLatexTag{}
	if err := tag.Get(tagId); err != nil {
		log.Error(err.Error())
		controllerError = ErrNotFoundTagRecord
		return
	}
	if tag.UserId != user.Id {
		controllerError = ErrNotFoundTagRecord
		return
	}
	total, err := models.GetUserLatexRecordTotal(user.Id, latexId)
	if err != nil {
		log.Error(err.Error())
		controllerError = ErrQueryDB
		return
	}
	if total == 0 {
		controllerError = ErrNotFoundLatexRecord
		return
	}
	return
}

// @Summary latex添加tag
// @Tags latexTag
// @Security ApiKeyAuth
// @Description tag添加latex
// @Accept  json
// @Produce  json
// @Param id path int64 true "tag ID"
// @Param latex_id path int64 true "latex ID"
// @Success 200 {string} json
// @Router /tag/{id}/latex/{latex_id} [post]
func (c Controller) TagAddLatex(ctx *gin.Context) {
	newTokne, user, tagId, latexId, controllerError := getTagUserLatex(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newTokne)
		return
	}

	tagLatex := &models.LatexTag{}
	if err := tagLatex.GetForUserTagLatex(user.Id, tagId, latexId); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newTokne)
		return
	}
	if tagLatex.Id > 0 {
		fail(ctx, ErrRecordIsExist, newTokne)
		return
	}

	nowTime := time.Now().Unix()
	tagLatex.LatexId = latexId
	tagLatex.UserId = user.Id
	tagLatex.UserLatexTagId = tagId
	tagLatex.CreateAt = nowTime
	if err := tagLatex.Create(); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrCreateDB, newTokne)
		return
	}

	resp(ctx, tagLatex, newTokne)
	return
}

// @Summary latex删除tag
// @Tags latexTag
// @Security ApiKeyAuth
// @Description tag添加latex
// @Accept  json
// @Produce  json
// @Param id path int64 true "tag ID"
// @Param latex_id path int64 true "latex ID"
// @Success 200 {string} json
// @Router /tag/{id}/latex/{latex_id} [delete]
func (c Controller) LatexDeleteTag(ctx *gin.Context) {
	newToken, user, tagId, latexId, controllerError := getTagUserLatex(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}

	tagLatex := &models.LatexTag{}
	if err := tagLatex.GetForUserTagLatex(user.Id, tagId, latexId); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}
	if tagLatex.Id == 0 {
		fail(ctx, ErrNotFoundRecord, newToken)
		return
	}

	ups := map[string]interface{}{"delete_at": time.Now().Unix()}
	if err := tagLatex.Update(ups); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	resp(ctx, "ok", newToken)
	return
}
