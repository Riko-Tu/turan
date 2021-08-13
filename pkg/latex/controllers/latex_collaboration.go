package controllers

import (
	"TEFS-BE/pkg/database"
	latexServer "TEFS-BE/pkg/latex"
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/notifyContent"
	"TEFS-BE/pkg/tencentCloud/ses"
	"encoding/base64"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"math"
	"regexp"
	"strconv"
	"time"
)

const (
	urlEffectiveTime = 60 * 60 * 24 * 7 // url默认有效期1个星期
	statusPending = 1 // 待处理
	statusAccept = 2 // 接受
	statusReject = 3 // 拒绝
	statusCancel = 4 // 取消共享
	emailSubject = "文档协作分享"
	latexReadWritePowerText = "编辑"
	latexReadPowerText = "查看"
	defaultOffset = 0
	defaultConfigJson = `{"main_document":""}`

	// 腾讯云错误码 https://cloud.tencent.com/document/product/1288/51053
	frequencyLimitCode = 1006
	invalidEmailCode = 1007
	)

// @Summary 邮件发送latex协作邀请URL
// @Tags share
// @Security ApiKeyAuth
// @Description 邮件发送latex协作邀请URL
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Param targetEmail query string true "目标用户邮箱"
// @Param power query int true "权限：1=读写 2=只读"
// @Success 200 {string} json
// @Router /latex/{id}/share [POST]
func (c Controller) EmailLatexURL(ctx *gin.Context) {
	// 刷新用户token,获取用户信息
	newToken, user, latex, controllerError := getUserAndLatex(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}

	// 从url中获取latex文档ID
	latexId, err := conversionId(ctx.Param("id"))
	if err != nil {
		fail(ctx, ErrId, newToken)
		return
	}

	// 校验用户是否为该latex文档owner
	res, err := isLatexOwner(user.Id, latexId)
	if err != nil || !res {
		fail(ctx, ErrLatexOwner, newToken)
		return
	}

	// 从url参数中获取文档分享权限
	power, err := strconv.ParseInt(ctx.Query("power"), 10, 64)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrLatexPower, newToken)
		return
	}

	// 从url参数中获取目标邮箱
	targetEmail := ctx.Query("targetEmail")
	reg := "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"
	matched, err := regexp.MatchString(reg, targetEmail)
	if err != nil {
		//正则表达式错误
		log.Error(`regexp: Compile(` + reg + `): ` + err.Error())
		fail(ctx, ErrEmail, newToken)
		return
	}
	if !matched {
		fail(ctx, ErrEmail, newToken)
		return
	}

	// 协作者数量是否达到上限
	isLimit, controllerErr := checkCollaboratorsLimit(latex.Id)
	if controllerErr != nil {
		fail(ctx, controllerErr, newToken)
		return
	}
	if isLimit {
		fail(ctx, ErrShare, newToken)
		return
	}

	nowTime := time.Now().Unix()
	latexShare := &models.LatexShare{}
	latexShare.UserId = user.Id
	latexShare.LatexId = latexId
	latexShare.TargetEmail = targetEmail
	latexShare.Power = power
	latexShare.Status = statusPending
	latexShare.CreateAt = nowTime
	latexShare.UpdateAt = nowTime

	// 开启事务,如果发送邮件不成功，回滚
	db := database.GetDb()
	tx := db.Begin()
	opSuccess := false
	defer func() {
		if opSuccess {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	// 将记录写入latex_share表
	if err := tx.Create(latexShare).Error; err != nil {
		log.Error(err.Error())
		fail(ctx, ErrCreateDB, newToken)
		return
	}

	// 过期时间,默认一个星期
	exp := time.Now().Add(urlEffectiveTime * time.Second).Unix()
	key, err := CreateKey(latexShare.Id, exp)
	url := fmt.Sprintf(latexServer.UrlPattern, key, latexId)
	if err != nil {
		fail(ctx, ErrCreateUrl, newToken)
		return
	}
	log.Debug("create url success:" + url)

	// 发送URL到目标用户邮箱
	subject := emailSubject
	var powerText string
	if power == 1 {
		powerText = latexReadWritePowerText
	} else if power == 2 {
		powerText = latexReadPowerText
	}
	html := fmt.Sprintf(notifyContent.EmailCollaborateUrl, user.Email, powerText, url, latex.Name, url)
	if res, code := ses.SendEmailRespCode(targetEmail, "", html, subject); !res {
		switch code {
		case frequencyLimitCode:
			// 发送同一邮箱次数过多
			fail(ctx, ErrEmailLimit, newToken)
			return
		case invalidEmailCode:
			// 不存在此邮箱
			fail(ctx, ErrEmail, newToken)
			return
		default:
			// 发送邮件失败
			fail(ctx, ErrEmailService, newToken)
			return
		}
	}

	opSuccess = true

	resp(ctx, "ok", newToken)
	return
}

// @Summary 根据URL中的key获取latex进行协作
// @Tags share
// @Security ApiKeyAuth
// @Description 根据URL中的key获取latex进行协作
// @Accept  json
// @Produce  json
// @Param key query string true "分享latex的url中的key"
// @Success 200 {string} json
// @Router /share [POST]
func (c Controller) CollaborateLatex(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}

	key := ctx.Query("key")
	// 解析URL
	shareId, isExpired, err := ParseKey(key)
	if isExpired || err != nil {
		// 非法url或过期url
		fail(ctx, ErrUrl, newToken)
		return
	}

	// 开启事务
	db := database.GetDb()
	tx := db.Begin()
	opSuccess := false
	defer func() {
		if opSuccess {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	latexShare := &models.LatexShare{}
	// 行锁，锁定latex_share中分享记录（owner取消分享的同时用户打开url参与协作），同时根据shareId取出latex_share表文档分享记录
	if err := tx.Set(
		"gorm:query_option",
		"FOR UPDATE",
	).First(&latexShare, shareId).Error; err != nil {
		// 非法url，shareId不存在或已被使用
		log.Error(err.Error())
		fail(ctx, ErrUrl, newToken)
		return
	}

	// 判断分享链接是否可用
	if latexShare.Status != statusPending {
		fail(ctx, ErrUrl, newToken)
		return
	}

	// 协作者数量是否达到上限
	isLimit, controllerErr := checkCollaboratorsLimit(latexShare.LatexId)
	if controllerErr != nil {
		fail(ctx, controllerErr, newToken)
		return
	}
	if isLimit {
		fail(ctx, ErrShare, newToken)
		return
	}

	// 校验当前用户是否已在user_latex表中（包含latex owner检查）
	total, err := models.GetUserLatexList(user.Id, latexShare.LatexId)
	if err != nil {
		// 获取UserLatex记录错误
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}
	// user_latex表是否已经存在当前用户与latex文档关联记录
	if total > 0 {
		fail(ctx, ErrRecordIsExist, newToken)
		return
	}

	// 获取文档owner配置
	ownerLatex := &models.UserLatex{}
	if err = ownerLatex.GetForUserAndLatex(latexShare.UserId, latexShare.LatexId); err != nil {
		// 获取UserLatex记录错误
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}

	// 绑定latex和对应用户
	userLatex := &models.UserLatex{}
	nowTime := time.Now().Unix()
	userLatex.UserId = user.Id
	userLatex.LatexId = latexShare.LatexId
	userLatex.Power = latexShare.Power
	userLatex.Status = models.NormalStatus
	userLatex.ConfigJson = ownerLatex.ConfigJson
	userLatex.CreateAt = nowTime
	userLatex.UpdateAt = nowTime

	if latexShare.Status != statusPending {
		// url已被使用或已被取消分享
		fail(ctx, ErrUrl, newToken)
		return
	}

	// 更新user_share中状态，待处理->已接收
	latexShare.Status = statusAccept
	if err := tx.Model(latexShare).Update(latexShare).Error; err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, newToken)
		return
	}
	// 将记录写入user_latex表
	if err := tx.Set(
		"gorm:insert_option",
		fmt.Sprintf("ON DUPLICATE KEY UPDATE `delete_at`='0', update_at='%d', power='%d', status='%d'", nowTime, userLatex.Power, models.NormalStatus),
	).Create(userLatex).Error; err != nil {
		log.Error(err.Error())
		fail(ctx, ErrCreateDB, newToken)
		return
	}

	opSuccess = true
	log.Debug("collaborate success! get key:" + key)

	resp(ctx, "ok", newToken)
	return
}

// @Summary 查询文档分享URL待处理列表
// @Tags share
// @Security ApiKeyAuth
// @Description 根据latex_id查看当前latex文档邮件分享待处理情况
// @Accept  json
// @Produce  json
// @Param id query int64 true "latex_id"
// @Param offset query int64 true "从多少条开始"
// @Param limit query int64 true "返回的条数"
// @Success 200 {string} json
// @Router /share [GET]
func (c Controller) WaitingShareList(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}

	// 从请求中取出latexId
	latexId, err := conversionId(ctx.Query("id"))
	if err != nil {
		fail(ctx, ErrId, newToken)
		return
	}
	// 从latex表中读取
	latex := &models.Latex{}
	if err = latex.Get(latexId); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrLatexId, newToken)
		return
	}

	// 从请求出取出分页offset和limit
	offset, limit, controllerError := verifyOffsetLimit(ctx)
	if controllerError != nil {
		fail(ctx, controllerError, newToken)
		return
	}

	// 检查用户与latexId是否存在于user_latex表中, 是否有权限查看该信息
	userLatex := &models.UserLatex{}
	if err = userLatex.GetForUserAndLatex(user.Id, latexId); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrLatexId, newToken)
		return
	}

	// 获取待用户接受分享url记录
	waitingRecords, total, err := models.GetWaitingListByUserAndLatex(latex.Id, offset, limit)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrNotFoundRecord, newToken)
		return
	}

	data := make(map[string]interface{})
	data["total"] = total // 待定
	data["waiting"] = waitingRecords // 待处理的分享记录
	resp(ctx, data, newToken)
	return
}

// @Summary 查询文档协作者列表
// @Tags share
// @Security ApiKeyAuth
// @Description 根据latex_id查看当前latex文档协作者情况
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Success 200 {string} json
// @Router /latex/{id}/share [GET]
func (c Controller) CollaboratorsList(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}

	// 从请求中取出latex文档ID
	latexId, err := conversionId(ctx.Param("id"))
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrId, newToken)
		return
	}
	latex := &models.Latex{}
	if err = latex.Get(latexId); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrLatexId, newToken)
		return
	}

	// 检查用户与latexId是否存在于user_latex表中, 是否有权限查看该信息
	userLatex := &models.UserLatex{}
	if err = userLatex.GetForUserAndLatex(user.Id, latexId); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrLatexId, newToken)
		return
	}

	// 协作者有数量限制，在配置文件中默认为10，因此设置默认值offset为0，limit为0
	offset, limit := int64(defaultOffset), latexServer.ShareLimit

	// 获取文档owner
	owner, err := models.GetLatexOwner(latexId, latex.UserId)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}

	// 获取当前协作者
	collaborators, _, err := models.GetLatexCollaboratorList(latexId, latex.UserId, offset, limit)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrNotFoundRecord, newToken)
		return
	}

	data := make(map[string]interface{})
	data["owner"] = owner // latexOwner
	data["collaborators"] = collaborators // user_latex中该latex文档的记录
	resp(ctx, data, newToken)
	return
}

// @Summary 删除文档分享URL，取消分享
// @Tags share
// @Security ApiKeyAuth
// @Description 根据share_id删除当前文档分享URL
// @Accept  json
// @Produce  json
// @Param id path int64 true "share ID"
// @Success 200 {string} json
// @Router /share/{id} [DELETE]
func (c Controller) CancelShareUrl(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		fail(ctx, ErrToken, newToken)
		return
	}

	// 从路径中获取shareId
	shareId, err := conversionId(ctx.Param("id"))
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrId, newToken)
		return
	}

	latexShare := &models.LatexShare{}
	if err = latexShare.Get(shareId); err != nil {
		// ErrRecordNotFound
		log.Error(err.Error())
		fail(ctx, ErrId, newToken)
		return
	}

	if user.Id != latexShare.UserId {
		// 文档share记录不是当前用户创建
		fail(ctx, ErrLatexOwner, newToken)
		return
	}

	ups := map[string]interface{}{"delete_at": time.Now().Unix()}
	if err = latexShare.TxUpdate(ups, shareId, statusCancel); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	resp(ctx, "ok", newToken)
	return
}

// @Summary 修改文档协作者权限
// @Tags share
// @Security ApiKeyAuth
// @Description 根据user_id和latex_id修改文档协作者权限
// @Accept  json
// @Produce  json
// @Param user_id query int64 true "user_id"
// @Param id path int64 true "latex_id"
// @Param power query int64 true "权限 1=读写 2=只读"
// @Success 200 {string} json
// @Router /latex/{id}/share [PUT]
func (c Controller) EditCollaboratorPower(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrToken, newToken)
		return
	}

	// 从路径中获取latexId和userId
	latexId, targetUserId, err := getUserIdAndLatexId(ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrId, newToken)
		return
	}

	// 从路径中获取权限power
	power, err := conversionId(ctx.Query("power"))
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrLatexPower, newToken)
		return
	}

	// 校验当前用户是否为文档owner
	res, err := isLatexOwner(user.Id, latexId)
	// user.Id == targetUserId 校验要修改的目标用户是是否为自己
	if err != nil || !res || user.Id == targetUserId {
		fail(ctx, ErrLatexOwner, newToken)
		return
	}

	userLatex := &models.UserLatex{}
	err = userLatex.GetForUserAndLatex(targetUserId, latexId)
	if err != nil {
		// ErrRecordNotFound
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}

	// 更新用户文档权限
	ups := map[string]interface{}{"power": power}
	err = userLatex.Update(ups)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	resp(ctx, "ok", newToken)
	return
}

// @Summary 移除文档协作者
// @Tags share
// @Security ApiKeyAuth
// @Description 根据user_id和latex_id移除文档协作者
// @Accept  json
// @Produce  json
// @Param id path int64 true "latex ID"
// @Param user_id query int64 true "user ID"
// @Success 200 {string} json
// @Router /latex/{id}/share [DELETE]
func (c Controller) DeleteCollaborator(ctx *gin.Context) {
	user, newToken, err := getUserForToken(ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrToken, newToken)
		return
	}

	// 从路径中获取latexId和userId
	latexId, deleteCollaboratorId, err := getUserIdAndLatexId(ctx)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrId, newToken)
		return
	}

	// 校验当前用户是否为文档owner
	res, err := isLatexOwner(user.Id, latexId)
	// user.Id != deleteCollaboratorId 校验owner不能移除自己
	if err != nil || !res || user.Id == deleteCollaboratorId {
		fail(ctx, ErrLatexOwner, newToken)
		return
	}

	userLatex := &models.UserLatex{}
	err = userLatex.GetForUserAndLatex(deleteCollaboratorId, latexId)
	if err != nil {
		// ErrRecordNotFound
		log.Error(err.Error())
		fail(ctx, ErrQueryDB, newToken)
		return
	}

	// 删除用户
	ups := map[string]interface{}{"delete_at": time.Now().Unix()}
	err = userLatex.Update(ups)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	resp(ctx, "ok", newToken)
	return
}

// 创建分享url的key，封装latex_share表中记录的id，过期时间exp
func CreateKey(shareId int64, exp int64) (string, error) {
	latexToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"shareId": shareId,
		"exp":     exp,
	})
	key, err := latexToken.SignedString([]byte(latexServer.Secret))
	if err != nil {
		return "", err
	}

	key = base64.StdEncoding.EncodeToString([]byte(key))
	return key, nil
}

// 解析分享的url中的key, 解析latex_share表中记录的id，过期时间exp
func ParseKey(key string) (shareId int64, isExpired bool, err error) {
	byteKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		err = fmt.Errorf("invalid url")
		return
	}

	claim, err := jwt.Parse(string(byteKey), func(token *jwt.Token) (interface{}, error) {
		return []byte(latexServer.Secret), nil
	})

	var urlIsInvalid bool
	if err != nil {
		// url是否过期
		isExpired = true
		if err.Error() == "Token is expired" {
			urlIsInvalid = false
		} else {
			urlIsInvalid = true
		}
	}
	if urlIsInvalid || claim == nil {
		err = fmt.Errorf("invalid url")
		return
	}

	// 解析jwt token, 含shareId和exp过期时间
	jwtClaims := claim.Claims.(jwt.MapClaims)
	shareId = int64(jwtClaims["shareId"].(float64) * math.Pow10(0))
	// 将时间戳转化为time.Time结构
	expireTime := time.Unix(int64(jwtClaims["exp"].(float64) * math.Pow10(0)), 0)
	// 判断过期时间
	isExpired = time.Now().After(expireTime)

	return shareId, isExpired, nil
}

func isLatexOwner(userId, latexId int64) (res bool, err error) {
	latex := &models.Latex{}
	if err = latex.Get(latexId); err != nil {
		err = fmt.Errorf("latex id error")
		return
	}
	// 判断当前操作的用户是否为owner
	res = userId == latex.UserId
	return
}

// 获取参数中的latex_id和user_id
func getUserIdAndLatexId(ctx *gin.Context)(latexId, userId int64, err error) {
	latexId, err = conversionId(ctx.Param("id"))
	if err != nil {
		return
	}
	userId, err = conversionId(ctx.Query("user_id"))
	if err != nil {
		return
	}
	return
}

func checkCollaboratorsLimit(latexId int64) (isLimit bool, controllerError *ControllerError){
	// 判断分享是否已经达到上限
	collaboratorsCount, err := models.GetAllLatexCollaboratorsTotal(latexId)
	if err != nil {
		log.Error(err.Error())
		controllerError = ErrQueryDB
		return
	}
	if collaboratorsCount >= latexServer.ShareLimit {
		// 达到上限
		isLimit = true
	}
	return
}