package controllers

import (
	"TEFS-BE/pkg/admin/model"
	admin "TEFS-BE/pkg/admin/service"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// api controller
type Controller struct {
}

// api controller error
type ControllerError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func resp(c *gin.Context, data interface{}, newToken string) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": data,
		"newToken": newToken,
	})
}

func fail(c *gin.Context, errs *ControllerError, newToken string) {
	c.JSON(http.StatusOK, gin.H{
		"code": errs.Code,
		"msg":  errs.Message,
		"newToken": newToken,
	})
}

// database id str to int64
func conversionId(idStr string) (id int64, err error) {
	id, err = strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return
	}
	if id <= 0 {
		err = fmt.Errorf("id must be greater than 0")
		return
	}
	return
}

// get user model
// If the token is expired, a new token is generated
func getUserForToken(ctx *gin.Context) (user *model.User, newToken string, err error) {
	token := ctx.GetHeader("authorization")
	user, newToken, err = admin.HandleToken(token)
	if err != nil {
		return
	}
	if user.Id == 0 {
		err = fmt.Errorf("not found user")
		return
	}
	return
}

// verify params offset limit sort
func verifyOffsetLimitSort(ctx *gin.Context) (offset,limit int64, sort string, controllerError *ControllerError) {
	offset, limit, controllerError = verifyOffsetLimit(ctx)
	if controllerError != nil {
		return
	}

	sort = ctx.Query("sort")
	if sort != "asc" && sort != "desc" {
		controllerError = ErrSort
		return
	}
	return
}

func verifyOffsetLimit(ctx *gin.Context) (offset,limit int64, controllerError *ControllerError) {
	offset, err := strconv.ParseInt(ctx.Query("offset"), 10, 64)
	if err != nil {
		controllerError = ErrOffset
		return
	}
	limit, err = strconv.ParseInt(ctx.Query("limit"), 10, 64)
	if err != nil {
		controllerError = ErrLimit
		return
	}
	return
}

var (
	SUCCESS = &ControllerError{Code:200, Message:"success"}
	ErrToken = &ControllerError{Code:3001, Message:"token err"}
	ErrId = &ControllerError{Code:3002, Message:"id err"}
	ErrNotFoundRecord = &ControllerError{Code:3003, Message:"not found record"}
	ErrNotFoundLatexRecord = &ControllerError{Code:3004, Message:"not found latex record"}
	ErrNotFoundTagRecord = &ControllerError{Code:3005, Message:"not found tag record"}
	ErrLatexPower = &ControllerError{Code:3006, Message:"power err"} // 权限
	ErrLatexId = &ControllerError{Code:3007, Message:"latex_id err"} // 文档id错误
	ErrUrl = &ControllerError{Code:3008, Message:"invalid url or expired url"} // url token解析失败
	ErrLatexOwner = &ControllerError{Code:3009, Message:"owner update records error"} // 不是文档owner
	ErrEmail = &ControllerError{Code:3010, Message:"invalid email"}  // 邮箱正则校验错误
	ErrLatexCollaborator = &ControllerError{Code:3011, Message:"owner collaborate latex error"}
	ErrShare = &ControllerError{Code:3012, Message:"latex share limit"} // 文档协作者上限
	ErrLatexName = &ControllerError{Code:3013, Message:"latex name err"}
	ErrLatexTagName = &ControllerError{Code:3014, Message:"latex tag name err"}
	ErrCreateLatex = &ControllerError{Code:3015, Message:"create latex err"}
	ErrGetTree = &ControllerError{Code:3016, Message:"get tree err"}
	ErrLatexStatus = &ControllerError{Code:3017, Message:"latex status error"}
	ErrOffset = &ControllerError{Code:3018, Message:"offset error"}
	ErrCompile = &ControllerError{Code:3019, Message:"latex compile failed"}
	ErrLimit = &ControllerError{Code:3020, Message:"limit error"}
	ErrSortField = &ControllerError{Code:3021, Message:"not found sort field"}
	ErrSort = &ControllerError{Code:3022, Message:"sort error"}
	ErrQueryDB = &ControllerError{Code:3023, Message:"query db record error"}
	ErrUpdateDB = &ControllerError{Code:3024, Message:"update db record error"}
	ErrCreateDB = &ControllerError{Code:3025, Message:"create db record error"}
	ErrUploadFile = &ControllerError{Code:3026, Message:"upload file error"}
	ErrPath = &ControllerError{Code:3027, Message:"path error"}
	ErrSaveFile = &ControllerError{Code:3028, Message:"save file error"}
	ErrGetSize = &ControllerError{Code:3029, Message:"get size error"}
	ErrSizeLimit = &ControllerError{Code:3030, Message:"file size limit 100M"}
	ErrNotFoundOption = &ControllerError{Code:3031, Message:"not found option"}
	ErrRm = &ControllerError{Code:3031, Message:"rm error"}
	ErrIsExist = &ControllerError{Code:3033, Message:"file or dir is exist"}
	ErrCreateFile = &ControllerError{Code:3034, Message:"create file err"}
	ErrCreateDir = &ControllerError{Code:3035, Message:"create dir err"}
	ErrCreateType = &ControllerError{Code:3036, Message:"create type err"}
	ErrNewName = &ControllerError{Code:3037, Message:"new name format err"}
	ErrReName = &ControllerError{Code:3038, Message:"rename err"}
	ErrDirNotExist = &ControllerError{Code:3039, Message:"dir is not exist"}
	ErrDeleteLatex = &ControllerError{Code:3040, Message:"delete latex failed"}
	ErrLatexTagIsExist = &ControllerError{Code:3041, Message:"latex tag is exist"}
	ErrLatexTagNameRepetition = &ControllerError{Code:3042, Message:"latex tag name repetition"}
	ErrRecordIsExist = &ControllerError{Code:3043, Message:"record is exist"}
	ErrCreateWebsocket = &ControllerError{Code:3044, Message:"create websocket err"}
	ErrCurator = &ControllerError{Code:3045, Message:"create curator err"}
	ErrCreateUrl = &ControllerError{Code:3046, Message:"create url failed"}
	ErrIds = &ControllerError{Code:3047, Message:""}
	ErrRsaDecrypt = &ControllerError{Code:3048, Message:"rsa decrypt failed"}
	ErrNotFoundToken = &ControllerError{Code:3049, Message:"not found token"}
	ErrParam = &ControllerError{Code:3050, Message:"invalid param"}
	ErrUploadZip = &ControllerError{Code:3051, Message:"upload zip failed"}
	ErrGetLatexZip = &ControllerError{Code:3052, Message:"get zip failed"}
	ErrCompress = &ControllerError{Code:3053, Message:"compress failed"}
	ErrNotAuthority = &ControllerError{Code:3054, Message:"not authority"}
	ErrEmailLimit = &ControllerError{Code:3055, Message:"request same email limit"} // 发送同一邮箱太多次，腾讯云SES服务拒绝
	ErrEmailService = &ControllerError{Code:3056, Message: "SES service error"} // 可能是不存在邮箱，第一次发送成功，邮件被退回，第二发送重复邮箱，腾讯云SES服务拒绝
	ErrTemplateName = &ControllerError{Code:3057, Message: "latex name It can only contain Chinese characters, upper and lower case letters, numbers and characters-_, and len limit 1-50"}
	ErrConfigFormat = &ControllerError{Code:3058, Message:"latex config format err"}
	ErrSaveImage = &ControllerError{Code:3059, Message:"latex template image save err"}
	ErrMV = &ControllerError{Code:3060, Message:"latex file or dir mv err"}
	ErrLatexGitInit = &ControllerError{Code:3061, Message:"get latex git init err"}
)