package controllers

import (
	adminModels "TEFS-BE/pkg/admin/model"
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/log"
	"github.com/gin-gonic/gin"
	"time"
	"unicode/utf8"
)

const LatexTagNameMaxLen int = 30

func verifyLatexTagName(name string) bool {
	nameLen := utf8.RuneCountInString(name)
	if nameLen > LatexTagNameMaxLen || nameLen == 0 {
		return false
	}
	return true
}

// @Summary 创建tag
// @Tags tag
// @Security ApiKeyAuth
// @Description 创建tag
// @Accept  json
// @Produce  json
// @Param name query string true "tag  名字"
// @Success 200 {string} json
// @Router /tag [post]
func (c Controller) CreateLatexTag(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}
	name := ctx.Query("name")
	if !verifyLatexTagName(name) {
		fail(ctx, ErrLatexTagName, newToken)
		return
	}
	userLatexTag := &models.UserLatexTag{}
	if err := userLatexTag.GetByUserAndName(user.Id, name); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}
	if userLatexTag.Id > 0 {
		fail(ctx, ErrLatexTagIsExist, newToken)
		return
	}

	nowTime := time.Now().Unix()
	userLatexTag.Name = name
	userLatexTag.UserId = user.Id
	userLatexTag.CreateAt = nowTime
	userLatexTag.UpdateAt = nowTime
	if err := userLatexTag.Create(); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrCreateDB, newToken)
		return
	}

	data := map[string]int64{"tag_id": userLatexTag.Id}
	resp(ctx, data, newToken)
	return
}

// @Summary 获取tag列表
// @Tags tag
// @Security ApiKeyAuth
// @Description 获取tag列表
// @Accept  json
// @Produce  json
// @Param offset query int64 true "偏移量"
// @Param limit query int64 true "条数"
// @Param sort_field query string true "名字 name; 创建时间create_at"
// @Param sort query string true "asc升序 desc降序"
// @Success 200 {string} json
// @Router /tag [get]
func (c Controller) GetLatexTagList(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}
	offset, limit, sort, controllerError := verifyOffsetLimitSort(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}
	sortField := ctx.Query("sort_field")
	if sortField != "name" && sortField != "create_at" {
		fail(ctx, ErrSortField, newToken)
		return
	}
	records, total, err := models.GetUserLatexTags(user.Id, offset, limit, sort, sortField)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}
	data := make(map[string]interface{})
	data["total"] = total
	data["records"] = records
	resp(ctx, data, newToken)
	return
}

func getLatexTagAndUser(ctx *gin.Context) (token string, user *adminModels.User, userLatexTag *models.UserLatexTag, controllerError *ControllerError) {
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
	userLatexTag = &models.UserLatexTag{}
	if err := userLatexTag.Get(id); err != nil || userLatexTag.UserId != user.Id {
		controllerError = ErrNotFoundRecord
		return
	}
	return
}

// @Summary 修改tag
// @Tags tag
// @Security ApiKeyAuth
// @Description 修改tag
// @Accept  json
// @Produce  json
// @Param id path int64 true "tag ID"
// @Param name query string true "名字"
// @Success 200 {string} json
// @Router /tag/{id} [put]
func (c Controller) ModifyLatexTag(ctx *gin.Context) {
	newToken, user, userLatexTag, controllerError := getLatexTagAndUser(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}
	name := ctx.Query("name")
	if !verifyLatexTagName(name) {
		fail(ctx, ErrLatexTagName, newToken)
		return
	}
	if userLatexTag.Name == name {
		resp(ctx, "ok", newToken)
		return
	}

	latexTag := models.UserLatexTag{}
	if err := latexTag.GetByUserAndName(user.Id, name); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}
	if latexTag.Id > 0 {
		fail(ctx, ErrLatexTagNameRepetition, newToken)
		return
	}
	ups := map[string]interface{}{"name": name}
	if err := userLatexTag.Update(ups); err != nil {
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	resp(ctx, "ok", newToken)
	return
}

// @Summary 删除tag
// @Tags tag
// @Security ApiKeyAuth
// @Description 删除tag
// @Accept  json
// @Produce  json
// @Param id path int64 true "tag ID"
// @Success 200 {string} json
// @Router /tag/{id} [delete]
func (c Controller) DeleteLatexTag(ctx *gin.Context) {
	newToken, _, userLatexTag, controllerError := getLatexTagAndUser(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}

	nowTime := time.Now().Unix()
	ups := map[string]interface{}{"delete_at": nowTime}
	if err := userLatexTag.Update(ups); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	resp(ctx, "ok", newToken)
	return
}
