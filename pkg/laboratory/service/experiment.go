package service

import (
	pb "TEFS-BE/pkg/laboratory/proto"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/tencentCloud/batchCompute"
	"TEFS-BE/pkg/utils"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// 创建实验环境
func (s Service) CreateExperimentEnv(ctx context.Context,
	in *pb.CreateExperimentEnvRequest) (*pb.CreateExperimentEnvReply, error) {

	nodeNum := in.GetNodeNum()
	inputCosPath := in.GetInputCosPath()
	cvmImageId := in.CvmImageId
	zone := in.GetZone()
	instanceType := in.GetInstanceType()
	diskType := in.GetDiskType()
	diskSize := in.GetDiskSize()

	cosPath := cosBaseUrl + inputCosPath
	cvmPassword := utils.GeneratePassWord(16)

	log.Info(fmt.Sprintf("cvmPassword:%s", cvmPassword))
	log.Info(fmt.Sprintf("cosPath:%s", cosPath))

	batchEnvId, err := batchCompute.CreateEnv(nodeNum, cvmImageId, cvmPassword, cosPath, zone, instanceType, diskType, diskSize)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("create compute env failed:%s", err.Error()))
	}
	return &pb.CreateExperimentEnvReply{EnvId: *batchEnvId}, nil
}

// 查询实验环境详情
func (s Service) QueryExperimentEnv(ctx context.Context,
	in *pb.QueryExperimentEnvRequest) (*pb.QueryExperimentEnvReply, error) {

	batchEnvId := in.GetBatchEnvId()
	response, err := batchCompute.DescribeEnv(batchEnvId)
	if err != nil {
		return nil, err
	}
	ret, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("response to json str error:%s", err.Error()))
	}
	describeEnvJson := string(ret)
	return &pb.QueryExperimentEnvReply{
		DescribeEnvJson: describeEnvJson,
	}, nil
}

// 查询实验环境详情
func (s Service) QueryExperimentEnvList(ctx context.Context,
	in *pb.QueryExperimentEnvListRequest) (*pb.QueryExperimentEnvListReply, error) {

	response, err := batchCompute.DescribeEnvs(in.GetOffset(), in.GetLimit())
	if err != nil {
		return nil, err
	}
	return &pb.QueryExperimentEnvListReply{
		JsonResponse: response.ToJsonString(),
	}, nil
}

// 删除计算环境
func (s Service) DeleteExperimentEnv(ctx context.Context,
	in *pb.DeleteExperimentEnvRequest) (*pb.DeleteExperimentEnvReply, error) {

	batchEnvId := in.GetBatchEnvId()
	err := batchCompute.DeleteEnv(batchEnvId)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteExperimentEnvReply{
		Message: "ok",
	}, nil
}

// 提交实验
func (s Service) SubmitExperiment(ctx context.Context,
	in *pb.SubmitExperimentRequest) (*pb.SubmitExperimentReply, error) {

	b64RunCmd := in.GetB64RunCmd()
	cosPath := in.GetCosPath()
	zone := in.GetZone()
	batchEnvId := in.GetBatchEnvId()
	timeOut := in.GetTimeout()
	name := in.GetName()
	description := in.GetDescription()
	runCmd, err := base64.StdEncoding.DecodeString(b64RunCmd)
	if err != nil {
		return nil, fmt.Errorf("run cmd error")
	}

	// 提交任务
	jid, err := batchCompute.SubmitJob(string(runCmd), name, description,
		cosPath, batchEnvId, zone, &timeOut, cloudProjectId)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	// 返回提交成功
	return &pb.SubmitExperimentReply{Jid: *jid}, nil
}

// 查询实验
func (s Service) QueryExperiment(ctx context.Context,
	in *pb.QueryExperimentRequest) (*pb.QueryExperimentReply, error) {

	jid := in.GetJid()
	response, err := batchCompute.GetJobDetails(jid)
	if err != nil {
		return nil, err
	}
	ret, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("response to json str error:%s", err.Error()))
	}
	describeEnvJson := string(ret)
	return &pb.QueryExperimentReply{
		JobDetailsJson: describeEnvJson,
	}, nil
}

// 终止实验
func (s Service) TerminateExperiment(ctx context.Context,
	in *pb.TerminateExperimentRequest) (*pb.TerminateExperimentReply, error) {

	jid := in.GetJid()
	if err := batchCompute.TerminateJob(jid); err != nil {
		return nil, err
	}
	return &pb.TerminateExperimentReply{Message: "ok"}, nil
}

// 删除实验
func (s Service) DeleteExperiment(ctx context.Context,
	in *pb.DeleteExperimentRequest) (*pb.DeleteExperimentReply, error) {

	jid := in.GetJid()
	if err := batchCompute.DeleteJob(jid); err != nil {
		return nil, err
	}
	return &pb.DeleteExperimentReply{Message: "ok"}, nil
}
