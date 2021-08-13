package client

import (
	pb "TEFS-BE/pkg/laboratory/proto"
	"TEFS-BE/pkg/log"
	"context"
	"encoding/json"
	"fmt"
	batch "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/batch/v20170312"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"github.com/tencentyun/cos-go-sdk-v5"
	"google.golang.org/grpc"
	"net/http"
	"net/url"
	"time"
)

var (
	requestTimeout = time.Second * 60
)

// rpc客户端
func GetClient(address string) (pb.LaboratoryClient, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Error(fmt.Sprintf("did not connect:%s", err.Error()))
		return nil, err
	}
	c := pb.NewLaboratoryClient(conn)
	return c, nil
}

// 获取cos临时密钥
func GetCosTmpSecret(client pb.LaboratoryClient,
	cosPath string) (cosTmpSecret *pb.CosCredential, cosBaseUrl *string, err error) {

	in := pb.GetCosTmpSecretRequest{CosPath: cosPath}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.GetCosTmpSecret(ctx, &in)
	if err != nil {
		return nil, nil, err
	}
	return reply.CosCredential, &reply.CosBaseUrl, nil
}

// 获取上传cos临时密钥，指定上传文件目录范围
func GetCosUploadTmpSecret(client pb.LaboratoryClient,
	experimentCosPath ...string) (cosTmpSecret *pb.CosCredential, cosBaseUrl *string, err error) {

	in := pb.GetCosUploadTmpSecretRequest{
		OpType:        0,
		ExperimentDir: experimentCosPath[0],
	}
	if len(experimentCosPath) > 1 {
		in.CopyExperimentDir = experimentCosPath[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.GetCosUploadTmpSecret(ctx, &in)
	if err != nil {
		return nil, nil, err
	}
	return reply.CosCredential, &reply.CosBaseUrl, nil
}

// 按用户id获取获取下载cos临时密钥
func GetCosDownloadTmpSecret(client pb.LaboratoryClient,
	userId int64) (cosTmpSecret *pb.CosCredential, cosBaseUrl *string, err error) {

	in := pb.GetCosDownloadTmpSecretRequest{
		UserId: userId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.GetCosDownloadTmpSecret(ctx, &in)
	if err != nil {
		return nil, nil, err
	}
	return reply.CosCredential, &reply.CosBaseUrl, nil
}

// 删除实验的cos文件
func GetCosDeleteTmpSecret(client pb.LaboratoryClient,
	userId, experimentId int64) (cosTmpSecret *pb.CosCredential, cosBaseUrl *string, err error) {

	in := pb.GetCosDeleteTmpSecretRequest{
		UserId:       userId,
		ExperimentId: experimentId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.GetCosDeleteTmpSecret(ctx, &in)
	if err != nil {
		return nil, nil, err
	}
	return reply.CosCredential, &reply.CosBaseUrl, nil
}

// 获取实验cos下的文件列表
func GetExperimentCosFiles(cosClient *cos.Client, filePrefix string) (files []string, err error) {
	opt := &cos.BucketGetOptions{
		Prefix: filePrefix,
		// 获取cos文件名列表最大数量。因为cos接口未设置分页。这里指的一个比较高的值。用户基本上不会到达这个值
		MaxKeys: 9999999999,
	}
	result, _, err := cosClient.Bucket.Get(context.Background(), opt)
	if err != nil {
		return nil, err
	}
	for _, v := range result.Contents {
		files = append(files, v.Key)
	}
	return
}

// 删除实验cos
func DeleteExperimentCos(cosClient *cos.Client, files []string) error {
	objects := []cos.Object{}
	for _, v := range files {
		objects = append(objects, cos.Object{Key: v})
	}
	opt := &cos.ObjectDeleteMultiOptions{
		Objects: objects,
	}
	_, _, err := cosClient.Object.DeleteMulti(context.Background(), opt)
	if err != nil {
		return err
	}
	return nil
}

// 获取cos临时客户端
func GetCosTmpClient(tmpSecret *pb.CosCredential, cosBaseUrl string) (*cos.Client, error) {
	u, err := url.Parse(cosBaseUrl)
	if err != nil {
		return nil, err
	}
	u.Scheme = "https"
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     tmpSecret.TmpSecretID,
			SecretKey:    tmpSecret.TmpSecretKey,
			SessionToken: tmpSecret.SessionToken,
		},
	})
	return client, nil
}

// 上传文件到cos
func UploadCos(cosTmpSecret *pb.CosCredential, cosBaseUrl string, files map[string]string) error {
	client, err := GetCosTmpClient(cosTmpSecret, cosBaseUrl)
	if err != nil {
		return err
	}
	for k, v := range files {
		_, err := client.Object.PutFromFile(context.Background(), k, v, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// 创建实验环境
func CreateExperimentEnv(client pb.LaboratoryClient, nodeNum, diskSize int64,
	inputCosPath, cvmImageId, zone, instanceType, diskType string) (batchEnvId string, err error) {

	in := pb.CreateExperimentEnvRequest{
		CvmImageId:   cvmImageId,
		NodeNum:      nodeNum,
		InputCosPath: inputCosPath,
		Zone:         zone,
		InstanceType: instanceType,
		DiskType:     diskType,
		DiskSize:     diskSize,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.CreateExperimentEnv(ctx, &in)
	if err != nil {
		return "", err
	}
	return reply.EnvId, nil
}

// 查询实验环境详情
func QueryExperimentEnv(client pb.LaboratoryClient, batchEnvId string) (*batch.DescribeComputeEnvResponse, error) {
	in := pb.QueryExperimentEnvRequest{
		BatchEnvId: batchEnvId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.QueryExperimentEnv(ctx, &in)
	if err != nil {
		return nil, err
	}
	describeEnv := &batch.DescribeComputeEnvResponse{}
	if err := json.Unmarshal([]byte(reply.DescribeEnvJson), describeEnv); err != nil {
		return nil, err
	}
	return describeEnv, nil
}

// 查询实验环境详情列表
func QueryExperimentEnvList(client pb.LaboratoryClient,
	offset, limit uint64) (*batch.DescribeComputeEnvsResponse, error) {

	in := pb.QueryExperimentEnvListRequest{
		Offset: offset,
		Limit:  limit,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.QueryExperimentEnvList(ctx, &in)
	if err != nil {
		return nil, err
	}
	describeEnvList := &batch.DescribeComputeEnvsResponse{}
	if err := json.Unmarshal([]byte(reply.JsonResponse), describeEnvList); err != nil {
		return nil, err
	}
	return describeEnvList, nil
}

// 删除计算环境
func DeleteExperimentEnv(client pb.LaboratoryClient, batchEnvId string) error {
	in := pb.DeleteExperimentEnvRequest{
		BatchEnvId: batchEnvId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	_, err := client.DeleteExperimentEnv(ctx, &in)
	return err
}

// 提交实验
func SubmitExperiment(client pb.LaboratoryClient, name, description, b64RunCmd,
	cosPath, batchEnvId, zone string, timeout uint64) (jid *string, err error) {

	in := pb.SubmitExperimentRequest{
		Name:        name,
		Description: description,
		B64RunCmd:   b64RunCmd,
		CosPath:     cosPath,
		BatchEnvId:  batchEnvId,
		Timeout:     timeout,
		Zone:        zone,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.SubmitExperiment(ctx, &in)
	if err != nil {
		return nil, err
	}
	return &reply.Jid, nil
}

// 查询实验
func QueryExperiment(client pb.LaboratoryClient, jid string) (*batch.DescribeJobResponse, error) {
	in := pb.QueryExperimentRequest{
		Jid: jid,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.QueryExperiment(ctx, &in)
	if err != nil {
		return nil, err
	}
	describeJob := &batch.DescribeJobResponse{}
	if err := json.Unmarshal([]byte(reply.JobDetailsJson), describeJob); err != nil {
		return nil, err
	}
	return describeJob, nil
}

// 终止实验
func TerminateExperiment(client pb.LaboratoryClient, jid string) error {
	in := pb.TerminateExperimentRequest{
		Jid: jid,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	_, err := client.TerminateExperiment(ctx, &in)
	return err
}

// 删除实验
func DeleteExperiment(client pb.LaboratoryClient, jid string) error {
	in := pb.DeleteExperimentRequest{
		Jid: jid,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	_, err := client.DeleteExperiment(ctx, &in)
	return err
}

// 查询cvm详情list
func GetCvmList(client pb.LaboratoryClient, zone string, offset, limit int64) (*cvm.DescribeInstancesResponse, error) {
	in := pb.GetCvmListRequest{
		Offset: offset,
		Limit:  limit,
		Zone:   zone,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.GetCvmList(ctx, &in)
	if err != nil {
		return nil, err
	}
	describeInstancesResponse := &cvm.DescribeInstancesResponse{}
	if err = json.Unmarshal([]byte(reply.DescribeCvmListJson), describeInstancesResponse); err != nil {
		return nil, err
	}
	return describeInstancesResponse, nil
}

// 查询可用zone list
func AvailableZoneList(client pb.LaboratoryClient) ([]string, error) {
	in := pb.AvailableZoneListRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.AvailableZoneList(ctx, &in)
	if err != nil {
		return nil, err
	}
	return reply.ZoneList, nil
}

// 按cmvImageId 查询 cmvImage
func CvmImage(client pb.LaboratoryClient, cvmImageId string) ([]cvm.Image, string, error) {
	in := pb.CvmImageRequest{
		CvmImageId: cvmImageId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	reply, err := client.CvmImage(ctx, &in)
	if err != nil {
		return nil, "", err
	}
	image := []cvm.Image{}
	if err := json.Unmarshal([]byte(reply.ResponseJson), &image); err != nil {
		return nil, "", err
	}
	return image, reply.CloudAccount, nil
}

// 创建console shell docker容器
func CreateWebConsole(client pb.LaboratoryClient, userId, projectId int64, secret, tefsUrl string) (port string, err error) {
	in := pb.CreateWebConsoleRequest{
		UserId:    userId,
		Secret:    secret,
		ProjectId: projectId,
		TefsUrl:   tefsUrl,
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout*time.Duration(5))
	defer cancel()
	reply, err := client.CreateWebConsole(ctx, &in)
	if err != nil {
		return "", err
	}
	return reply.Message, nil
}
