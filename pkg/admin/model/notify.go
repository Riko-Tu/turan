package model

import (
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"github.com/jinzhu/gorm"
	"time"
)

type Notify struct {
	Id         int64
	NotifyType int64 // 消息类型 1=系统通知 2=项目通知 3=实验通知
	UserId     int64 // 用户id,如果为0,表示根据用户权限可见通知
	ProjectId  int64 // 项目id,如果为0表示为系统通知消息
	Title      string
	Content    string
	CreateAt   int64
	UpdateAt   int64
	DeleteAt   int64
}

type NotifyStatus struct {
	Id        int64
	NotifyId  int64
	UserId    int64
	ProjectId int64
}

type NotifyRecord struct {
	Id         int64
	NotifyType int64
	Title      string
	Content    string
	CreateAt   string
	ReadId     int64
}

type NotifyService struct {
}

func (Notify) TableName() string {
	return "notify"
}

func (NotifyStatus) TableName() string {
	return "notify_status"
}

func (n NotifyService) Create(notify *Notify) *gorm.DB {
	db := database.GetDb()
	return db.Create(notify)
}

func (n NotifyService) Update(notify *Notify, ups map[string]interface{}) *gorm.DB {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(notify).Update(ups)
}

func (n NotifyService) Get(id int64) *Notify {
	db := database.GetDb()
	notify := &Notify{}
	if err := db.Where("id = ? AND delete_at = 0", id).First(notify).Error; err != nil {
		log.Error(err.Error())
	}
	return notify
}

func (n NotifyService) GetList(offset, limit, userId int64) ([]NotifyRecord, int64, error) {
	db := database.GetDb()
	var unreadTotal int64
	var notifyRecords []NotifyRecord

	// 未读消息数量sql
	unreadTotalSql := `
	SELECT COUNT(*) FROM (
	SELECT id FROM notify WHERE user_id=? AND delete_at=0
	UNION SELECT id FROM notify WHERE user_id=0 AND project_id=0 AND delete_at=0
	UNION SELECT n.id FROM (SELECT id,project_id FROM notify WHERE user_id=0 AND project_id!=0 AND delete_at=0) AS n
	INNER JOIN (SELECT project_id FROM user_project WHERE user_id=? AND delete_at=0 AND status=1 AND role<3) AS p
	ON n.project_id=p.project_id) 
	AS r
	LEFT JOIN (SELECT id,notify_id FROM notify_status WHERE user_id=?) AS s ON r.id=s.notify_id WHERE ISNULL(s.notify_id)
    `
	if err := db.Raw(unreadTotalSql, userId, userId, userId).Count(&unreadTotal).Error; err != nil {
		return nil, 0, err
	}

	// 消息
	sql := `
	SELECT r.*, s.id as read_id FROM (
	SELECT * FROM notify WHERE user_id=? AND delete_at=0
	UNION SELECT * FROM notify WHERE user_id=0 AND project_id=0 AND delete_at=0
	UNION SELECT n.* FROM (SELECT * FROM notify WHERE user_id=0 AND project_id!=0 AND delete_at=0) AS n
	INNER JOIN (SELECT project_id FROM user_project WHERE user_id=? AND delete_at=0 AND status=1 AND role<3) AS p
	ON n.project_id=p.project_id) 
	AS r
	LEFT JOIN (SELECT id,notify_id FROM notify_status WHERE user_id=?) AS s ON r.id=s.notify_id
	ORDER BY create_at DESC
    `
	if err := db.Raw(sql, userId, userId, userId).Limit(limit).Offset(offset).Scan(&notifyRecords).Error; err != nil {
		return nil, 0, err
	}
	return notifyRecords, unreadTotal, nil
}

func (n NotifyService) CreateNotifyStatus(userId, projectId, notifyId int64) *gorm.DB {
	notifyStatus := &NotifyStatus{
		UserId:    userId,
		ProjectId: projectId,
		NotifyId:  notifyId,
	}
	db := database.GetDb()
	return db.Create(notifyStatus)
}
