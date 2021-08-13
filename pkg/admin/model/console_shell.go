package model

import (
	"TEFS-BE/pkg/database"
)

type ConsoleShell struct {
	Id        int64  `json:"id"`
	ProjectId int64  `json:"project_id"`
	UserId    int64  `json:"user_id"`
	Address   string `json:"address"`
	Secret    string `json:"-"`
	CreateAt  int64  `json:"create_at"`
	UpdateAt  int64  `json:"update_at"`
	DeleteAt  int64  `json:"-"`
}

func (ConsoleShell) TableName() string {
	return "console_shell"
}

func (c *ConsoleShell) Create() error {
	db := database.GetDb()
	return db.Create(c).Error
}

func (c *ConsoleShell) Get(userId, projectId int64) error {
	db := database.GetDb()
	return db.Where("user_id = ? AND project_id = ? AND delete_at = 0", userId, projectId).Find(c).Error
}
