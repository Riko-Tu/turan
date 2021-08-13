package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"context"
	"time"
	"unicode/utf8"
)

var tagService = model.TagService{}

const maxTag = 50

// 创建实验标签
func (s Service) CreateTag(ctx context.Context, in *pb.CreateTagRequest) (*pb.CreateTagReply, error) {
	user := ctx.Value("user").(*model.User)
	tagName := in.GetName()
	tagColour := in.GetColour()

	if len(tagName) == 0 || len(tagColour) == 0 {
		return nil, InvalidParams.ErrorParam("tagName,tagColour",
			"not found params tagName or tagColour")
	}
	if utf8.RuneCountInString(tagName) > 10 {
		return nil, InvalidParams.ErrorParam("tagName", "tag name max len 10")
	}
	if utf8.RuneCountInString(tagColour) > 50 {
		return nil, InvalidParams.ErrorParam("tagColour", "tag colour max len 50")
	}

	userTagTotal, err := tagService.GetUserTagTotal(user.Id)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if userTagTotal >= maxTag {
		return nil, TotalLimit.Error()
	}

	tag, err := tagService.GetByUserAndName(user.Id, tagName)
	if err != nil && err != database.NotFoundErr {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}
	if tag != nil && tag.Id > 0 {
		return nil, DuplicateTagName.Error()
	}

	nowTime := time.Now().Unix()
	newTag := &model.Tag{
		UserId:   user.Id,
		Name:     tagName,
		Colour:   tagColour,
		CreateAt: nowTime,
		UpdateAt: nowTime,
	}
	if err := tagService.Create(newTag).Error; err != nil {
		log.Error(err.Error())
		return nil, CreateRecordFailed.Error()
	}

	return &pb.CreateTagReply{
		Message: "ok",
	}, nil
}

// getTagList
func (s Service) GetTagList(ctx context.Context, in *pb.GetTagListRequest) (*pb.GetTagListReply, error) {
	user := ctx.Value("user").(*model.User)
	offset := in.GetOffset()
	limit := in.GetLimit()

	tags, total, err := tagService.GetList(user.Id, offset, limit)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}

	var data []*pb.Tag

	for _, v := range tags {
		data = append(data, &pb.Tag{
			Id:     v.Id,
			Name:   v.Name,
			Colour: v.Colour,
		})
	}
	return &pb.GetTagListReply{
		Tags:  data,
		Total: total,
	}, nil
}
