package service

import (
	"TEFS-BE/pkg/admin/compute"
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/admin/task"
	"TEFS-BE/pkg/cache"
	"TEFS-BE/pkg/database"
	laboratoryCli "TEFS-BE/pkg/laboratory/client"
	labPb "TEFS-BE/pkg/laboratory/proto"
	laboratoryPb "TEFS-BE/pkg/laboratory/proto"
	"TEFS-BE/pkg/log"
	"context"
	"encoding/json"
	"fmt"
	mchTasks "github.com/RichardKnop/machinery/v1/tasks"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	LaboratoryPort    = 32500
	experimentService = model.ExperimentService{}

	experimentNameMaxLen int = 60
	experimentMemoMaxLen int = 200
)

// vasp license 被禁用的状态
const VaspLicenseIsProhibit int64 = 2

type oszicar struct {
	IonicStep    string `json:"ionicStep"`
	ElectronStep string `json:"electronStep"`
	Energy       string `json:"energy"`
	Algorithm    string `json:"algorithm"`
}

func VerifyExperimentNameAndMemo(name, memo string, nameIsMust bool) error {
	if nameIsMust {
		nameCount := utf8.RuneCountInString(name)
		if len(name) <= 0 || nameCount > experimentNameMaxLen {
			return fmt.Errorf("invalid name,max len 60")
		}
	} else {
		if len(name) > 0 && utf8.RuneCountInString(name) > experimentNameMaxLen {
			return fmt.Errorf("invalid name, max len 60")
		}
	}
	if len(memo) > 0 && utf8.RuneCountInString(memo) > experimentMemoMaxLen {
		return fmt.Errorf("invalid memo, max len 200")
	}
	return nil
}

// 创建实验记录
func (s Service) CreateExperiment(ctx context.Context,
	in *pb.CreateExperimentRequest) (*pb.CreateExperimentReply, error) {

	user := ctx.Value("user").(*model.User)
	projectId := in.GetProjectId()
	experimentName := in.GetName()
	memo := in.GetMemo()
	experimentType := in.GetExperimentType().String()
	pid := in.GetPid()

	// vasp license 状态查询
	licenseStatus, err := experimentService.GetVaspLicenseStatusForProject(projectId)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if licenseStatus == VaspLicenseIsProhibit {
		return nil, VaspDisable.Error()
	}

	err = VerifyExperimentNameAndMemo(experimentName, memo, true)
	if err != nil {
		return nil, ErrorErr.ErrorErr(err)
	}

	cloudEnv, err := userProjectService.GetProjectCloudEnvIp(user.Id, projectId)
	if err != nil {
		if err != database.NotFoundErr {
			log.Error(err.Error())
			return nil, QueryDbFailed.Error()
		} else {
			return nil, NotFoundRecord.ErrorRecord("cloudEnv")
		}
	}

	// 验证父级实验参数
	if pid > 0 {
		pe, err := experimentService.Get(pid)
		if err != nil {
			return nil, QueryDbFailed.Error()
		}
		if pe.ProjectId != projectId || pe.UserId != user.Id {
			return nil, NotFoundRecord.Error()
		}
	}

	laboratoryAddress := fmt.Sprintf("%s:%d", cloudEnv.InstanceIp, LaboratoryPort)
	client, err := laboratoryCli.GetClient(laboratoryAddress)
	if err != nil {
		return nil, GetClientError.Error()
	}

	nowTime := time.Now().Unix()
	// 开启事务
	db := database.GetDb()
	tx := db.Begin()
	experiment := &model.Experiment{
		UserId:            user.Id,
		ProjectId:         projectId,
		CloudEnvId:        cloudEnv.Id,
		Name:              experimentName,
		ExperimentType:    experimentType,
		Status:            1, // 未实验
		LaboratoryAddress: laboratoryAddress,
		IsTemplate:        1,
		LastEditAt:        nowTime,
		CreateAt:          nowTime,
		UpdateAt:          nowTime,
	}
	if len(memo) > 0 {
		experiment.Memo = memo
	}
	if pid > 0 {
		experiment.Pid = pid
	}
	if err := tx.Create(experiment).Error; err != nil {
		tx.Rollback()
		log.Error(err.Error())
		return nil, CreateRecordFailed.Error()
	}

	cosPath := fmt.Sprintf("/users/%d/experiments/%d/", user.Id, experiment.Id)
	cosTmpSecret, cosBaseUrl, err := laboratoryCli.GetCosUploadTmpSecret(client, cosPath)
	if err != nil {
		log.Error(err.Error())
		tx.Rollback()
		return nil, GetCosTmpSercretFailed.Error()
	}

	ups := make(map[string]interface{})
	ups["cos_base_path"] = *cosBaseUrl + cosPath
	if err := tx.Model(experiment).Update(ups).Error; err != nil {
		log.Error(err.Error())
		tx.Rollback()
		return nil, UpdateDbFailed.Error()
	}

	tx.Commit()

	return &pb.CreateExperimentReply{
		ExperimentId: experiment.Id,
		CosBaseUrl:   *cosBaseUrl + cosPath,
		CosCredential: &pb.CosCredential{
			TmpSecretID:  cosTmpSecret.TmpSecretID,
			TmpSecretKey: cosTmpSecret.TmpSecretKey,
			SessionToken: cosTmpSecret.SessionToken,
			Bucket:       cosTmpSecret.Bucket,
			Region:       cosTmpSecret.Region,
			ExpiredTime:  cosTmpSecret.ExpiredTime,
			Expiration:   cosTmpSecret.Expiration,
			StartTime:    cosTmpSecret.StartTime,
		},
	}, nil
}

// 实验室cos临时秘钥（上传，下载, 删除）
func (s Service) LaboratoryCosCredential(ctx context.Context,
	in *pb.LaboratoryCosCredentialRequest) (*pb.LaboratoryCosCredentialReply, error) {

	user := ctx.Value("user").(*model.User)
	experimentId := in.GetExperimentId()
	copyExperimentId := in.GetCopyExperimentId()
	opType := in.GetOpType().String()

	experiment, err := experimentService.Get(experimentId)
	if err != nil {
		if err == database.NotFoundErr {
			return nil, NotFoundRecord.ErrorRecord("experiment")
		}
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if experiment.UserId != user.Id {
		return nil, NotFoundRecord.Error()
	}
	if copyExperimentId > 0 {
		experimentCopy, err := experimentService.Get(experimentId)
		if err != nil {
			if err == database.NotFoundErr {
				return nil, NotFoundRecord.ErrorRecord("experiment")
			}
			log.Error(err.Error())
			return nil, QueryDbFailed.Error()
		}
		if experimentCopy.UserId != user.Id {
			return nil, NotFoundRecord.Error()
		}
		if experimentCopy.LaboratoryAddress != experiment.LaboratoryAddress {
			return nil, CopyExErr.Error()
		}
	}

	client, err := laboratoryCli.GetClient(experiment.LaboratoryAddress)
	if err != nil {
		return nil, GetLabCliErr.Error()
	}

	var cosTmpSecret *laboratoryPb.CosCredential
	var err1 error
	switch opType {
	case "upload":
		cosPath := fmt.Sprintf("/users/%d/experiments/%d/*", user.Id, experiment.Id)
		copyCosPath := fmt.Sprintf("/users/%d/experiments/%d/*", user.Id, copyExperimentId)
		cosTmpSecret, _, err1 = laboratoryCli.GetCosUploadTmpSecret(client, cosPath, copyCosPath)
	case "download":
		cosTmpSecret, _, err1 = laboratoryCli.GetCosDownloadTmpSecret(client, user.Id)
	case "delete":
		cosTmpSecret, _, err1 = laboratoryCli.GetCosDeleteTmpSecret(client, user.Id, experimentId)
	default:
		return nil, InvalidParams.ErrorParam("opType", "")
	}

	if err1 != nil {
		log.Error(err1.Error())
		return nil, GetCosTmpSercretFailed.Error()
	}
	return &pb.LaboratoryCosCredentialReply{
		CosCredential: &pb.CosCredential{
			TmpSecretID:  cosTmpSecret.TmpSecretID,
			TmpSecretKey: cosTmpSecret.TmpSecretKey,
			SessionToken: cosTmpSecret.SessionToken,
			Bucket:       cosTmpSecret.Bucket,
			Region:       cosTmpSecret.Region,
			ExpiredTime:  cosTmpSecret.ExpiredTime,
			Expiration:   cosTmpSecret.Expiration,
			StartTime:    cosTmpSecret.StartTime,
		},
		CosBaseUrl: experiment.CosBasePath,
	}, nil
}

func verifySubmitExperimentParams(computeNodeNum, computeNodeMaxNum int64) error {
	if computeNodeNum <= 0 {
		return InvalidParams.ErrorParam("computeNodeNum", "computeNodeNum > 0")
	}

	if computeNodeNum > computeNodeMaxNum {
		tips := fmt.Sprintf("computeNodeNum <= %d", computeNodeMaxNum)
		return InvalidParams.ErrorParam("computeNodeNum", tips)
	}
	return nil
}

func getExperiment(experimentId, userId int64) (*model.Experiment, error) {
	experiment, err := experimentService.Get(experimentId)
	if err != nil {
		if err == database.NotFoundErr {
			return nil, NotFoundRecord.ErrorRecord("experiment")
		}
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if experiment.UserId != userId {
		return nil, NotFoundRecord.ErrorRecord("experiment")
	}
	if experiment.Status == 2 || experiment.Status == 3 {
		return nil, ExperimentComputing.Error()
	}
	return experiment, nil
}

func setRedisLock(redisCli *redis.Client, redisLockKey string) error {
	startTime := time.Now().Unix()
	var timeout int64 = 40
	for {
		ok, err := redisCli.SetNX(redisLockKey, 1, time.Second*30).Result()
		if err != nil {
			log.Error(err.Error())
		}
		if ok {
			break
		}
		if (time.Now().Unix() - startTime) > timeout {
			return GetLockTimeOut.Error()
		}
		time.Sleep(time.Millisecond * 500)
	}
	return nil
}

func selectZones(client labPb.LaboratoryClient) (targetZones []string, err error) {

	// 计算环境zone选择
	//var targetZone *string
	//minAvailableCount := computeNodeMaxNum
	// 获取可用zone
	availableZoneList, err := laboratoryCli.AvailableZoneList(client)
	if err != nil {
		log.Error(err.Error())
		return nil, GetZoneFailed.Error()
	}
	return availableZoneList, nil
}

func shareComputeImage(experimentType string,
	client labPb.LaboratoryClient) (computeImage string, err error) {

	var cvmComputeImage string
	if strings.HasPrefix(experimentType, "gpu_") {
		cvmComputeImage = settingService.GetGPUCvmComputeImage()
	} else {
		cvmComputeImage = settingService.GetCvmComputeImage(experimentType)
	}

	// 计算环境cvm镜像查询是否存在，如果不存在就共享给用户腾讯云账户
	if len(cvmComputeImage) == 0 {
		return "", NotFoundComputeImage.Error()
	}
	imageInfo, laboratoryCloudAccount, err := laboratoryCli.CvmImage(client, cvmComputeImage)
	if err != nil {
		log.Error(err.Error())
		return "", GetLabCvmImageFailed.Error()
	}
	if len(imageInfo) == 0 {
		// 共享镜像到项目实验室腾讯云账户
		err := cvmService.ModifyImageSharePermission(cvmComputeImage, "SHARE", []string{laboratoryCloudAccount})
		if err != nil {
			log.Error(err.Error())
			return "", ShareComputeImageFailed.Error()
		}
	}
	return cvmComputeImage, nil
}

func verifyBatchEnvLimit(client labPb.LaboratoryClient) error {
	// 查询计算环境列表
	// batch compute 环境限额是否超出
	// limit=100,100为最大有效值。
	response, err := laboratoryCli.QueryExperimentEnvList(client, 0, 100)
	if err != nil {
		log.Error(err.Error())
		return QueryBatchEnvFailed.Error()
	}
	experimentEnvCount := response.Response.TotalCount
	experimentEnvLimit, err := settingService.GetBatchComputeEnvLimit()
	if err != nil {
		log.Error(err.Error())
		experimentEnvLimit = 10
		log.Warn("db get batch compute env limit failed, use default limit 10")
	}
	if *experimentEnvCount >= experimentEnvLimit {
		return BatchEnvLimit.Error()
	}
	return nil
}

// 批量计算环境实例磁盘大小
const BatchENVDiskSize int64 = 50

// 磁盘类型
// LOCAL_BASIC：本地硬盘
// LOCAL_SSD：本地SSD硬盘
// CLOUD_BASIC：普通云硬盘
// CLOUD_SSD：SSD云硬盘
// CLOUD_PREMIUM：高性能云硬盘
var DiskTypes = []string{"CLOUD_BASIC", "CLOUD_PREMIUM", "CLOUD_SSD", "LOCAL_SSD", "LOCAL_BASIC"}

func createBatchEnv(cosPath, cvmComputeImage, instanceType string, computeNodeNum int64, targetZones []string,
	client labPb.LaboratoryClient) (batchEnvId, useZone string, err error) {
	// 创建计算环境
	var createEnvErr error
	for _, targetZone := range targetZones {
		for _, diskType := range DiskTypes {
			batchEnvId, createEnvErr = laboratoryCli.CreateExperimentEnv(client, computeNodeNum, BatchENVDiskSize, cosPath, cvmComputeImage, targetZone, instanceType, diskType)
			if createEnvErr != nil {
				log.Error(createEnvErr.Error())
			} else {
				return batchEnvId, targetZone, nil
			}
		}

	}
	return "", "", BatchEnvCreateFailed.Error()
}

func verifyVaspLicense(experimentId int64) error {
	status, err := experimentService.GetExperimentVaspLicenseStatus(experimentId)
	if err != nil {
		log.Error(err.Error())
		return QueryDbFailed.Error()
	}
	if status == VaspLicenseIsProhibit {
		return VaspDisable.Error()
	}
	return nil
}

// 提交实验到batch compute
func (s Service) SubmitExperiment(ctx context.Context,
	in *pb.SubmitExperimentRequest) (*pb.SubmitExperimentReply, error) {

	user := ctx.Value("user").(*model.User)
	experimentId := in.GetExperimentId()
	computeNodeNum := in.GetComputeNodeNum()
	experimentType := in.GetExperimentType().String()

	computeNodeMaxNum, err := settingService.GetCvmRegionLimit()
	if err != nil {
		log.Error(err.Error())
		computeNodeMaxNum = 30
	}

	// 输入参数验证
	if err := verifySubmitExperimentParams(computeNodeNum, computeNodeMaxNum); err != nil {
		return nil, err
	}

	// 验证vaspLicense
	if err := verifyVaspLicense(experimentId); err != nil {
		return nil, err
	}

	// 获取数据库实验记录
	experiment, err := getExperiment(experimentId, user.Id)
	if err != nil {
		return nil, err
	}

	client, err := laboratoryCli.GetClient(experiment.LaboratoryAddress)
	if err != nil {
		return nil, GetLabCliErr.Error()
	}

	redisCli := cache.GetRedis()
	redisLock := fmt.Sprintf("laboratory.%d.experimentSubmit.lock", experiment.ProjectId)
	if err = setRedisLock(redisCli, redisLock); err != nil {
		return nil, err
	}
	defer redisCli.Del(redisLock)

	// zone
	targetZones, err := selectZones(client)
	if err != nil {
		return nil, err
	}

	// 共享计算镜像
	cvmComputeImage, err := shareComputeImage(experimentType, client)
	if err != nil {
		return nil, err
	}

	// 获取实例类型
	instanceType, err := settingService.GetInstanceType(experimentType)
	if err != nil {
		return nil, err
	}

	if err = verifyBatchEnvLimit(client); err != nil {
		return nil, err
	}

	// 创建batch计算环境
	cosPath := fmt.Sprintf("/users/%d/experiments/%d/", user.Id, experimentId)
	computeEnvId, useZone, err := createBatchEnv(cosPath, cvmComputeImage, instanceType, computeNodeNum, targetZones, client)
	if err != nil {
		return nil, err
	}

	isDeleteExperimentEnv := false
	defer func() {
		if isDeleteExperimentEnv {
			err := laboratoryCli.DeleteExperimentEnv(client, computeEnvId)
			if err != nil {
				log.Error(fmt.Sprintf("cloud env %d delete ExperimentEnv %s failed.err:%s",
					experiment.CloudEnvId, computeEnvId, err.Error()))
			}
		}
	}()

	// 更新数据库
	nowTime := time.Now().Unix()
	ups := make(map[string]interface{})
	ups["start_at"] = nowTime
	ups["zone"] = useZone
	ups["batch_compute_env_id"] = computeEnvId
	ups["compute_node_num"] = computeNodeNum
	ups["status"] = 2
	ups["done_at"] = 0
	ups["batch_jid"] = ""
	ups["err_msg"] = ""
	ups["experiment_type"] = experimentType
	if err := experimentService.Update(experiment, ups).Error; err != nil {
		log.Error(err.Error())
		isDeleteExperimentEnv = true
		return nil, UpdateDbFailed.Error()
	}

	// 发送异步任务
	taskArg := mchTasks.Arg{
		Name:  "experimentId",
		Type:  "int64",
		Value: experiment.Id,
	}
	eta := time.Now().UTC().Add(time.Second * 30) // 延时30s执行
	_, err = task.SendExperimentTask(context.Background(),
		0, task.MonitoringExperimentEnvFunc, experiment.Id, &eta, taskArg)
	if err != nil {
		isDeleteExperimentEnv = true
		log.Error(err.Error())
		return nil, SendExToConsumerFailed.Error()
	}
	log.Info(fmt.Sprintf("experiment %d, send task success", experiment.Id))
	return &pb.SubmitExperimentReply{Message: "ok"}, nil
}

// 终止实验
func (s Service) TerminateExperiment(ctx context.Context,
	in *pb.TerminateExperimentRequest) (*pb.TerminateExperimentReply, error) {

	user := ctx.Value("user").(*model.User)
	experimentId := in.GetExperimentId()

	dbTx, experiment, err := getExperimentForUpdate(experimentId)
	if err != nil {
		return nil, err
	}

	updateSuccess := false
	defer func() {
		if updateSuccess {
			dbTx.Commit()
		} else {
			dbTx.Rollback()
		}
	}()

	if experiment.UserId != user.Id {
		return nil, NotFoundRecord.ErrorRecord("experiment")
	}
	// 实验只能在，创建计算环境阶段或计算节点被终止
	if experiment.Status != 2 && experiment.Status != 3 {
		return nil, ExIsNotComputing.Error()
	}

	ups := make(map[string]interface{})
	ups["update_at"] = time.Now().Unix()
	ups["status"] = 4 // 被终止

	// 更新 项目申请记录状态
	if err := dbTx.Model(experiment).Update(ups).Error; err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}
	updateSuccess = true
	return &pb.TerminateExperimentReply{Message: "ok"}, nil
}

func createExperimentsData(records []model.Experiment) (data []*pb.Experiment) {
	for _, v := range records {
		e := &pb.Experiment{}
		e.Id = v.Id
		e.Name = v.Name
		e.ExperimentType = v.ExperimentType
		e.CosBasePath = v.CosBasePath
		e.Memo = v.Memo
		e.Status = v.Status
		e.IsTemplate = v.IsTemplate
		e.LastEditAt = v.LastEditAt
		e.ComputeNodeNum = v.ComputeNodeNum
		e.OszicarJson = v.OszicarJson
		e.Image = v.Image
		e.ErrMsg = v.ErrMsg
		e.StartAt = v.StartAt
		e.DoneAt = v.DoneAt
		e.CreateAt = v.CreateAt
		e.UpdateAt = v.UpdateAt
		e.Pid = v.Pid

		if len(v.OszicarJson) > 0 {
			//o := &oszicar{}
			o := &compute.ExpDictJson{}
			if err := json.Unmarshal([]byte(v.OszicarJson), o); err == nil {
				e.IonicStep = strconv.Itoa(o.IonicStep)
				e.ElectronStep = strconv.Itoa(o.ElectronStep)
				e.Energy = strconv.FormatFloat(o.Energy, 'E', -1, 64)
			}
		}
		data = append(data, e)
	}
	return
}

// 获取实验列表
func (s Service) GetExperimentList(ctx context.Context,
	in *pb.GetExperimentListRequest) (*pb.GetExperimentListReply, error) {

	user := ctx.Value("user").(*model.User)
	status := in.GetStatus()
	offset := in.GetOffset()
	limit := in.GetLimit()
	likes := in.GetLikes()
	projectId := in.GetProjectId()
	if projectId <= 0 {
		return nil, InvalidId.Error()
	}

	// 查询各种状态总数
	experimenting, done, draft, template, allParents, err := experimentService.GetRecordCount(user.Id, projectId)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}

	records, err := experimentService.GetList(user.Id, projectId, offset, limit, status, likes)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}

	return &pb.GetExperimentListReply{
		//Count:              experimenting + done + draft,
		Count:              allParents,
		ExperimentingCount: experimenting,
		DoneCount:          done,
		DraftCount:         draft,
		TemplateCount:      template,
		ExperimentList:     createExperimentsData(records),
	}, nil
}

// 获取实验详情
func (s Service) GetExperiment(ctx context.Context, in *pb.GetExperimentRequest) (*pb.GetExperimentReply, error) {
	user := ctx.Value("user").(*model.User)
	experimentId := in.GetExperimentId()
	if experimentId <= 0 {
		return nil, InvalidId.Error()
	}

	experiment, err := experimentService.Get(experimentId)
	if err != nil {
		if err == database.NotFoundErr {
			return nil, NotFoundRecord.Error()
		}
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if experiment.UserId != user.Id {
		return nil, NotFoundRecord.Error()
	}

	data := &pb.Experiment{
		Id:             experiment.Id,
		Name:           experiment.Name,
		ExperimentType: experiment.ExperimentType,
		CosBasePath:    experiment.CosBasePath,
		Memo:           experiment.Memo,
		Status:         experiment.Status,
		IsTemplate:     experiment.IsTemplate,
		LastEditAt:     experiment.LastEditAt,
		ComputeNodeNum: experiment.ComputeNodeNum,
		OszicarJson:    experiment.OszicarJson,
		Image:          experiment.Image,
		ErrMsg:         experiment.ErrMsg,
		StartAt:        experiment.StartAt,
		DoneAt:         experiment.DoneAt,
		CreateAt:       experiment.CreateAt,
		UpdateAt:       experiment.UpdateAt,
		Pid:            experiment.Pid,
	}
	if len(data.OszicarJson) > 0 {
		o := &compute.ExpDictJson{}
		if err := json.Unmarshal([]byte(data.OszicarJson), o); err == nil {
			data.IonicStep = strconv.Itoa(o.IonicStep)
			data.ElectronStep = strconv.Itoa(o.ElectronStep)
			data.Energy = strconv.FormatFloat(o.Energy, 'E', -1, 64)
		}
	}

	return &pb.GetExperimentReply{
		Experiment: data,
	}, nil
}

// 更新实验,名字 备注 最后编辑时间
func (s Service) UpdateExperiment(ctx context.Context,
	in *pb.UpdateExperimentRequest) (*pb.UpdateExperimentReply, error) {

	user := ctx.Value("user").(*model.User)
	experimentId := in.GetExperimentId()
	name := in.GetName()
	memo := in.GetMemo()
	clearMemo := in.GetClearMemo()
	isUpdateEdit := in.GetUpdateLastEditTime()
	if experimentId <= 0 {
		return nil, InvalidId.Error()
	}
	if err := VerifyExperimentNameAndMemo(name, memo, false); err != nil {
		return nil, ErrorErr.ErrorErr(err)
	}

	experiment, err := experimentService.Get(experimentId)
	if err != nil {
		if err == database.NotFoundErr {
			return nil, NotFoundRecord.Error()
		}
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if experiment.UserId != user.Id {
		return nil, NotFoundRecord.Error()
	}

	ups := make(map[string]interface{})
	if len(name) > 0 {
		ups["name"] = name
	}
	if len(memo) > 0 {
		ups["memo"] = memo
	}
	if clearMemo {
		ups["memo"] = ""
	}
	if isUpdateEdit {
		ups["last_edit_at"] = time.Now().Unix()
	}
	if err := experimentService.Update(experiment, ups).Error; err != nil {
		return nil, UpdateDbFailed.Error()
	}

	return &pb.UpdateExperimentReply{Message: "ok"}, nil
}

func getExperimentForUpdate(experimentId int64) (tx *gorm.DB, experiment *model.Experiment, err error) {
	if experimentId <= 0 {
		return nil, nil, InvalidId.Error()
}
	// 开启事务
	db := database.GetDb()
	tx = db.Begin()
	experiment = &model.Experiment{}
	// 行锁
	err = tx.Set("gorm:query_option", "FOR UPDATE").First(experiment, experimentId).Error
	if err != nil {
		tx.Rollback()
		if err == database.NotFoundErr {
			return nil, nil, NotFoundRecord.Error()
		}
		log.Error(err.Error())
		return nil, nil, TryAgainLater.Error()
	}
	return
}

// 删除实验
func (s Service) DeleteExperiment(ctx context.Context,
	in *pb.DeleteExperimentRequest) (*pb.DeleteExperimentReply, error) {

	user := ctx.Value("user").(*model.User)
	experimentId := in.GetExperimentId()

	// 如果实验下存在子实验，不允许删除
	sonExperimentTotal, err := experimentService.GetExperimentForPid(experimentId)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if sonExperimentTotal != 0 {
		return nil, ExistSonExperiment.Error()
	}

	tx, experiment, err := getExperimentForUpdate(experimentId)
	if err != nil {
		return nil, err
	}

	updateSuccess := false
	defer func() {
		if updateSuccess {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	if experiment.UserId != user.Id {
		return nil, NotFoundRecord.Error()
	}
	// 实验状态只有在特定状态下才能删除（1草稿 5失败 6成功）
	experimentStatus := experiment.Status
	if experimentStatus == 2 || experimentStatus == 3 {
		return nil, ExIsNotComputing.Error()
	}
	if experimentStatus == 4 {
		return nil, ExIsTerminating.Error()
	}

	laboratoryAddress := experiment.LaboratoryAddress
	client, err := laboratoryCli.GetClient(laboratoryAddress)
	if err != nil {
		log.Error(fmt.Sprintf("get %s grpc client err:%s", laboratoryAddress, err.Error()))
		return nil, GetLabCliErr.Error()
	}
	// 获取删除临时秘钥
	cosTmpSecret, cosBaseUrl, err := laboratoryCli.GetCosDeleteTmpSecret(client, user.Id, experimentId)
	if err != nil {
		log.Error(fmt.Sprintf("get %s cos tmp secret failed", laboratoryAddress))
		return nil, GetCosTmpSercretFailed.Error()
	}

	// 获取cos临时client
	cosClient, err := laboratoryCli.GetCosTmpClient(cosTmpSecret, *cosBaseUrl)
	if err != nil {
		log.Error(err.Error())
		return nil, GetCosCliFailed.Error()
	}
	// 查询当前实验下文件夹
	cosFilePrefix := fmt.Sprintf("users/%d/experiments/%d/", user.Id, experimentId)
	// 删除实验cos文件夹
	if err := compute.DeleteExperimentCos(cosClient, cosFilePrefix); err != nil {
		log.Error(fmt.Sprintf("delete experiment cos dir %s failed,err:%s", cosFilePrefix, err.Error()))
		return nil, DelExCosDirFailed.Error()
	}

	ups := make(map[string]interface{})
	ups["delete_at"] = time.Now().Unix()
	// 更新数据库
	if err := tx.Model(experiment).Update(ups).Error; err != nil {
		log.Error(err.Error())
		return nil, UpdateDbFailed.Error()
	}
	updateSuccess = true
	return &pb.DeleteExperimentReply{Message: "ok"}, nil
}

func (s Service) GetSubExperimentList(ctx context.Context,
	in *pb.GetSubExperimentListRequest) (*pb.GetSubExperimentListReply, error) {
	user := ctx.Value("user").(*model.User)
	pid := in.GetExperimentPid()
	projectId := in.GetProjectId()
	offset := in.GetOffset()
	limit := in.GetLimit()
	likes := in.GetLikes()
	searchScope := in.GetSearchScope().String()
	sortField := in.GetSortField().String()
	order := in.GetOrder().String()
	var pidP *int64
	switch searchScope {
	case "all":
		pidP = nil
	case "sub":
		pidP = &pid
	}
	if pid != 0 {
		ex, err := experimentService.Get(pid)
		if err != nil {
			return nil, QueryDbFailed.Error()
		}
		if ex.UserId != user.Id {
			return nil, NotFoundRecord.Error()
		}
	}

	subExs, total, err := experimentService.GetSubExperimentInfoForPid(user.Id, projectId, offset, limit, pidP, likes, sortField, order)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	for _, v := range subExs {
		if len(v.SubExperiments) == 0 {
			continue
		}
		exs := strings.Split(v.SubExperiments, ",")
		currSubEx := make(map[string]interface{})
		currSubEx["sub_total"] = len(exs)
		var subExList []interface{}
		for _, ex := range exs {
			exInfo := strings.Split(ex, " ")
			exMap := make(map[string]interface{})
			exMap["id"] = exInfo[0]
			exMap["name"] = exInfo[1]
			exMap["status"] = exInfo[2]
			subExList = append(subExList, exMap)
		}
		currSubEx["sub_info"] = subExList
		v.SubExInfo = currSubEx
	}

	data := make(map[string]interface{})
	data["total"] = total
	data["experiments"] = subExs
	dataByte, err := json.Marshal(data)
	if err != nil {
		log.Error(err.Error())
		return nil, ToJsonFailed.Error()
	}
	return &pb.GetSubExperimentListReply{SubExperimentsJson: string(dataByte)}, nil
}

func (s Service) GetExperimentLevel(ctx context.Context,
	in *pb.GetExperimentLevelRequest) (*pb.GetExperimentLevelReply, error) {
	user := ctx.Value("user").(*model.User)
	projectId := in.GetProjectId()
	exs, err := experimentService.GetExperimentLevel(user.Id, projectId, false)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	data, err := json.Marshal(exs)
	if err != nil {
		log.Error(err.Error())
		return nil, ToJsonFailed.Error()
	}
	return &pb.GetExperimentLevelReply{ExperimentLevelJson: string(data)}, nil
}

func  (s Service)GetExperimentBreadLine(ctx context.Context, in *pb.GetExperimentBreadLineRequest) (*pb.GetExperimentBreadLineReply, error) {
	user := ctx.Value("user").(*model.User)
	experimentId := in.GetExperimentId()
	if experimentId <= 0 {
		return nil, InvalidId.Error()
	}
	experiment, err := experimentService.Get(experimentId)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if experiment.UserId != user.Id {
		return nil, NotFoundRecord.Error()
	}
	breadLine, err := experimentService.GetExperimentBreadLine(user.Id, experiment)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	retByte, err := json.Marshal(breadLine)
	if err != nil {
		return nil, ToJsonFailed.Error()
	}
	return  &pb.GetExperimentBreadLineReply{Data: string(retByte)}, nil
}