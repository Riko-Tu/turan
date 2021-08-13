package model

import (
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/jinzhu/gorm"
	"strconv"
	"strings"
	"time"
)

type Setting struct {
	Id       int64
	Title      string
	Value    string
	Describe string
	CreateAt int64
	UpdateAt int64
	DeleteAt int64
}

type SettingService struct {
}

func (Setting) TableName() string {
	return "setting"
}

func (s SettingService) Create(setting *Setting) *gorm.DB {
	db := database.GetDb()
	return db.Create(setting)
}

// id获取记录
func (s SettingService) Get(id int64) *Setting {
	db := database.GetDb()
	setting := &Setting{}
	if err := db.Where("id = ? AND delete_at = 0", id).First(setting).Error; err != nil {
		log.Error(err.Error())
	}
	return setting
}

// 使用标题获取value
func (s SettingService) GetSettingByTitle(title string) *Setting {
	db := database.GetDb()
	setting := &Setting{}
	if err := db.Where("title = ? AND delete_at = 0", title).First(setting).Error; err != nil {
		log.Error(err.Error())
	}
	return setting
}

// 判断用户是否为管理员
func (s SettingService) UserIsAdmin(user *User) bool {
	var key string = "admin_accounts"
	setting := s.GetSettingByTitle(key)
	accounts := strings.Split(setting.Value, ",")
	isAdmin := false
	for _, v := range accounts {
		if v == user.Account {
			isAdmin = true
			break
		}
	}
	return isAdmin
}

// 获取记录列表
func (s SettingService) GetList(offset, limit int64) ([]Setting, int64) {
	db := database.GetDb()
	var total int64
	var settings []Setting
	db.Model(&Setting{}).Where("delete_at = ?", 0).Count(&total)
	db.Where("delete_at = ?", 0).Offset(offset).Limit(limit).Find(&settings)
	return settings, total
}

// 更新记录
func (s SettingService) Update(setting *Setting, ups map[string]interface{}) *gorm.DB {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(setting).Update(ups)
}

// 获取发送邮件时的别名
func (s SettingService) GetEmailAlias() string {
	title := "email_alias"
	setting := s.GetSettingByTitle(title)
	return setting.Value
}

// 获取官网邮箱
func (s SettingService) GetOfficialEmail() string {
	title := "official_email"
	setting := s.GetSettingByTitle(title)
	return setting.Value
}

// 获取官网链接
func (s SettingService) GetOfficialLink() string {
	title := "official_link"
	setting := s.GetSettingByTitle(title)
	return setting.Value
}

// 获取cvm的地区限制，南京地区为30
func (s SettingService) GetCvmRegionLimit() (int64, error) {
	title := "cvm_region_limit"
	setting := s.GetSettingByTitle(title)
	return strconv.ParseInt(setting.Value, 10, 64)
}

// 根据传入实验类型，获取计算cvm镜像
func (s SettingService) GetCvmComputeImage(experimentType string) string {
	items := strings.Split(experimentType,"_")
	if len(items) <= 0 {
		return ""
	}
	experimentTypePrefix := items[0]
	title := fmt.Sprintf("%s_cvm_image", experimentTypePrefix)
	setting := s.GetSettingByTitle(title)
	return setting.Value
}

// 获取batch compute 计算环境数量限制
func (s SettingService) GetBatchComputeEnvLimit() (int64, error) {
	title := "batch_compute_env_limit"
	setting := s.GetSettingByTitle(title)
	return strconv.ParseInt(setting.Value, 10, 64)
}

// 获取vasp计算指令(bas64编码)
func (s SettingService) GetVaspB64Cmd() string {
	title := "vasp_b64_cmd"
	setting := s.GetSettingByTitle(title)
	return setting.Value
}

// 获取vasp计算指令(bas64编码)
func (s SettingService) GetVaspExecPath(vaspType string) string {
	title := fmt.Sprintf("%s_exec_path", vaspType)
	setting := s.GetSettingByTitle(title)
	return setting.Value
}

// 获取用户计算任务使用cvm实例类型
func (s SettingService) GetInstanceType(experimentType string) (string, error) {
	var title string
	if strings.HasPrefix(experimentType, "gpu_") {
		title = "batch_compute_gpu_cvm_instance_type"
	} else {
		title = "batch_compute_cvm_instance_type"
	}
	setting := s.GetSettingByTitle(title)
	if setting.Id == 0 {
		return "", fmt.Errorf("db setting not found instance type")
	}
	return setting.Value, nil
}

// 获取gpu cmv 镜像
func (s SettingService) GetGPUCvmComputeImage() string {
	title := fmt.Sprintf("vasp_gpu_cvm_image")
	setting := s.GetSettingByTitle(title)
	return setting.Value
}

// 获取vasp GPU计算指令(bas64编码)
func (s SettingService) GetVaspGPUB64Cmd() string {
	title := "vasp_gpu_b64_cmd"
	setting := s.GetSettingByTitle(title)
	return setting.Value
}

// 获取vasp GPU计算指令(bas64编码)
func (s SettingService) GetVaspGPUExecBasePath() string {
	title := "vasp_gpu_exec_base_path"
	setting := s.GetSettingByTitle(title)
	return setting.Value
}