package models

import (
	"TEFS-BE/pkg/database"
	"time"
)

type LatexTemplate struct {
	Id       int64
	Title    string
	Author   string
	License  string
	Abstract string
	Image    string
	Type     string
	Path	 string
	CreateAt int64
	UpdateAt int64
	DeleteAt int64
}

func (LatexTemplate) TableName() string {
	return "latex_template"
}

func (lt *LatexTemplate) Create() error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	lt.CreateAt = nowTime
	lt.UpdateAt = nowTime
	return db.Create(lt).Error
}

func (lt *LatexTemplate) Update(ups map[string]interface{}) error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(lt).Update(ups).Error
}

func (lt LatexTemplate) Delete() error {
	db := database.GetDb()
	// 模板文件，直接删除，不做逻辑删除
	return db.Delete(lt).Error
}

func (lt *LatexTemplate) Get(id int64) error {
	db := database.GetDb()
	return db.Where("id = ? AND delete_at = 0", id).First(lt).Error
}

func GetLatexTemplates(templateType string, offset, limit int64) (LatexTemplates []LatexTemplate, total int64, err error) {
	db := database.GetDb()
	query := "type=? AND delete_at=0"
	err = db.Model(&LatexTemplate{}).Where(query, templateType).Count(&total).Error
	if err != nil {
		return
	}
	err = db.Where(query, templateType).Offset(offset).Limit(limit).Find(&LatexTemplates).Error
	return
}

