package model

import (
	"TEFS-BE/pkg/database"
	"fmt"
	"github.com/jinzhu/gorm"
)

// 标签数据库服务
type TagService struct {
}

// 标签表结构
type Tag struct {
	Id       int64
	UserId   int64
	Name     string
	Colour   string
	CreateAt int64
	UpdateAt int64
}

// 标签表名
func (Tag) TableName() string {
	return "tag"
}

// 创建记录
func (t TagService) Create(tag *Tag) *gorm.DB {
	db := database.GetDb()
	return db.Create(tag)
}

// 获取用户标签总数
func (t TagService) GetUserTagTotal(userId int64) (int64, error) {
	db := database.GetDb()
	var total int64
	query := fmt.Sprintf("user_id = %d AND delete_at = 0", userId)
	err := db.Model(&Tag{}).Where(query).Count(&total).Error
	return total, err
}

// 使用用户id和标签名获取标签
func (t TagService) GetByUserAndName(userId int64, tagName string) (*Tag, error) {
	db := database.GetDb()
	tag := &Tag{}
	err := db.Where("user_id = ? AND name = ?", userId, tagName).First(tag).Error
	return tag, err
}

// 获取标签列表
func (t TagService) GetList(userId, offset, limit int64) ([]*Tag, int64, error) {
	db := database.GetDb()
	tags := []*Tag{}
	var total int64
	var err error
	query := fmt.Sprintf("user_id = %d AND delete_at = 0", userId)

	err = db.Model(&Tag{}).Where(query).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Where(query).Offset(offset).Limit(limit).Find(&tags).Error
	return tags, total, err
}