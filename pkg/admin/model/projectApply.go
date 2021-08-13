package model

import (
	"TEFS-BE/pkg/database"
	"fmt"
	"github.com/jinzhu/gorm"
)

// 项目申请结构体
type ProjectApply struct {
	Id        int64
	UserId    int64
	ProjectId int64
	Status    int64
	Details   string
	CreateAt  int64
	UpdateAt  int64
	DeleteAt  int64
}

// 项目申请和申请人用户信息
type ProjectApplyAndUserInfo struct {
	Id          int64
	UserName    string
	ProjectName string
	UserPhone   string
	Details     string
	CreateAt    int64
}

// 用户项目申请数据库服务
type ProjectApplyService struct {
}

// 表名
func (ProjectApply) TableName() string {
	return "project_apply"
}

// 创建记录
func (p ProjectApplyService) Create(projectApply *ProjectApply) *gorm.DB {
	db := database.GetDb()
	return db.Create(projectApply)
}

// 获取未审核记录
func (p ProjectApplyService) GetNotReviewRecord(userId, projectId int64) (*ProjectApply, error) {
	db := database.GetDb()
	projectApply := &ProjectApply{}
	err := db.Where("user_id = ? AND project_id = ? AND status = 1 AND delete_at = 0",
		userId, projectId).First(projectApply).Error
	if err == database.NotFoundErr {
		err = nil
	}
	return projectApply, err
}

// 获取记录列表
func (p ProjectApplyService) GetProjectApplyAndUserInfoList(offset, limit, userId int64) ([]ProjectApplyAndUserInfo,
	int64, error) {

	db := database.GetDb()
	var total int64
	var ProjectApplyAndUserInfoList []ProjectApplyAndUserInfo

	sql := "SELECT %s FROM " +
		"(SELECT project_apply.* FROM " +
		"(SELECT project_id FROM user_project WHERE user_id=? AND role < 3 AND status=1) " +
		"AS up INNER JOIN project_apply ON up.project_id=project_apply.project_id " +
		" WHERE project_apply.status=1 AND delete_at=0) " +
		"AS p INNER JOIN `user` ON p.user_id=user.id INNER JOIN" +
		" project ON p.project_id=project.id WHERE project.delete_at=0 "

	countSql := fmt.Sprintf(sql, "COUNT(*)")
	if err := db.Raw(countSql, userId).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	fields := "p.id,p.details,p.create_at,`user`.name AS user_name," +
		"`user`.phone as user_phone, project.name as project_name"
	querySql := fmt.Sprintf(sql, fields)
	if err := db.Raw(querySql,
		userId).Limit(limit).Offset(offset).Scan(&ProjectApplyAndUserInfoList).Error; err != nil {
		return nil, 0, err
	}
	return ProjectApplyAndUserInfoList, total, nil
}
