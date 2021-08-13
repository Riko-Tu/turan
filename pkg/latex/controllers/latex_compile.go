package controllers

import (
	latexServer "TEFS-BE/pkg/latex"
	"TEFS-BE/pkg/latex/tectonic"
	"TEFS-BE/pkg/latex/utils"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/gin-gonic/gin"
	"path/filepath"
	"strings"
)

const LatexOut = "out"

// @Summary 编译latex
// @Tags latex
// @Security ApiKeyAuth
// @Description 编译latex
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Param filepath query string true "latex 编译文件路径, 例如/main.tex(最外层目录)，/test/main.tex(test目录下文件)"
// @Param keep_logs query boolean false "试试保存log文件"
// @Success 200 {string} json
// @Router /latex/{id}/pdf [post]
func (c Controller) LatexCompile(ctx *gin.Context) {
	newToken, _, latex, _, controllerError := getModels(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}
	latexPath := fmt.Sprintf(latexServer.LatexDirFormat, latexServer.LatexBaseDir, latex.UserId, latex.Id)
	compileFilePath := filepath.Join(latexPath, LatexRaw, ctx.Query("filepath"))
	if !utils.PathExists(compileFilePath) {
		fail(ctx, ErrPath, newToken)
		return
	}
	tectonicCli := tectonic.Tectonic{
		ExecDir: compileFilePath,
	}
	if ctx.Query("keep_logs") == "true" {
		tectonicCli.KeepIntermediates()
		tectonicCli.KeepLogs()
	}
	outDir := filepath.Join(latexPath, LatexOut)
	_, stdErr, err := tectonicCli.Run(outDir)
	if err != nil {
		log.Error(err.Error())
		tmpControllerError := ErrCompile
		if len(stdErr) > 0 {
			e1 := strings.Split(stdErr, "error: something bad happened inside TeX; its output follows:")[0]
			e2 := strings.Split(e1, "See the LaTeX manual or LaTeX Companion for explanation.")[0]
			if len(e2) > 0 {
				tmpControllerError.Message = e2
			} else {
				tmpControllerError.Message = stdErr
			}
		}
		fail(ctx, tmpControllerError, newToken)
		return
	}
	resp(ctx, "ok", newToken)
	return
}
