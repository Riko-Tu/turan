package controllers

import (
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

func findErrIds(ids []string) string {
	var errIds string
	for _, v := range ids {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			errIds += ","
			errIds += v
		}
	}
	return errIds
}

// @Summary latex批量修改,单个修改(归档,丢弃到垃圾桶,删除)
// @Tags latex
// @Security ApiKeyAuth
// @Description latex批量修改,单个修改(归档,丢弃到垃圾桶,删除)
// @Accept  json
// @Produce  json
// @Param latex_ids query string false "latex ID列表(只要option=3的时 并且该latex为用户创建时,需要此参数)，以,分隔 例如 1,2,3"
// @Param user_latex_ids query string false "user latex ID列表，以,分隔 例如 1,2,3 (只要option=3的时, 并且该latex不是用户创建，放入此参数。表示用户要离开该latex)"
// @Param option query string true "1=恢复到正常状态;2=归档;3=丢弃到回收站;4=删除"
// @Success 200 {string} json
// @Router /latex [put]
func (c Controller) PutUserLatex(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrToken, newToken)
		return
	}

	var latexIdList,userLatexIdList []string
	latexIds := ctx.Query("latex_ids")
	if len(latexIds) > 0 {
		latexIdList = strings.Split(latexIds, ",")
	}
	userLatexIds := ctx.Query("user_latex_ids")
	if len(userLatexIds) > 0 {
		userLatexIdList = strings.Split(userLatexIds, ",")
	}

	option := ctx.Query("option")
	var updateStatus int64
	switch option {
	case "1":
		updateStatus = models.NormalStatus
	case "2":
		updateStatus = models.ArchiveStatus
	case "3":
		updateStatus = models.DeleteStatus
	case "4":
	default:
		fail(ctx, ErrNotFoundOption, newToken)
		return
	}

	var tmpList []string
	if len(latexIdList) > 0 {
		tmpList = append(tmpList, latexIdList...)
	}
	if len(userLatexIdList) > 0 {
		tmpList = append(tmpList, userLatexIdList...)
	}
	if len(tmpList) == 0 {
		fail(ctx, ErrId, newToken)
		return
	}
	errIds := findErrIds(tmpList)
	if len(errIds) > 0 {
		failErr := ErrIds
		failErr.Message = fmt.Sprintf("id %s err", errIds)
		fail(ctx, failErr, newToken)
		return
	}

	if option == "4" {
		if len(userLatexIdList) > 0 {
			total, err := models.GetUserLatexTotal(user.Id, userLatexIdList, true)
			if err != nil {
				log.Error(err.Error())
				fail(ctx, ErrQueryDB, newToken)
				return
			}
			if len(userLatexIdList) != total {
				fail(ctx, ErrId, newToken)
				return
			}
		}
		if err := models.TxDeleteLatexList(user.Id, latexIdList, userLatexIdList); err != nil {
			log.Error(err.Error())
			fail(ctx, ErrUpdateDB, newToken)
			return
		}
	} else {
		if len(userLatexIdList) == 0 {
			fail(ctx, ErrId, newToken)
			return
		}
		total, err := models.GetUserLatexTotal(user.Id, userLatexIdList,false)
		if err != nil {
			log.Error(err.Error())
			fail(ctx, ErrQueryDB, newToken)
			return
		}
		if len(userLatexIdList) != total {
			fail(ctx, ErrId, newToken)
			return
		}
		if err := models.UpdateUserLatexListStatus(user.Id, updateStatus, userLatexIdList); err != nil {
			log.Error(err.Error())
			fail(ctx, ErrUpdateDB, newToken)
			return
		}
	}
	resp(ctx, "ok", newToken)
	return
}