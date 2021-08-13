package model

import (
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"encoding/base64"
	"fmt"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"strings"
	"time"
)

type Project struct {
	Id             int64
	UserId         int64
	CloudEnvId     int64
	VaspLicenseId  int64
	Name           string
	InvitationCode string
	Link           string
	Status         int64
	Describe       string
	CreateAt       int64
	UpdateAt       int64
	DeleteAt       int64
}

type ProjectService struct {
}

func (Project) TableName() string {
	return "project"
}

// 创建邀请码，不与已存在记录重复
func (p ProjectService) CreateInvitationCode(project *Project) error {
	var invitationCode string
	db := database.GetDb()
	for {
		invitationCode = strings.ToUpper(strings.ReplaceAll(uuid.NewV4().String(), "-", "")[0:7])
		if err := db.Where("invitation_code = ?", invitationCode).First(&Project{}).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				project.InvitationCode = invitationCode
				return nil
			}
			return err
		}
	}
}

// 创建项目链接
func (p ProjectService) CreateLink(project *Project) {
	link := base64.StdEncoding.EncodeToString([]byte(project.InvitationCode))
	project.Link = link
}

func (p ProjectService) Create(project *Project) *gorm.DB {
	db := database.GetDb()
	return db.Create(project)
}

func (p ProjectService) TxCreateProjectAndCloudEnv(project *Project, cloudEnv *CloudEnv) error {
	db := database.GetDb()
	// 开启事务
	tx := db.Begin()
	if err := tx.Create(cloudEnv).Error; err != nil {
		tx.Rollback()
		return err
	}
	project.CloudEnvId = cloudEnv.Id
	if err := tx.Create(project).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	tx.Commit()
	return nil
}

func (p ProjectService) InvitationCodeIsExist(invitationCode string) bool {
	db := database.GetDb()
	err := db.Where("invitation_code = ?", invitationCode).First(&Project{}).Error
	if err == gorm.ErrRecordNotFound {
		return false
	} else {
		return true
	}
}


func (p ProjectService) Update(project *Project, ups map[string]interface{}) *gorm.DB {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(project).Update(ups)
}

func (p ProjectService) Get(id int64) *Project {
	db := database.GetDb()
	project := &Project{}
	if err := db.Where("id = ? AND delete_at = 0", id).First(project).Error; err != nil {
		log.Error(err.Error())
	}
	return project
}

func (p ProjectService) GetByCloudEnvId(cloudEnvId int64) *Project {
	db := database.GetDb()
	project := &Project{}
	if err := db.Where("cloud_env_id = ? AND delete_at = 0", cloudEnvId).First(project).Error; err != nil {
		log.Error(err.Error())
	}
	return project
}

func (p ProjectService) GetByUserAndName(userId int64, name string) (*Project, error) {
	db := database.GetDb()
	project := &Project{}
	if err := db.Where("user_id = ? AND name = ? AND delete_at = 0", userId, name).First(project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return project, nil
		}
		return nil, err
	}
	return project, nil
}

func (p ProjectService) GetList(offset, limit int64, userId int64, status int64) ([]Project, int64) {
	db := database.GetDb()
	var total int64
	var projects []Project
	var query string
	query = "delete_at = 0"
	if userId != 0 {
		query = fmt.Sprintf("user_id = %d AND ", userId) + query
	}
	if status != 0 {
		query = query + fmt.Sprintf(" AND status = %d", status)
	} else {
		query = query + " AND status != 1"
	}
	db.Model(&Project{}).Where(query).Count(&total)
	db.Where(query).Offset(offset).Limit(limit).Find(&projects)
	return projects, total
}

func (p ProjectService) GetProjectNumByLicenseId(vaspLicenseId int64) (int64, error) {
	var total int64
	db := database.GetDb()
	err := db.Model(&Project{}).Where("vasp_license_id = ? AND delete_at = 0", vaspLicenseId).Count(&total).Error
	return total, err
}

func (p ProjectService) GetByInvitationCode(invitationCode string) (*Project, error) {
	db := database.GetDb()
	project := &Project{}
	err := db.Where("invitation_code = ? AND delete_at = 0", invitationCode).First(project).Error;
	if err == database.NotFoundErr {
		err = nil
	}
	return project, err
}