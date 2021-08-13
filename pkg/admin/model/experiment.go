package model

import (
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/utils"
	"fmt"
	"github.com/jinzhu/gorm"
	"sort"
	"strings"
	"time"
)

type Experiment struct {
	Id                int64
	Pid               int64
	UserId            int64
	ProjectId         int64
	CloudEnvId        int64
	Name              string
	ExperimentType    string
	CosBasePath       string
	Memo              string
	Status            int64
	LaboratoryAddress string
	Zone              string
	BatchComputeEnvId string
	BatchJid          string
	IsTemplate        int64
	LastEditAt        int64
	ComputeNodeNum    int64
	OszicarJson       string
	Image             string
	ErrMsg            string
	StartAt           int64
	DoneAt            int64
	CreateAt          int64
	UpdateAt          int64
	DeleteAt          int64
}

type ExperimentStatusCount struct {
	Status int64
	Count  int64
}

type ExperimentService struct {
}

func (Experiment) TableName() string {
	return "experiment"
}

func (e ExperimentService) Create(experiment *Experiment) *gorm.DB {
	db := database.GetDb()
	return db.Create(experiment)
}

func (e ExperimentService) Get(experimentId int64) (*Experiment, error) {
	db := database.GetDb()
	experiment := &Experiment{}
	err := db.Where("id = ? AND delete_at = 0", experimentId).First(experiment).Error
	return experiment, err
}

func (e ExperimentService) Update(experiment *Experiment, ups map[string]interface{}) *gorm.DB {
	db := database.GetDb()
	nowTime := time.Now().Unix()
	ups["update_at"] = nowTime
	return db.Model(experiment).Update(ups)
}

func (e ExperimentService) GetList(userId, projectId, offset, limit, status int64, likes []string) ([]Experiment, error) {
	db := database.GetDb()
	var experiments []Experiment
	var query string
	query = fmt.Sprintf("user_id = %d AND project_id = %d AND delete_at = 0", userId, projectId)

	switch status {
	case 1:
		query += " AND status in (2, 3, 4)" // 实验中(2=创建计算环境，3=计算中)
	case 2:
		query += " AND status in (5, 6) " // 完成(4=终止 5=失败 6=成功)
	case 3:
		query += " AND status = 1" // 草稿
	case 4:
		query += " AND is_template=2" // 模板
	}
	if len(likes) > 0 {
		var regexps []string
		for _, v := range likes {
			if len(v) > 0 {
				ret := utils.Addcslashes(v, utils.SpecialStr)
				regexps = append(regexps, fmt.Sprintf(".*%s.*", ret))
			}
		}
		if len(regexps) > 0 {
			query += " AND concat(experiment.`name`,experiment.memo) REGEXP ?"
			err := db.Where(query, strings.Join(regexps, "|")).Offset(offset).Limit(limit).Order("create_at DESC").Find(&experiments).Error
			return experiments, err
		}
	}
	err := db.Where(query).Offset(offset).Limit(limit).Order("create_at DESC").Find(&experiments).Error
	return experiments, err
}

func (e ExperimentService) GetRecordCount(userId,
	projectId int64) (experimenting, done, draft, template, allParents int64, err error) {

	db := database.GetDb()
	query := fmt.Sprintf("user_id = %d AND project_id = %d AND delete_at=0", userId, projectId)

	var experimentStatusCountList []ExperimentStatusCount
	experiment := &Experiment{}
	err = db.Model(&Experiment{}).Select(
		"status, COUNT(id) as count").Where(query).Group("status").Scan(&experimentStatusCountList).Error
	if err != nil {
		return
	}
	for _, v := range experimentStatusCountList {
		switch v.Status {
		case 1:
			draft = v.Count
		case 2:
			experimenting += v.Count
		case 3:
			experimenting += v.Count
		case 4:
			experimenting += v.Count
		case 5:
			done += v.Count
		case 6:
			done += v.Count
		}
	}
	queryTemplate := query + " AND is_template = 2"
	err = db.Model(experiment).Where(queryTemplate).Count(&template).Error
	queryAllParents := query + " AND pid = 0"
	err = db.Model(experiment).Where(queryAllParents).Count(&allParents).Error
	return
}

func (e ExperimentService) GetLastUpdateRecord(userId int64) Project {
	db := database.GetDb()
	var project = &Project{}
	sql := "SELECT project.* FROM (" +
		"SELECT e.project_id, e.update_at as last FROM " +
		"(SELECT project_id, update_at FROM experiment WHERE user_id=?) as e " +
		"INNER JOIN " +
		"(SELECT project_id FROM user_project WHERE user_id=? AND `status`=1 AND delete_at=0) as u " +
		"ON e.project_id=u.project_id ) as tmp " +
		"INNER JOIN project ON tmp.project_id = project.id " +
		"ORDER BY last DESC LIMIT 1"
	db.Raw(sql, userId, userId).Scan(&project)
	return *project
}

// 获取实验的vasplicense状态
func (e ExperimentService) GetExperimentVaspLicenseStatus(experimentId int64) (status int64, err error) {
	db := database.GetDb()
	sql := "SELECT vasp_license.`status` FROM vasp_license " +
		"INNER JOIN " +
		"(SELECT project.vasp_license_id FROM project INNER JOIN " +
		"(SELECT project_id FROM experiment WHERE id=?) as e " +
		"ON project.id=e.project_id) as p ON p.vasp_license_id=vasp_license.id"
	var data struct {
		Status int64
	}
	err = db.Raw(sql, experimentId).Scan(&data).Error
	status = data.Status
	return
}

// 获取项目vasp license 状态
func (e ExperimentService) GetVaspLicenseStatusForProject(projectId int64) (status int64, err error) {
	db := database.GetDb()
	sql := "SELECT status from (SELECT vasp_license_id FROM project WHERE project.id=?) as p " +
		"INNER JOIN vasp_license on vasp_license.id = p.vasp_license_id"
	var data struct {
		Status int64
	}
	err = db.Raw(sql, projectId).Scan(&data).Error
	status = data.Status
	return
}

// 获取子实验
func (e ExperimentService) GetSubExperiment(userId, experimentPid int64) ([]Experiment, error) {
	db := database.GetDb()
	var experiments []Experiment
	var query string
	query = fmt.Sprintf("user_id = %d AND pid = %d AND delete_at = 0", userId, experimentPid)
	err := db.Where(query).Order("create_at DESC").Find(&experiments).Error
	return experiments, err
}

// 子实验信息
type SubExperiment struct {
	Id             int64                  `json:"id"`
	Pid            int64                  `json:"pid"`
	Name           string                 `json:"name"`
	Status         int64                  `json:"status"`
	Memo           string                 `json:"memo"`
	ErrMsg         string                 `json:"err_msg"`
	LastEditAt     int64                  `json:"last_edit_at"`
	StartAt        int64                  `json:"start_at"`
	DoneAt         int64                  `json:"done_at"`
	CreateAt       int64                  `json:"create_at"`
	UpdateAt       int64                  `json:"update_at"`
	SubExperiments string                 `json:"-"`
	SubExInfo      map[string]interface{} `json:"sub_ex_info"`
}

// 使用pid获取子实验，以及子实验的子实验简明信息
func (e ExperimentService) GetSubExperimentInfoForPid(userId, projectId, offset, limit int64, pid *int64, likes []string, sortField, order string) (subExperiments []*SubExperiment, total int, err error) {
	db := database.GetDb()
	query := "SELECT t1.id,t1.pid, t1.name,t1.status,t1.memo,t1.err_msg, t1.last_edit_at, t1.start_at,t1.done_at, t1.create_at, t1.update_at, GROUP_CONCAT(t2.id, ' ', t2.name, ' ', t2.`status`) as sub_experiments FROM (SELECT * from experiment WHERE user_id=? %s AND project_id=? AND %s delete_at=0) as t1 LEFT JOIN (SELECT * FROM experiment WHERE user_id=? AND project_id=? AND delete_at=0) as t2 ON t1.id=t2.pid GROUP BY id"
	queryTotal := "SELECT COUNT(id) as count FROM experiment WHERE user_id=? %s AND project_id=? AND %s delete_at=0 "
	var queryPid, queryRegexp, regexpsStr string
	if pid != nil {
		queryPid = "AND pid=? "
	}
	if len(likes) > 0 {
		var regexps []string
		for _, v := range likes {
			if len(v) > 0 {
				ret := utils.Addcslashes(v, utils.SpecialStr)
				regexps = append(regexps, fmt.Sprintf(".*%s.*", ret))
			}
		}
		if len(regexps) > 0 {
			queryRegexp = " concat(experiment.`name`) REGEXP ? AND "
			//queryRegexp = " concat(experiment.`name`,experiment.memo) REGEXP ? AND "
		}
		regexpsStr = strings.Join(regexps, "|")
	}

	query = fmt.Sprintf(query, queryPid, queryRegexp)
	queryTotal = fmt.Sprintf(queryTotal, queryPid, queryRegexp)
	tmp := &struct {
		Count int
	}{}
	sortOrder := fmt.Sprintf("%s %s", sortField, order)
	if pid != nil && len(regexpsStr) > 0 {
		err = db.Raw(query, userId, pid, projectId, regexpsStr, userId, projectId, ).Offset(offset).Limit(limit).Order(sortOrder).Find(&subExperiments).Error
		err = db.Raw(queryTotal, userId, pid, projectId, regexpsStr).Scan(tmp).Error
	} else if pid == nil && len(regexpsStr) > 0 {
		err = db.Raw(query, userId, projectId, regexpsStr, userId, projectId).Offset(offset).Limit(limit).Order(sortOrder).Find(&subExperiments).Error
		err = db.Raw(queryTotal, userId, projectId, regexpsStr).Scan(tmp).Error
	} else if pid != nil && len(regexpsStr) == 0 {
		err = db.Raw(query, userId, pid, projectId, userId, projectId).Offset(offset).Limit(limit).Order(sortOrder).Find(&subExperiments).Error
		err = db.Raw(queryTotal, userId, pid, projectId).Scan(tmp).Error
	} else {
		err = fmt.Errorf("pid and likes is nill")
		return
	}
	total = tmp.Count
	return
}

// 实验关系信息
type ExperimentRelation struct {
	Id         int64                  `json:"id"`
	Pid        int64                  `json:"pid"`
	Status     int64                  `json:"status"`
	Name       string                 `json:"name"`
	CreateAt   int64                  `json:"create_at"`
	LastEditAt int64                  `json:"last_edit_at"`
	Child      ExperimentRelationList `json:"child"`
	DeleteAt   int64                  `json:"delete_at"`
}

type ExperimentRelationList []*ExperimentRelation

// 倒序
func reverseOrder(i, j int64) bool {
	if i < j {
		return true
	}
	return false
}

func (p ExperimentRelationList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ExperimentRelationList) Len() int      { return len(p) }

//func (p ExperimentRelationList) Less(i, j int) bool { return p[i].CreateAt > p[j].CreateAt } // 按创建时间排序
func (p ExperimentRelationList) Less(i, j int) bool { return reverseOrder(p[i].CreateAt, p[j].CreateAt) } // 按创建时间排序
//func (p ExperimentRelationList) Less(i, j int) bool { return p[i].LastEditAt > p[j].LastEditAt } // 按最后编辑时间排序

// 获取实验级别
func (e ExperimentService) GetExperimentLevel(userId, projectId int64, isAll bool) (experimentRelation ExperimentRelationList, err error) {
	db := database.GetDb()
	var query string
	if isAll {
		query = "SELECT e.id, e.pid, e.status, e.name, e.create_at, e.last_edit_at, e.delete_at FROM experiment AS e WHERE user_id=? AND project_id=? ORDER BY pid DESC"
	} else {
		query = "SELECT e.id, e.pid, e.status, e.name, e.create_at, e.last_edit_at FROM experiment AS e WHERE user_id=? AND project_id=? AND delete_at=0 ORDER BY pid DESC"
	}
	err = db.Raw(query, userId, projectId).Scan(&experimentRelation).Error
	if len(experimentRelation) > 0 {
		for {
			firstEx := experimentRelation[0]
			if firstEx.Pid == 0 {
				break
			}
			experimentRelation = experimentRelation[1:]
			arrange(experimentRelation, firstEx)
		}
		exSort(experimentRelation)
	}
	return
}

// 排列关系
func arrange(experimentRelation []*ExperimentRelation, firstEx *ExperimentRelation) {
	for _, v := range experimentRelation {
		if v.Id == firstEx.Pid {
			v.Child = append(v.Child, firstEx)
			return
		}
		arrange(v.Child, firstEx)
	}
}

// 排序
func exSort(experimentRelation ExperimentRelationList) {
	if experimentRelation != nil {
		sort.Sort(experimentRelation)
	}
	for _, v := range experimentRelation {
		exSort(v.Child)
	}
}

// pid获取子实验
func (e ExperimentService) GetExperimentForPid(pid int64) (total int, err error) {
	db := database.GetDb()
	query := "SELECT COUNT(id) as count from experiment where pid=? and delete_at=0"
	tmp := struct {
		Count int
	}{}
	err = db.Raw(query, pid).Scan(&tmp).Error
	total = tmp.Count
	return
}

// 面包线
type ExInfo struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
	Pid  int64  `json:"pid"`
}

type ExInfoList []*ExInfo

func (p ExInfoList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ExInfoList) Len() int           { return len(p) }
func (p ExInfoList) Less(i, j int) bool { return p[i].Pid < p[j].Pid } //按id排序

func (e ExperimentService) GetExperimentBreadLine(userId int64, experiment *Experiment) (breadLine ExInfoList, err error) {
	db := database.GetDb()
	query := "SELECT e.id, e.pid, e.name FROM experiment AS e WHERE user_id=? AND id <= ? AND delete_at=0 ORDER BY id DESC"
	var experimentRelation ExperimentRelationList
	err = db.Raw(query, userId, experiment.Pid).Scan(&experimentRelation).Error
	currPid := experiment.Pid
	for _, v := range experimentRelation {
		if v.Id == currPid {
			breadLine = append(breadLine, &ExInfo{
				Id:   v.Id,
				Name: v.Name,
				Pid:  v.Pid,
			})
			currPid = v.Pid
		}
	}
	sort.Sort(breadLine)
	breadLine = append(breadLine, &ExInfo{
		Id:   experiment.Id,
		Name: experiment.Name,
		Pid:  experiment.Pid,
	})
	return
}

func genVar(ids []int64) string {
	inCondition := ""
	for range ids {
		if inCondition != "" {
			inCondition += ", "
		}
		inCondition += "?"
	}
	return inCondition
}

func (e ExperimentService)DeleteExperiments(experimentIds []int64, userId, projectId int64) error {
	deleteAt := time.Now().Unix()
	updateSql := "UPDATE `experiment` SET `delete_at`=? WHERE user_id=? AND project_id=? AND delete_at=0 AND id in (" + genVar(experimentIds) + ")"
	var params []interface{}
	params = append(params, deleteAt, userId, projectId)
	for _, v := range experimentIds {
		params = append(params, v)
	}
	db := database.GetDb()
	tx := db.Begin()
	ret := tx.Exec(updateSql, params...)
	if ret.Error != nil {
		tx.Rollback()
		return ret.Error
	}
	rowsAffected := ret.RowsAffected
	if rowsAffected != int64(len(experimentIds)) {
		tx.Rollback()
		return fmt.Errorf("update expeiment rows affected error")
	}
	tx.Commit()
	return nil
}