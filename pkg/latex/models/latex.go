package models

import (
	"TEFS-BE/pkg/database"
	latexServer "TEFS-BE/pkg/latex"
	"fmt"
	"time"
)

type Latex struct {
	Id                 int64
	UserId             int64
	LastModifiedUserId int64
	Name               string
	CosPath            string
	CreateAt           int64
	UpdateAt           int64
	DeleteAt           int64
}

func (Latex) TableName() string {
	return "latex"
}

func (l *Latex) Create() error {
	db := database.GetDb()
	return db.Create(l).Error
}

func (l *Latex) TxCreate(createFile func(latexDir string) error) error {
	db := database.GetDb()
	tx := db.Begin()

	nowTime := time.Now().Unix()
	l.CreateAt = nowTime
	l.UpdateAt = nowTime
	if err := tx.Create(l).Error; err != nil {
		tx.Rollback()
		return err
	}
	userLatex := &UserLatex{
		UserId:   l.UserId,
		LatexId:  l.Id,
		Status:   1,
		Power:    1,
		CreateAt: nowTime,
		UpdateAt: nowTime,
		ConfigJson: `{"main_document":""}`,
	}
	if err := tx.Create(userLatex).Error; err != nil {
		tx.Rollback()
		return err
	}

	latexDir := fmt.Sprintf(latexServer.LatexDirFormat, latexServer.LatexBaseDir, l.UserId, l.Id)
	if err := createFile(latexDir); err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (l Latex) Update(ups map[string]interface{}) error {
	db := database.GetDb()
	return db.Model(l).Update(ups).Error
}

func (l Latex) TxDelete(ups map[string]interface{}, deleteDir func(latexDir string) error) (err error) {
	db := database.GetDb()
	tx := db.Begin()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	if err = tx.Model(l).Update(ups).Error; err != nil {
		tx.Rollback()
		return
	}
	latexDir := fmt.Sprintf(latexServer.LatexDirFormat, latexServer.LatexBaseDir, l.UserId, l.Id)
	if err = deleteDir(latexDir); err != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}

func (l Latex) Delete() error {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups := make(map[string]interface{})
	ups["update_at"] = nowTime
	ups["delete_at"] = nowTime
	return db.Model(l).Update(ups).Error
}

func (l *Latex) Get(id int64) error {
	db := database.GetDb()
	return db.Where("id = ? AND delete_at = 0", id).First(l).Error
}

type LatexInfo struct {
	Id int64
	UserId int64
	Name string
}

func GetLatexJoinUserLatex(userId int64, latexIds []string) (LatexInfoItems []*LatexInfo, err error) {
	db := database.GetDb()
	query := "SELECT t1.id, t1.user_id, t1.name FROM (SELECT * from latex WHERE id in ("+ genVar(latexIds) +") AND delete_at=0) AS t1 INNER JOIN user_latex ON t1.id=user_latex.latex_id WHERE user_latex.user_id=? AND user_latex.delete_at=0"
	var params []interface{}
	for _,v := range latexIds {
		params = append(params, v)
	}
	params = append(params, userId)
	if err = db.Raw(query, params...).Scan(&LatexInfoItems).Error; err != nil {
		return
	}
	return
}