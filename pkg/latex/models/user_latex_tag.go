package models

import (
	"TEFS-BE/pkg/database"
	"fmt"
	"time"
)

type UserLatexTag struct {
	Id       int64
	UserId   int64
	Name     string
	CreateAt int64
	UpdateAt int64
	DeleteAt int64
}

func (UserLatexTag) TableName() string {
	return "user_latex_tag"
}

func (ult *UserLatexTag) Create() error {
	db := database.GetDb()
	return db.Create(ult).Error
}

func (ult UserLatexTag) Update(ups map[string]interface{}) error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(ult).Update(ups).Error
}

func (ult UserLatexTag) Delete() error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups := make(map[string]interface{})
	ups["update_at"] = nowTime
	ups["delete_at"] = nowTime
	return db.Model(ult).Update(ups).Error
}

func (ult *UserLatexTag) Get(id int64) error {
	db := database.GetDb()
	return db.Where("id = ? AND delete_at = 0", id).First(ult).Error
}

func (ult *UserLatexTag) GetByUserAndName(userId int64, name string) error {
	db := database.GetDb()
	err := db.Where("user_id = ? AND name = ? AND delete_at=0", userId, name).First(ult).Error
	if err == database.NotFoundErr {
		err = nil
	}
	return err
}

func GetUserLatexTags(userId, offset, limit int64, sort, sortField string) (UserLatexTags []UserLatexTag, total int64, err error) {
	db := database.GetDb()
	query := "user_id=? AND delete_at=0"
	err = db.Model(&UserLatexTag{}).Where(query, userId).Count(&total).Error
	if err != nil {
		return
	}
	order := fmt.Sprintf("%s %s", sortField, sort)
	err = db.Where(query, userId).Offset(offset).Limit(limit).Order(order).Find(&UserLatexTags).Error
	return
}
