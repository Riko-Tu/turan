package model

import (
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"github.com/jinzhu/gorm"
)

type UserService struct {
}

type User struct {
	Id               int64
	Account          string // qq 微信 唯一标识
	Way              string // 登录方式 qq=QQ | wx=微信
	Name             string
	Email            string
	Phone            string
	IsNotify         int64
	AgreementVersion string // 用户协议版本
	CreateAt         int64
	UpdateAt         int64
	LastLoginAt      int64
}

func (User) TableName() string {
	return "user"
}

func (u UserService) Create(user *User) *gorm.DB {
	db := database.GetDb()
	return db.Create(user)
}

func (u UserService) GetUserByAccount(account string) *User {
	db := database.GetDb()
	user := &User{}
	if err := db.Where("account = ?", account).First(user).Error; err != nil {
		log.Error(err.Error())
	}
	return user
}

func (u UserService) IsExist(account string) bool {
	user := u.GetUserByAccount(account)
	return user.Id > 0
}

func (u UserService) Update(user *User, ups map[string]interface{}) *gorm.DB {
	db := database.GetDb()
	return db.Model(user).Update(ups)
}

func (u UserService) Get(id int64) *User {
	user := &User{}
	db := database.GetDb()
	if err := db.Where("id = ?", id).First(user).Error; err != nil {
		log.Error(err.Error())
	}
	return user
}

func (u UserService) GetList(offset, limit int64) ([]User, int64) {
	db := database.GetDb()
	var total int64
	var users []User
	db.Model(&User{}).Count(&total)
	db.Offset(offset).Limit(limit).Find(&users)
	return users, total
}