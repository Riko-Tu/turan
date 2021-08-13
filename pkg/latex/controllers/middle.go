package controllers

import (
	"TEFS-BE/pkg/latex/history"
	"TEFS-BE/pkg/log"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"time"
)

func ChangeHandle(ctx *gin.Context, operation string, document map[string]string) {
	userId := ctx.GetInt64("userId")
	latexId := ctx.GetInt64("latexId")
	fileChange := history.Change{
		Document:  document,
		Time:      time.Now().Unix(),
		Transform: history.Transform{},
		UserId:    userId,
		LatexId:   latexId,
		Operation: operation,
	}
	if err := fileChange.FileChangeToCache(); err != nil {
		data, _ := json.Marshal(fileChange)
		errMsg := fmt.Sprintf("file change to redis failed. data:%s, err:%s", string(data), err.Error())
		log.Error(errMsg)
	}
}

func FileChangeMiddle(ctx *gin.Context) {
	ctx.Next()
	option := ctx.Query("option")
	if option == fileDownload {
		return
	}
	if ctx.GetBool(operationIsSuccess) != true {
		return
	}

	document := make(map[string]string)
	document["id"] = ctx.GetString("filePath")
	document["fileType"] = ctx.GetString("fileType")
	if option == fileRename {
		document["new_name"] = ctx.GetString("newName")
	}
	if option == fileUpload {
		option = ctx.GetString("option")
	}
	ChangeHandle(ctx, option, document)
}

func FileMVMiddle(ctx *gin.Context) {
	ctx.Next()
	if ctx.GetBool(operationIsSuccess) != true {
		return
	}
	document := make(map[string]string)
	document["id"] = ctx.GetString("filePath")
	document["fileType"] = ctx.GetString("fileType")
	document["targetPath"] = ctx.GetString("targetPath")
	ChangeHandle(ctx, "move", document)
}
