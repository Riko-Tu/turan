package service

import (
	pb "TEFS-BE/pkg/laboratory/proto"
	"TEFS-BE/pkg/log"
	"context"
	"encoding/json"
	"fmt"
)

// 获取region下cvm list
func (s Service) GetCvmList(ctx context.Context, in *pb.GetCvmListRequest) (*pb.GetCvmListReply, error) {
	filters := make(map[string]string)
	offset := in.GetOffset()
	limit := in.GetLimit()
	zone := in.GetZone()
	if len(zone) > 0 {
		filters["zone"] = zone
	}
	response, err := cvmService.DescribeInstances(filters, offset, limit)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	responseByte, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("response to json failed:%s", err.Error()))
	}
	return &pb.GetCvmListReply{
		DescribeCvmListJson: string(responseByte),
	}, nil
}

// 获取region下可用zone
func (s Service) AvailableZoneList(ctx context.Context,
	in *pb.AvailableZoneListRequest) (*pb.AvailableZoneListReply, error) {

	zoneInfoList, err := cvmService.GetAvailableZone()
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	zoneNames := []string{}
	for _, zone := range zoneInfoList {
		zoneNames = append(zoneNames, *zone.Zone)
	}
	return &pb.AvailableZoneListReply{
		ZoneList: zoneNames,
	}, nil
}

// 按cmvImageId 查询 cmvImage
func (s Service) CvmImage(ctx context.Context, in *pb.CvmImageRequest) (*pb.CvmImageReply, error) {
	imageId := in.GetCvmImageId()
	response, err := cvmService.GetImage(imageId)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	responseByte, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("response to json failed:%s", err.Error()))
	}
	return &pb.CvmImageReply{
		ResponseJson: string(responseByte),
		CloudAccount: cloudAccount,
	}, nil
}
