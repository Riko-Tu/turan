package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/log"
	"context"
)

var cloudEnvService = model.CloudEnvService{}

// 更新用户服务腾讯云环境
func (s Service) UpdateCloudEnv(ctx context.Context, in *pb.UpdateCloudEnvRequest) (*pb.UpdateCloudEnvReply, error) {
	user := ctx.Value("user").(*model.User)
	envId := in.GetId()
	if envId <= 0 {
		return nil, InvalidId.Error()
	}
	cloudEnv := cloudEnvService.Get(envId)
	if cloudEnv.Id <= 0 || cloudEnv.UserId != user.Id {
		return nil, NotFoundRecord.Error()
	}

	var ups = make(map[string]interface{})
	var params = make(map[string]string)
	params["region"] = in.GetRegion()
	params["zone"] = in.GetZone()
	params["cloud_account_id"] = in.GetCloudAccountId()
	params["cloud_project_id"] = in.GetCloudProjectId()
	params["vpc_id"] = in.GetVpcId()
	params["subnet_id"] = in.GetSubnetId()
	params["security_group_id"] = in.GetSecurityGroupId()
	params["cos_bucket"] = in.GetCosBucket()
	params["cluster_id"] = in.GetClusterId()
	params["instance_id"] = in.GetInstanceId()
	params["instance_ip"] = in.GetInstanceIp()
	for k, v := range params {
		if len(v) > 0 {
			ups[k] = v
		}
	}
	errMsg := in.GetErrMsg()
	if len(errMsg) > 0 {
		ups["err_msg"] = errMsg
		ups["status"] = 2
	}
	if err := cloudEnvService.Update(cloudEnv, ups).Error; err != nil {
		log.Error(err.Error())
		return nil, DbSaveFailed.Error()
	}
	return &pb.UpdateCloudEnvReply{Message: "ok"}, nil
}

// 获取用户服务腾讯云环境
func (s Service) GetCloudEnv(ctx context.Context, in *pb.GetCloudEnvRequest) (*pb.GetCloudEnvEnvReply, error) {
	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}
	envId := in.GetId()
	if envId <= 0 {
		return nil, InvalidId.Error()
	}
	cloudEnv := cloudEnvService.Get(envId)
	if cloudEnv.Id <= 0 {
		return nil, NotFoundRecord.Error()
	}

	return &pb.GetCloudEnvEnvReply{
		CloudEnv: createPbCloudEnv(cloudEnv),
	},nil
}

// 获取用户服务腾讯云环境列表
func (s Service) GetCloudEnvList(ctx context.Context,
	in *pb.GetCloudEnvListRequest) (*pb.GetCloudEnvListEnvReply, error) {

	user := ctx.Value("user").(*model.User)
	if !settingService.UserIsAdmin(user) {
		return nil, NoAuthority.Error()
	}

	offset := in.GetOffset()
	limit := in.GetLimit()
	status := in.GetStatus()
	allStatus := []int64{0, 1, 2, 3}
	statusIslegal := false
	for _, v := range allStatus {
		if status == v {
			statusIslegal = true
		}
	}
	if !statusIslegal {
		return nil, InvalidParams.ErrorParam("status", "status in [0 1 2 3]")
	}

	cloudEnvs, total := cloudEnvService.GetList(offset, limit, 0, status)
	var cloudEnvList []*pb.CloudEnv
	for _, cloudEnv := range cloudEnvs {
		cloudEnvList = append(cloudEnvList, &pb.CloudEnv{
			Id:            cloudEnv.Id,
			UserId:        cloudEnv.UserId,
			VaspLicenseId: cloudEnv.VaspLicenseId,
			InstanceIp:    cloudEnv.InstanceIp,
			Status:        cloudEnv.Status,
			CreateAt:      cloudEnv.CreateAt,
			UpdateAt:      cloudEnv.UpdateAt,
		})
	}
	return &pb.GetCloudEnvListEnvReply{
		CloudEnv: cloudEnvList,
		Total:    total,
	}, nil
}
