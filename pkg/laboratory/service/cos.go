package service

import (
	pb "TEFS-BE/pkg/laboratory/proto"
	"context"
	"fmt"
	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
)

// 按cos路径获取所有权限
func (s Service) GetCosTmpSecret(ctx context.Context,
	in *pb.GetCosTmpSecretRequest) (*pb.GetCosTmpSecretReply, error) {
	cosDir := in.GetCosPath()
	secret, err := cosService.GetTmpSecretForAll([]string{cosDir})
	if err != nil {
		return nil, err
	}
	credential := &pb.CosCredential{
		TmpSecretID:  secret.Credentials.TmpSecretID,
		TmpSecretKey: secret.Credentials.TmpSecretKey,
		SessionToken: secret.Credentials.SessionToken,
		Bucket:       cosService.Bucket,
		Region:       cosService.Region,
		StartTime:    int64(secret.StartTime),
		ExpiredTime:  int64(secret.ExpiredTime),
		Expiration:   secret.Expiration,
	}
	return &pb.GetCosTmpSecretReply{CosCredential: credential, CosBaseUrl:cosBaseUrl}, nil
}

// 获取cos上传临时密钥
func (s Service) GetCosUploadTmpSecret(ctx context.Context,
	in *pb.GetCosUploadTmpSecretRequest) (*pb.GetCosUploadTmpSecretReply, error) {

	opType := in.GetOpType().String()
	var secret *sts.CredentialResult
	var err error
	switch opType {
	case "experiment":
		cosDir := in.GetExperimentDir()
		copyCosDir := in.GetCopyExperimentDir()
		if len(copyCosDir) > 0 {
			secret, err = cosService.GetUploadTmpSecret([]string{cosDir, copyCosDir})
		} else {
			secret, err = cosService.GetUploadTmpSecret([]string{cosDir})
		}
		if err != nil {
			return nil, fmt.Errorf("GetUploadTmpSecret err:%s", err.Error())
		}
	default:
		return nil, fmt.Errorf("invalid op_type")
	}

	credential := &pb.CosCredential{
		TmpSecretID:  secret.Credentials.TmpSecretID,
		TmpSecretKey: secret.Credentials.TmpSecretKey,
		SessionToken: secret.Credentials.SessionToken,
		Bucket:       cosService.Bucket,
		Region:       cosService.Region,
		StartTime:    int64(secret.StartTime),
		ExpiredTime:  int64(secret.ExpiredTime),
		Expiration:   secret.Expiration,
	}
	return &pb.GetCosUploadTmpSecretReply{CosCredential: credential, CosBaseUrl:cosBaseUrl}, nil
}

// 获取cos下载权限临时密钥
func (s Service) GetCosDownloadTmpSecret(ctx context.Context,
	in *pb.GetCosDownloadTmpSecretRequest) (*pb.GetCosDownloadTmpSecretReply, error) {

	userId := in.GetUserId()
	if userId <= 0 {
		return nil, fmt.Errorf("invalid userId")
	}
	cosPaths := []string{fmt.Sprintf("/users/%d/*", userId)}
	secret, err := cosService.GetDownloadTmpSecret(cosPaths)
	if err != nil {
		return nil, fmt.Errorf("GetUploadTmpSecret err:%s", err.Error())
	}

	credential := &pb.CosCredential{
		TmpSecretID:  secret.Credentials.TmpSecretID,
		TmpSecretKey: secret.Credentials.TmpSecretKey,
		SessionToken: secret.Credentials.SessionToken,
		Bucket:       cosService.Bucket,
		Region:       cosService.Region,
		StartTime:    int64(secret.StartTime),
		ExpiredTime:  int64(secret.ExpiredTime),
		Expiration:   secret.Expiration,
	}
	return &pb.GetCosDownloadTmpSecretReply{CosCredential: credential, CosBaseUrl:cosBaseUrl}, nil
}

// 获取删除的临时权限
func (s Service) GetCosDeleteTmpSecret(ctx context.Context,
	in *pb.GetCosDeleteTmpSecretRequest) (*pb.GetCosDeleteTmpSecretReply, error) {

	userId := in.GetUserId()
	experimentId := in.GetExperimentId()
	if userId <= 0 {
		return nil, fmt.Errorf("invalid userId")
	}
	if experimentId <= 0 {
		return nil, fmt.Errorf("invalid experimentId")
	}

	cosPaths := []string{fmt.Sprintf("/users/%d/experiments/%d/*", userId, experimentId)}
	secret, err := cosService.GetDeleteTmpSecret(cosPaths)
	if err != nil {
		return nil, fmt.Errorf("GetDeleteTmpSecret err:%s", err.Error())
	}
	credential := &pb.CosCredential{
		TmpSecretID:  secret.Credentials.TmpSecretID,
		TmpSecretKey: secret.Credentials.TmpSecretKey,
		SessionToken: secret.Credentials.SessionToken,
		Bucket:       cosService.Bucket,
		Region:       cosService.Region,
		StartTime:    int64(secret.StartTime),
		ExpiredTime:  int64(secret.ExpiredTime),
		Expiration:   secret.Expiration,
	}
	return &pb.GetCosDeleteTmpSecretReply{CosCredential: credential, CosBaseUrl:cosBaseUrl}, nil
}