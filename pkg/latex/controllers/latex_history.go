package controllers

import (
	"github.com/gin-gonic/gin"
)

// @Summary latex提交版本
// @Tags latex
// @Security ApiKeyAuth
// @Description latex提交版本
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Param memo query string true "备注"
// @Success 200 {string} json
// @Router /latex/{id}/version [post]
func (c Controller) LatexVersion(ctx *gin.Context) {
	newToken, user, latex, userLatex, ctxErr := getModels(ctx)
	if ctxErr != nil {
		fail(ctx, ctxErr, newToken)
		return
	}


}
