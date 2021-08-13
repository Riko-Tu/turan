package models

import (
	"TEFS-BE/pkg/database"
	"fmt"
	"time"
)

type UserLatex struct {
	Id         int64
	UserId     int64
	LatexId    int64
	Status     int64
	Power      int64
	ConfigJson string
	CreateAt   int64
	UpdateAt   int64
	DeleteAt   int64
}

type LatexData struct {
	LatexId                  int64  `json:"latex_id"`
	LatexName                string `json:"latex_name"`
	LatexIsCurrentUser       bool   `json:"latex_is_current_user"`
	UserLatexId              int64  `json:"user_latex_id"`
	CreateUserId             int64  `json:"create_user_id"`
	CreateUserName           string `json:"create_user_name"`
	CreateUserEmail          string `json:"create_user_email"`
	LastModifiedUserId       int64  `json:"last_modified_user_id"`
	LastModifiedUserName     string `json:"last_modified_user_name"`
	Status                   int64  `json:"status"`
	Power                    int64  `json:"power"`
	Tag                      string `json:"tag"`
	CreateAt                 int64  `json:"create_at"`
	UpdateAt                 int64  `json:"update_at"`
	CurrentUserOperationTime int64  `json:"current_user_operation_time"`
}

type LatexCollaborators struct {
	LatexId  int64  `json:"latex_id"`
	UserId   int64  `json:"user_id"`
	Name     string `json:"user_name"`
	Email    string `json:"user_email"`
	Phone    string `json:"user_phone"`
	Status   int64  `json:"status"`
	Power    int64  `json:"power"`
	CreateAt int64  `json:"join_at"`
	UpdateAt int64  `json:"update_at"`
}

func (UserLatex) TableName() string {
	return "user_latex"
}

func (ul *UserLatex) Create() error {
	db := database.GetDb()
	return db.Create(ul).Error
}

func (ul UserLatex) Update(ups map[string]interface{}) error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(ul).Update(ups).Error
}

func (ul UserLatex) Delete() error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups := make(map[string]interface{})
	ups["update_at"] = nowTime
	ups["delete_at"] = nowTime
	return db.Model(ul).Update(ups).Error
}

func (ul *UserLatex) Get(id int64) error {
	db := database.GetDb()
	return db.Where("id = ? AND delete_at = 0", id).First(ul).Error
}

func (ul *UserLatex) GetForUserAndLatex(userId, latexId int64) error {
	db := database.GetDb()
	// 使用First,查询不到记录会出现ErrRecordNotFound，用Find可避免
	return db.Where("user_id = ? AND latex_id = ? AND delete_at = 0", userId, latexId).First(ul).Error
}

func GetLatexCount(userId, createUserId, latexId int64) (total int64, err error) {
	db := database.GetDb()
	query := "SELECT COUNT(id) as count FROM (SELECT id FROM latex WHERE user_id=? AND id=? AND delete_at=0) AS t1 INNER JOIN (SELECT latex_id FROM user_latex WHERE user_id=? AND latex_id=? AND delete_at=0) as t2 ON t1.id=t2.latex_id"
	tmp := struct {
		Count int64
	}{}
	err = db.Raw(query, createUserId, latexId, userId, latexId).Scan(&tmp).Error
	total = tmp.Count
	return
}

func GetUserLatexRecordTotal(userId, latexId int64) (total int64, err error) {
	db := database.GetDb()
	query := "SELECT COUNT(*) as count FROM (SELECT latex_id FROM user_latex WHERE user_id=? AND latex_id=? AND delete_at=0) as t1  RIGHT JOIN (SELECT id FROM latex WHERE id=? AND delete_at=0) as t2 ON t1.latex_id = t2.id"
	tmp := struct {
		Count int64
	}{}
	err = db.Raw(query, userId, latexId, latexId).Scan(&tmp).Error
	total = tmp.Count
	return
}

func GetLatexList(userId, offset, limit, status int64, sort, sortField string, latexId *int64) (records []*LatexData, total int64, err error) {
	db := database.GetDb()
	totalQuery := "SELECT COUNT(latex_id) as total FROM (SELECT latex_id,`status`, power from user_latex WHERE user_id=? %s AND delete_at=0) as ul INNER JOIN latex ON ul.latex_id = latex.id WHERE latex.delete_at=0"
	latexListQuery := "select l.*, GROUP_CONCAT(tag.id, ' ', tag.name) AS tag from (select tmp.*, `user`.`name` as create_user_name, user_tmp.`name` as last_modified_user_name, `user`.email as create_user_email from (SELECT ul.*,latex.user_id as create_user_id, latex.`name` as latex_name, latex.last_modified_user_id, latex.create_at, latex.update_at FROM (SELECT id AS user_latex_id, latex_id,`status`, power, update_at AS current_user_operation_time from user_latex WHERE user_id=? %s AND delete_at=0) as ul INNER JOIN latex ON ul.latex_id = latex.id WHERE %s latex.delete_at=0) as tmp LEFT JOIN `user` ON tmp.create_user_id=`user`.id LEFT JOIN `user` as user_tmp ON tmp.last_modified_user_id=user_tmp.id) as l LEFT JOIN (SELECT t1.id, t1.name, t2.latex_id from (SELECT * from user_latex_tag WHERE user_id=? AND delete_at=0) as t1 INNER JOIN (SELECT * from latex_tag WHERE user_id=? AND delete_at=0) as t2 ON t1.id = t2.user_latex_tag_id) as tag ON l.latex_id=tag.latex_id GROUP BY latex_id"
	//latexIdQuery := "select l.*, GROUP_CONCAT(tag.id, ' ', tag.name) AS tag from (select tmp.*, `user`.`name` as create_user_name, user_tmp.`name` as last_modified_user_name, `user`.email as create_user_email from (SELECT ul.*,latex.user_id as create_user_id, latex.`name` as latex_name, latex.last_modified_user_id, latex.create_at, latex.update_at FROM (SELECT id AS user_latex_id, latex_id,`status`, power from user_latex WHERE user_id=? %s AND delete_at=0) as ul INNER JOIN latex ON ul.latex_id = latex.id WHERE latex.delete_at=0) as tmp LEFT JOIN `user` ON tmp.create_user_id=`user`.id LEFT JOIN `user` as user_tmp ON tmp.last_modified_user_id=user_tmp.id) as l LEFT JOIN (SELECT t1.id, t1.name, t2.latex_id from (SELECT * from user_latex_tag WHERE user_id=? AND delete_at=0) as t1 INNER JOIN (SELECT * from latex_tag WHERE user_id=? AND delete_at=0) as t2 ON t1.id = t2.user_latex_tag_id) as tag ON l.latex_id=tag.latex_id GROUP BY latex_id"

	if status < 0 || status > 3 {
		err = fmt.Errorf("not found status:%d", status)
		return

	}
	var whereStatus string
	switch status {
	case 0:
		whereStatus = ""
	default:
		whereStatus = fmt.Sprintf(" AND status=%d", status)
	}

	latexIdOption := ""
	if latexId != nil {
		latexIdOption = " latex.id=? AND"
	}

	totalQuery = fmt.Sprintf(totalQuery, whereStatus)
	latexListQuery = fmt.Sprintf(latexListQuery, whereStatus, latexIdOption)
	if err = db.Raw(totalQuery, userId).Count(&total).Error; err != nil {
		return
	}
	order := fmt.Sprintf("%s %s", sortField, sort)

	if latexId == nil {
		err = db.Raw(latexListQuery, userId, userId, userId).Offset(offset).Limit(limit).Order(order).Scan(&records).Error
	} else {
		err = db.Raw(latexListQuery, userId, latexId, userId, userId).Offset(offset).Limit(limit).Order(order).Scan(&records).Error
	}
	return
}

func GetUserLatexTotal(userId int64, userLatexIds []string, isJoinLatex bool) (total int, err error) {
	db := database.GetDb()
	var query string
	var params []interface{}
	params = append(params, userId)
	for _, v := range userLatexIds {
		params = append(params, v)
	}
	if !isJoinLatex {
		query = "SELECT COUNT(id) AS count FROM user_latex WHERE user_id=? AND delete_at=0 AND id IN (" + genVar(userLatexIds) + ")"
	} else {
		query = "SELECT COUNT(latex.id) AS count FROM (SELECT id, latex_id FROM user_latex WHERE user_id=? AND delete_at=0 AND id IN (" + genVar(userLatexIds) + ")) AS t1 INNER JOIN latex ON t1.latex_id=latex.id WHERE latex.user_id !=? AND delete_at=0"
		params = append(params, userId)
	}
	tmp := struct {
		Count int
	}{}
	if err = db.Raw(query, params...).Scan(&tmp).Error; err != nil {
		return
	}
	total = tmp.Count
	return
}

const NormalStatus int64 = 1  // 正常状态
const ArchiveStatus int64 = 2 // 归档状态
const DeleteStatus int64 = 3  // 删除状态（逻辑删除，丢弃到垃圾桶）

func genVar(ids []string) string {
	inCondition := ""
	for range ids {
		if inCondition != "" {
			inCondition += ", "
		}
		inCondition += "?"
	}
	return inCondition
}

func UpdateUserLatexListStatus(userId, status int64, userLatexIds []string) (err error) {
	db := database.GetDb()
	tx := db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	updateAt := time.Now().Unix()
	updateSql := "UPDATE user_latex SET `status`=?, `update_at`=? WHERE user_id=? AND delete_at=0 AND id in (" + genVar(userLatexIds) + ")"
	var params []interface{}
	params = append(params, status, updateAt, userId)
	for _, v := range userLatexIds {
		params = append(params, v)
	}
	ret := tx.Exec(updateSql, params...)
	err = ret.Error
	rowsAffected := ret.RowsAffected
	if rowsAffected != int64(len(userLatexIds)) {
		err = fmt.Errorf("update row count error")
	}
	return
}

func TxDeleteLatexList(userId int64, latexIds, userLatexIds []string) (err error) {
	db := database.GetDb()
	tx := db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	deleteAt := time.Now().Unix()
	updateUserLatexSql := "UPDATE user_latex SET `delete_at`=? WHERE user_id=? AND delete_at=0 AND id in (" + genVar(userLatexIds) + ")"
	updateLatexSql := "UPDATE latex SET `delete_at`=? WHERE user_id=? AND delete_at=0 AND id in (" + genVar(latexIds) + ")"

	latexIdsLen := len(latexIds)
	if latexIdsLen > 0 {
		var params []interface{}
		params = append(params, deleteAt, userId)
		for _, v := range latexIds {
			params = append(params, v)
		}
		ret := tx.Exec(updateLatexSql, params...)
		err = ret.Error
		rowsAffected := ret.RowsAffected
		if rowsAffected != int64(latexIdsLen) {
			err = fmt.Errorf("update latex rows affected error")
		}
		if err != nil {
			return
		}
	}

	userLatexIdsLen := len(userLatexIds)
	if userLatexIdsLen > 0 {
		var params []interface{}
		params = append(params, deleteAt, userId)
		for _, v := range userLatexIds {
			params = append(params, v)
		}
		ret := tx.Exec(updateUserLatexSql, params...)
		err = ret.Error
		rowsAffected := ret.RowsAffected
		if rowsAffected != int64(userLatexIdsLen) {
			err = fmt.Errorf("update user_latex rows affected error")
		}
		if err != nil {
			return
		}
	}
	return
}

func GetLatexOwner(latexId, ownerId int64) (owner LatexCollaborators, err error) {
	db := database.GetDb()
	fieldsSelect := "SELECT ul.latex_id, ul.user_id, ul.status, ul.power, ul.create_at, ul.update_at, user.name, user.email, user.phone "
	query := "FROM user_latex as ul " +
		"LEFT JOIN user " +
		"ON user.id = ul.user_id " +
		"WHERE ul.latex_id=? AND ul.user_id=? AND ul.delete_at=0"
	if err = db.Raw(fieldsSelect+query, latexId, ownerId).Scan(&owner).Error; err != nil {
		return
	}
	return
}

func GetAllLatexCollaboratorsTotal(latexId int64) (total int64, err error) {
	db := database.GetDb()
	// 获取latex文档现有协作者总数(加上owner)，不考虑权限power和status状态（考虑到删除状态的文档在回收站，用户依然可以恢复，故仍计算为协作者）
	query := "latex_id=? AND delete_at=0"
	if err = db.Model(&UserLatex{}).Where(query, latexId).Count(&total).Error; err != nil {
		return
	}
	return
}

func GetLatexCollaboratorList(latexId, ownerId, offset, limit int64) (records []LatexCollaborators, total int64, err error) {
	db := database.GetDb()
	// 查询拥有当前文档的用户记录(owner除外)
	totalSelect := "SELECT COUNT(*) "
	fieldsSelect := "SELECT ul.latex_id, ul.user_id, ul.status, ul.power, ul.create_at, ul.update_at, user.name, user.email, user.phone "
	query := "FROM user_latex as ul " +
		"LEFT JOIN user " +
		"ON user.id = ul.user_id " +
		"WHERE ul.latex_id=? AND ul.user_id!=? AND ul.delete_at=0"

	if err = db.Raw(totalSelect+query, latexId, ownerId).Count(&total).Error; err != nil {
		return
	}
	if err = db.Raw(fieldsSelect+query, latexId, ownerId).Offset(offset).Limit(limit).Scan(&records).Error; err != nil {
		return
	}

	return
}

func GetUserLatexList(userId, latexId int64) (total int64, err error) {
	db := database.GetDb()
	// 查询userId和latexId是否共同出现在user_latex表中
	query := "SELECT COUNT(*) as count " +
		"FROM (SELECT latex_id FROM user_latex WHERE user_id=? AND latex_id=? AND delete_at=0) as t1  " +
		"INNER JOIN (SELECT id FROM latex WHERE id=? AND delete_at=0) as t2 " +
		"ON t1.latex_id = t2.id"
	tmp := struct {
		Count int64
	}{}
	err = db.Raw(query, userId, latexId, latexId).Scan(&tmp).Error
	total = tmp.Count
	return
}
