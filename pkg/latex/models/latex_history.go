package models

import (
	"TEFS-BE/pkg/database"
)

type LatexHistory struct {
	Id         int64
	LatexId    int64
	Version    int64
	ChangeList string
	CreateAt   int64
}

func (LatexHistory) TableName() string {
	return "latex_history"
}

func (l *LatexHistory) Create() error {
	db := database.GetDb()
	return db.Create(l).Error
}

// 获取latex 历史最后一个版本
func GetLatexHistoryLastVersion(latexId int64) (lastVersion int64, err error) {
	db := database.GetDb()
	query := "SELECT latex_version.version FROM latex_version WHERE latex_id=? ORDER BY version DESC LIMIT 1"
	var tmp = struct {
		Version int64
	}{}
	if err = db.Raw(query, latexId).Scan(&tmp).Error; err != nil {
		if err == database.NotFoundErr {
			return 0, nil
		}
		return
	}
	lastVersion = tmp.Version
	return
}