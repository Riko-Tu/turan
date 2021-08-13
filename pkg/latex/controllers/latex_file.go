package controllers

import (
	latexServer "TEFS-BE/pkg/latex"
	"TEFS-BE/pkg/latex/localcommand"
	"TEFS-BE/pkg/latex/utils"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	path2 "path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	LatexDirMaxSize    int64 = 1024 * 1024 * 100 // 100M
	FileNameMaxLen     int   = 30
	LatexRaw                 = "raw"
	ReadAndWriteAccess int64 = 1

	fileDownload = "download"
	fileUpload   = "upload"
	fileDelete   = "delete"
	fileCreate   = "create"
	fileRename   = "rename"
	fileCover    = "cover"

	operationIsSuccess = "success"
)

// @Summary latex文件操作
// @Tags latex
// @Security ApiKeyAuth
// @Description latex文件操作
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Param option query string true "操作类型 upload:上传文件, download:下载文件, delete:删除, create:创建, rename:重命名"
// @Param file formData file false "要操作的文件"
// @Param path query string true "文件或目录,如果在某个目录下填写完整路径,如果最外层目录填 / "
// @Param create_type query string false "当option=create需要此参数，dir:创建目录，file创建文件夹"
// @Param new_name query string false "当option=rename需要此参数，目录或文件的新名字,最多30个字符"
// @Success 200 {string} json
// @Router /latex/{id}/file [post]
func (c Controller) LatexFile(ctx *gin.Context) {
	newToken, _, latex, userLatex, ctxErr := getModels(ctx)
	if ctxErr != nil {
		fail(ctx, ctxErr, newToken)
		return
	}
	latexPath := fmt.Sprintf(latexServer.LatexDirFormat, latexServer.LatexBaseDir, latex.UserId, latex.Id)
	latexPath = path2.Join(latexPath, LatexRaw)

	option := ctx.Query("option")
	if option != fileDownload && userLatex.Power != ReadAndWriteAccess {
		fail(ctx, ErrNotAuthority, newToken)
		return
	}

	path := path2.Join(latexPath, ctx.Query("path"))
	if option == fileUpload {
		ctx.Set("fileType", utils.FileType)
	} else {
		ctx.Set("fileType", utils.GetFileType(path))
	}

	var controllerError *ControllerError
	switch option {
	case fileUpload:
		controllerError = uploadFile(ctx, path)
	case fileDownload:
		controllerError = downloadFile(ctx, path)
		if controllerError == nil {
			return
		}
	case fileDelete:
		controllerError = deleteFile(latexPath, path)
	case fileCreate:
		controllerError = createFile(ctx, path)
	case fileRename:
		controllerError = rename(ctx, latexPath, path)
	default:
		controllerError = ErrNotFoundOption
	}
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}
	ctx.Set(operationIsSuccess, true)
	ctx.Set("userId", userLatex.UserId)
	ctx.Set("latexId", userLatex.LatexId)
	if option != fileUpload {
		ctx.Set("filePath", ctx.Query("path"))
	}
	resp(ctx, "ok", newToken)
	return
}

func rename(ctx *gin.Context, latexPath, path string) *ControllerError {
	if !utils.PathExists(path) {
		return ErrPath
	}
	if latexPath == path || latexPath == path[:len(path)-1] {
		return ErrPath
	}
	newName := ctx.Query("new_name")
	nameLen := utf8.RuneCountInString(newName)
	if nameLen < 1 || nameLen > FileNameMaxLen {
		return ErrNewName
	}
	newPath := filepath.Join(filepath.Dir(path), newName)
	if err := os.Rename(path, newPath); err != nil {
		log.Error(err.Error())
		return ErrReName
	}
	ctx.Set("newName", newName)
	return nil
}

func createFile(ctx *gin.Context, path string) *ControllerError {
	if utils.PathExists(path) {
		return ErrIsExist
	}
	createType := ctx.Query("create_type")
	switch createType {
	case "dir":
		if err := os.MkdirAll(path, 0644); err != nil {
			log.Error(err.Error())
			return ErrCreateDir
		}
	case "file":
		f, err := os.Create(path)
		if err != nil {
			log.Error(err.Error())
			return ErrCreateFile
		}
		f.Close()
	default:
		return ErrCreateType
	}
	return nil
}

func deleteFile(latexPath, path string) *ControllerError {
	if !utils.PathExists(path) {
		return ErrPath
	}
	if latexPath == path || latexPath == path[:len(path)-1] {
		return ErrPath
	}
	if err := localcommand.Rm(path); err != nil {
		log.Error(err.Error())
		return ErrRm
	}
	return nil
}

func downloadFile(ctx *gin.Context, path string) *ControllerError {
	if !utils.PathExists(path) {
		return ErrPath
	}
	items := strings.Split(path, "/")
	var fileName string
	if len(items) > 0 {
		fileName = items[len(items)-1]
	}
	ctx.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	ctx.Writer.Header().Add("Content-Type", "application/octet-stream")
	ctx.File(path)
	return nil
}

func uploadFile(ctx *gin.Context, path string) *ControllerError {
	fileInfo, err := os.Stat(path)
	if err != nil || !fileInfo.IsDir() {
		return ErrDirNotExist
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		return ErrUploadFile
	}
	fileName := file.Filename
	fileSize := file.Size
	totalSize, err := utils.GetDirSize(path)
	if err != nil {
		return ErrGetSize
	}
	if totalSize+fileSize > LatexDirMaxSize {
		return ErrSizeLimit
	}
	ctx.Set("filePath", path2.Join(ctx.Query("path"), fileName))
	absPath := path2.Join(path, fileName)
	if utils.PathExists(absPath) {
		ctx.Set("option", fileCover)
	} else {
		ctx.Set("option", fileUpload)
	}

	if err := ctx.SaveUploadedFile(file, absPath); err != nil {
		log.Error(err.Error())
		return ErrSaveFile
	}
	return nil
}
