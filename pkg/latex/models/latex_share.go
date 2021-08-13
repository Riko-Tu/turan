package models

import (
	"TEFS-BE/pkg/database"
	"time"
)

type LatexShare struct {
	Id          int64
	UserId      int64
	LatexId     int64
	Power       int64
	Status      int64
	CreateAt    int64
	UpdateAt    int64
	DeleteAt    int64
	TargetEmail string
}

func (LatexShare) TableName() string {
	return "latex_share"
}

func (l *LatexShare) Create() error {
	db := database.GetDb()
	return db.Create(l).Error
}

func (l LatexShare) Update(ups map[string]interface{}) error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(l).Update(ups).Error
}

func (l LatexShare) Delete() error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups := make(map[string]interface{})
	ups["update_at"] = nowTime
	ups["delete_at"] = nowTime
	return db.Model(l).Update(ups).Error
}

func (l *LatexShare) Get(id int64) error {
	db := database.GetDb()
	return db.Where("id = ? AND delete_at = 0", id).First(l).Error
}

func (l *LatexShare)TxUpdate(ups map[string]interface{}, shareId, status int64) (err error) {
	// 开启事务
	db := database.GetDb()
	tx := db.Begin()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	ups["status"] = status

	// 行锁，锁定latex_share中分享记录（owner取消分享的同时用户打开url参与协作），同时根据shareId取出latex_share表文档分享记录
	if err = tx.Set("gorm:query_option", "FOR UPDATE").First(l, shareId).Error; err != nil {
		// 非法url，shareId不存在
		tx.Rollback()
		return
	}

	// 更新操作
	if err = tx.Model(l).Update(ups).Error; err != nil {
		tx.Rollback()
		return
	}

	// 事务提交，释放锁
	tx.Commit()
	return
}

func GetListByUserAndLatex(userId, latexId, offset, limit int64) (records []LatexShare, total int64, err error) {
	db := database.GetDb()
	query := "SELECT * FROM latex_share WHERE user_id=? AND latex_id=? AND delete_at=0"
	if err = db.Model(&LatexShare{}).Where(query, userId, latexId).Count(&total).Error; err != nil {
		return
	}
	err = db.Where(query, userId, latexId).Offset(offset).Limit(limit).Find(records).Error
	return
}

func GetWaitingListByUserAndLatex(latexId, offset, limit int64) (records []LatexShare, total int64, err error) {
	db := database.GetDb()

	// 状态为待处理(status=1)的分享记录总数，不与userId一起查询是因为文档owner可能会转让
	query := "latex_id=? AND status=1 AND delete_at=0"
	total,err = GetWaitingListRecordsTotal(latexId)
	if err != nil {
		return
	}

	// 状态为待处理(status=1)的分享记录，根据offset，limit查询
	if err = db.Where(query, latexId).Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return
	}
	return
}

func GetWaitingListRecordsTotal(latexId int64) (total int64, err error) {
	db := database.GetDb()
	// 根据latexId获取share记录，不与userId一起查询是因为文档owner可能会转让
	query := "latex_id=? AND status=1 AND delete_at=0"
	if err = db.Model(&LatexShare{}).Where(query, latexId).Count(&total).Error; err != nil {
		return
	}
	return
}