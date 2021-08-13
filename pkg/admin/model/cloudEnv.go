package model

import (
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/jinzhu/gorm"
	"time"
)

type CloudEnv struct {
	Id               int64
	UserId           int64
	VaspLicenseId    int64
	CloudSecretId    string
	CloudAppId       string
	Region           string
	Zone             string
	CloudAccountId   string // 腾讯云账户主id
	CloudProjectId   string // 腾讯云账户下项目id
	VpcId            string
	SubnetId         string
	SecurityGroupId  string
	CosBucket        string
	ClusterId        string
	InstanceId       string
	InstanceIp       string
	InstancePassword string
	Status           int64
	ErrMsg           string
	CreateAt         int64
	UpdateAt         int64
	DeleteAt         int64
}

type CloudEnvService struct {
}

func (CloudEnv) TableName() string {
	return "cloud_env"
}

func (c CloudEnvService) Create(cloudEnv *CloudEnv) *gorm.DB {
	db := database.GetDb()
	return db.Create(cloudEnv)
}

func (c CloudEnvService) Update(cloudEnv *CloudEnv, ups map[string]interface{}) *gorm.DB {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(cloudEnv).Update(ups)
}

func (c CloudEnvService) Get(id int64) *CloudEnv {
	db := database.GetDb()
	cloudEnv := &CloudEnv{}
	if err := db.Where("id = ? AND delete_at = 0", id).First(cloudEnv).Error; err != nil {
		log.Error(err.Error())
	}
	return cloudEnv
}

func (c CloudEnvService) GetByCloudSecretId(cloudSecretId string) (*CloudEnv, error) {
	db := database.GetDb()
	cloudEnv := &CloudEnv{}
	if err := db.Where("cloud_secret_id = ? AND delete_at = 0", cloudSecretId).First(cloudEnv).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return cloudEnv, nil
		}
		return nil, err
	}
	return cloudEnv, nil
}

func (c CloudEnvService) GetList(offset, limit int64, userId int64, status int64) ([]CloudEnv, int64) {
	db := database.GetDb()
	var total int64
	var cloudEnvs []CloudEnv
	var query string
	var fields string
	query = "delete_at = 0"
	if userId != 0 {
		query = fmt.Sprintf("user_id = %d AND ", userId) + query
	}
	if status != 0 {
		query = query + fmt.Sprintf(" AND status = %d", &status)
	}
	fields = "id, user_id, vasp_license_id, address, status, create_at, update_at"
	db.Model(&CloudEnv{}).Where(query).Count(&total)
	db.Select(fields).Where(query).Offset(offset).Limit(limit).Find(&cloudEnvs)
	return cloudEnvs, total
}

func (c CloudEnvService) GetByLicenseId(vaspLicenseId int64) *CloudEnv {
	db := database.GetDb()
	cloudEnv := &CloudEnv{}
	if err := db.Where("vasp_license_id = ? AND status=3 AND delete_at = 0",
		vaspLicenseId).First(cloudEnv).Error; err != nil {
		log.Error(err.Error())
	}
	return cloudEnv
}

func (c CloudEnvService) TxUpdateProjectAndCloudEnv(project *Project, cloudEnv *CloudEnv,
	userProject *UserProject, projectUps, cloudEnvUps map[string]interface{}) error {

	// 开启事务
	db := database.GetDb()
	tx := db.Begin()
	if err := tx.Model(cloudEnv).Update(cloudEnvUps).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(project).Update(projectUps).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Create(userProject).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func GetInstanceIp(userId, projectId int64) (ip string, err error) {
	db := database.GetDb()
	query := "SELECT cloud_env.instance_ip FROM (SELECT project.cloud_env_id FROM (SELECT project_id FROM user_project WHERE user_id=? AND project_id=? AND delete_at=0) as p INNER JOIN project ON p.project_id = project.id WHERE project.`status`=3 AND delete_at=0) as t INNER JOIN cloud_env ON t.cloud_env_id=cloud_env.id WHERE cloud_env.`status`=3 and delete_at=0"
	tmp := struct {
		InstanceIp string
	}{}
	err = db.Raw(query, userId, projectId).Scan(&tmp).Error
	ip = tmp.InstanceIp
	return
}