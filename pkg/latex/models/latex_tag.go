package models

import (
	"TEFS-BE/pkg/database"
	"time"
)

type LatexTag struct {
	Id             int64
	UserId         int64
	LatexId        int64
	UserLatexTagId int64
	CreateAt       int64
	DeleteAt       int64
}

func (LatexTag) TableName() string {
	return "latex_tag"
}

func (l *LatexTag) Create() error {
	db := database.GetDb()
	return db.Create(l).Error
}

func (l LatexTag) Update(ups map[string]interface{}) error {
	db := database.GetDb()
	return db.Model(l).Update(ups).Error
}

func (l LatexTag) Delete() error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups := make(map[string]interface{})
	ups["update_at"] = nowTime
	ups["delete_at"] = nowTime
	return db.Model(l).Update(ups).Error
}

func (l *LatexTag) Get(id int64) error {
	db := database.GetDb()
	return db.Where("id = ? AND delete_at = 0", id).First(l).Error
}

func (l *LatexTag) GetForUserTagLatex(userId, tagId, latexId int64) error {
	db := database.GetDb()
	err := db.Where("user_id=? AND latex_id=? AND user_latex_tag_id=? AND delete_at = 0", userId, latexId, tagId).First(l).Error
	if err == database.NotFoundErr {
		err = nil
	}
	return err
}