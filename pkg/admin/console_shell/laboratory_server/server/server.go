package server

import (
	"TEFS-BE/pkg/admin/model"
	admin "TEFS-BE/pkg/admin/service"
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var experimentService = model.ExperimentService{}
var userProjectService = model.UserProjectService{}

// 获取实验层级树关系
func GetExperimentTree(w http.ResponseWriter, r *http.Request) {
	_, userId, projectId, err := GetRequestParams(r)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	exs, err := experimentService.GetExperimentLevel(userId, projectId, true)
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("get experiment tree failed"))
		return
	}
	data, err := json.Marshal(exs)
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("data to json failed"))
		return
	}

	w.Write(data)
	return
}

// 创建实验
func CreateExperiment(w http.ResponseWriter, r *http.Request) {
	params, userId, projectId, err := GetRequestParams(r)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	name, ok := params["name"]
	if !ok || len(name) == 0 {
		w.Write([]byte("not found param name"))
		return
	}

	pidStr, ok := params["pid"]
	if !ok || len(pidStr) == 0 {
		w.Write([]byte("not found param pid"))
		return
	}
	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		w.Write([]byte("param pid is not int"))
		return
	}

	experimentType, ok := params["experiment_type"]
	if !ok || len(experimentType) == 0 {
		w.Write([]byte("not found param experiment_type"))
		return
	}

	experimentStatusStr, ok := params["experiment_status"]
	if !ok || len(experimentType) == 0 {
		w.Write([]byte("not found param experiment_status"))
		return
	}
	experimentStatus, err := strconv.ParseInt(experimentStatusStr, 10, 64)
	if err != nil {
		w.Write([]byte("param experiment_status is not int"))
		return
	}

	// vasp license 状态查询
	licenseStatus, err := experimentService.GetVaspLicenseStatusForProject(projectId)
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("internal error: query lab license failed"))
		return
	}
	if licenseStatus == admin.VaspLicenseIsProhibit {
		w.Write([]byte("lab license invalid"))
		return
	}

	err = admin.VerifyExperimentNameAndMemo(name, "", true)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	cloudEnv, err := userProjectService.GetProjectCloudEnvIp(userId, projectId)
	if err != nil {
		if err != database.NotFoundErr {
			log.Error(err.Error())
			w.Write([]byte("internal error: query cloud env failed"))
			return
		} else {
			w.Write([]byte("not found cloud env"))
			return
		}
	}

	// 验证父级实验参数
	if pid > 0 {
		pe, err := experimentService.Get(pid)
		if err != nil {
			if err == database.NotFoundErr {
				w.Write([]byte("not found father experiment"))
				return
			}
			log.Error(err.Error())
			w.Write([]byte("internal error: query father experiment failed"))
			return
		}
		if pe.ProjectId != projectId || pe.UserId != userId {
			w.Write([]byte("not found father experiment"))
			return
		}
	}

	laboratoryAddress := fmt.Sprintf("%s:%d", cloudEnv.InstanceIp, admin.LaboratoryPort)

	nowTime := time.Now().Unix()
	// 开启事务
	db := database.GetDb()
	tx := db.Begin()
	experiment := &model.Experiment{
		UserId:            userId,
		ProjectId:         projectId,
		CloudEnvId:        cloudEnv.Id,
		Name:              name,
		ExperimentType:    experimentType,
		Status:            experimentStatus, // 实验状态：未运行或已完成
		LaboratoryAddress: laboratoryAddress,
		IsTemplate:        1,
		LastEditAt:        nowTime,
		CreateAt:          nowTime,
		UpdateAt:          nowTime,
	}
	if pid > 0 {
		experiment.Pid = pid
	}
	if err := tx.Create(experiment).Error; err != nil {
		tx.Rollback()
		log.Error(err.Error())
		w.Write([]byte("internal error: create experiment record failed"))
		return
	}

	cosPath := fmt.Sprintf("cos://%s.cos.%s.myqcloud.com/users/%d/experiments/%d/", cloudEnv.CosBucket, cloudEnv.Region, userId, experiment.Id)
	ups := make(map[string]interface{})
	ups["cos_base_path"] = cosPath
	if err := tx.Model(experiment).Update(ups).Error; err != nil {
		log.Error(err.Error())
		tx.Rollback()
		w.Write([]byte("internal error: update experiment cospath failed"))
		return
	}
	tx.Commit()

	w.Write([]byte(strconv.FormatInt(experiment.Id, 10)))
	return
}

func findTargetSubExperiments(experimentRelation model.ExperimentRelationList, targetExId int64) (subExperiments model.ExperimentRelationList) {
	for _, e := range experimentRelation {
		if e.Id == targetExId {
			return e.Child
		} else {
			subExperiments = findTargetSubExperiments(e.Child, targetExId)
			if subExperiments != nil {
				return subExperiments
			}
		}
	}
	return
}

func findDeleteExperimentIds(experiments model.ExperimentRelationList, deleteExperiments *[]int64) () {
	for _, e := range experiments {
		*deleteExperiments = append(*deleteExperiments, e.Id)
		if e.Child != nil {
			findDeleteExperimentIds(e.Child, deleteExperiments)
		}
	}
	return
}

// 删除实验,及其实验下的子实验
func DeleteExperiment(w http.ResponseWriter, r *http.Request) {
	params, userId, projectId, err := GetRequestParams(r)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	experimentIdStr, ok := params["experiment_id"]
	if !ok || len(experimentIdStr) == 0 {
		w.Write([]byte("not found experiment id"))
		return
	}
	experimentId, err := strconv.ParseInt(experimentIdStr, 10, 64)
	if err != nil {
		w.Write([]byte("param experiment_id is not int"))
		return
	}

	experiment, err := experimentService.Get(experimentId)
	if err == database.NotFoundErr || (err == nil && experiment.UserId != userId ) {
		w.Write([]byte("not found experiment record"))
		return
	}
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("internal error: get experiment record failed"))
		return
	}

	exs, err := experimentService.GetExperimentLevel(userId, projectId, false)
	if err != nil {
		w.Write([]byte("internal error: get experiment tree failed"))
		return
	}
	subExperiments := findTargetSubExperiments(exs, experimentId)
	var deleteExperiments []int64
	deleteExperiments = append(deleteExperiments, experimentId)
	findDeleteExperimentIds(subExperiments, &deleteExperiments)

	if err := experimentService.DeleteExperiments(deleteExperiments, userId, projectId); err != nil {
		log.Error(err.Error())
		w.Write([]byte("internal error: update experiment record failed"))
		return
	}

	data, _ := json.Marshal(deleteExperiments)
	w.Write(data)
	return
}