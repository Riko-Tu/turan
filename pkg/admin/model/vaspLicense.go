package model

import (
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/jinzhu/gorm"
	"time"
)

type License struct {
	Id           int64
	UserId       int64
	Organization string // 机构
	Domain       string // 研究领域
	BindEmail    string // vasp license绑定的邮箱
	CosPath      string
	Status       int64  // 状态  1=待审核  | 2=失败 | 3=成功
	CreateAt     int64
	UpdateAt     int64
	DeleteAt     int64
}

type LicenseService struct {
}

func (License) TableName() string {
	return "vasp_license"
}

func (l LicenseService) Create(license *License) *gorm.DB {
	db := database.GetDb()
	return db.Create(license)
}

func (l LicenseService) Update(license *License, ups map[string]interface{}) *gorm.DB {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(license).Update(ups)
}

func (l LicenseService) Get(id int64) *License {
	db := database.GetDb()
	license := &License{}
	if err := db.Where("id = ? AND delete_at = 0", id).First(license).Error; err != nil {
		log.Error(err.Error())
	}
	return license
}

func (l LicenseService) GetList(offset, limit int64, userId int64, status int64) ([]License, int64) {
	db := database.GetDb()
	var total int64
	var licenses []License
	var query string
	query = "delete_at = 0"
	if userId != 0 {
		query = fmt.Sprintf("user_id = %d AND ", userId) + query
	}
	if status != 0 {
		query = query + fmt.Sprintf(" AND status = %d", status)
	}
	db.Model(&License{}).Where(query).Count(&total)
	db.Where(query).Order("create_at desc").Offset(offset).Limit(limit).Find(&licenses)
	return licenses, total
}
