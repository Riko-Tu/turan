package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/cache"
	labCli "TEFS-BE/pkg/laboratory/client"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/tencentCloud"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// echart 实验数据，获取画图类型
	poscarKey = "poscar"
	energyKey = "energy"
	eigenKey  = "eigen"
	dosKey    = "dos"
)
const (
	// echart 实验数据，获取项目类型
	demoExType = "demoEx"
	userExType = "userEx"
)

const (
	timeData      = "time"
	xmlName       = "vasprun.xml"
	kpointsName   = "KPOINTS"
	keyExpiration =  time.Second * 60 * 60 * 24 * 7
	chartRedisKey = "chart.user.%d.experiment.%d."
	demoExUserId  = 0
)

const (
	// 时间格式化
	cosTimeFormat   = "2006-01-02T15:04:05.000Z"
	redisTimeFormat = "2006-01-02 15:04:05 +0000 UTC"
)

var energyTypeArray = [3]string{"e_wo_entrp", "e_fr_energy", "e_0_energy"}
var dataTypeArray = [5]string{poscarKey, energyKey, eigenKey, dosKey}

func (s Service) GetImportChartData(ctx context.Context,
	in *pb.GetImportChartDataRequest) (*pb.GetImportChartDataReply, error) {
	cosPath := in.GetCosPath()
	user := ctx.Value("user").(*model.User)

	// 解析cosPath地址，判断用户是否匹配
	userDir, _ := parseImportCosPath(cosPath)
	userIdStr := userDir[strings.Index(userDir, "_")+1:]
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		return nil, ErrUser.Error()
	}

	if user.Id != userId {
		return nil, ErrUser.Error()
	}

	chartService := getChartCosService(bucket, region)
	fileLocalDir := tmpChartDataFilePathDir + userDir + "/"
	// 创建数据文件所在文件夹
	if err := os.MkdirAll(fileLocalDir, 0755); err != nil {
		log.Error(err.Error())
		return nil, GetCosFileFailed.Error()
	}

	// 从cos上拉取数据，并放入redis缓存
	reply, err := downloadImportChartData(cosPath, fileLocalDir, chartService)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	return reply, nil
}

func (s Service) GetChartData(ctx context.Context,
	in *pb.GetChartDataRequest) (*pb.GetChartDataReply, error) {
	user := ctx.Value("user").(*model.User)
	experimentType := in.GetExType().String()
	experimentId := in.GetExperimentId()
	dataType := in.GetDataType().String()
	if err := dataTypeParamValid(dataType); err != nil {
		return nil, ErrDataType.Error()
	}
	// 只有能带实验用到 getEigenFromXml()
	key := in.GetKey().String()
	key, err := keyParamValid(key)
	if err != nil {
		return nil, ErrKey.Error()
	}

	if experimentId <= 0 {
		return nil, InvalidId.Error()
	}

	if experimentType == userExType {
		// 非演示项目
		experiment, err := experimentService.Get(experimentId)
		if err != nil {
			log.Error(err.Error())
			return nil, QueryDbFailed.Error()
		}
		if experiment.UserId != user.Id {
			return nil, NotFoundRecord.Error()
		}

		rpcCli, err := labCli.GetClient(experiment.LaboratoryAddress)
		if err != nil {
			log.Error(err.Error())
			return nil, GetCosFileFailed.Error()
		}
		cosTmpSecret, cosUrl, err := labCli.GetCosTmpSecret(rpcCli, "*")
		if err != nil {
			log.Error(err.Error())
			return nil, GetCosFileFailed.Error()
		}
		cosCli, err := tencentCloud.TmpSecretGetCosCli(cosTmpSecret.TmpSecretID, cosTmpSecret.TmpSecretKey, cosTmpSecret.SessionToken, *cosUrl)
		if err != nil {
			log.Error(err.Error())
			return nil, GetCosFileFailed.Error()
		}

		cosEndpoint := fmt.Sprintf("/users/%d/experiments/%d/", user.Id, experiment.Id)
		fileLocalDir := chartDataFilePathDir + cosEndpoint
		if err := os.MkdirAll(fileLocalDir, 0755); err != nil {
			log.Error(err.Error())
			return nil, GetCosFileFailed.Error()
		}

		u, _ := url.Parse(experiment.CosBasePath)
		u.Scheme = "https"
		cosCli.BaseURL = &cos.BaseURL{BucketURL: u}

		cosExperimentPath :=  fmt.Sprintf("/users/%d/experiments/%d/", user.Id, experiment.Id)

		opt := &cos.BucketGetOptions{
			Prefix:  cosExperimentPath[1:] + xmlName,
			MaxKeys: 99999,
		}
		result, _, err := cosCli.Bucket.Get(context.Background(), opt)
		if err != nil {
			log.Error(err.Error())
			return nil, GetCosFileFailed.Error()
		}
		// cos文件最后修改时间
		if len(result.Contents) == 0 {
			return nil, ExpDataNotFound.Error()
		}
		xmlLastModifiedAt, err := time.Parse(cosTimeFormat, result.Contents[0].LastModified)
		if err != nil {
			log.Error(err.Error())
			return nil, ExpDataNotFound.Error()
		}

		cacheKey := fmt.Sprintf("echar.experiment.%d", experiment.Id)
		ret, err := cache.Get(cacheKey)
		if err == nil {
			echar := make(map[string]string)
			if err := json.Unmarshal([]byte(ret), &echar); err == nil {
				if echar["xmlLastModifiedAt"] == xmlLastModifiedAt.String() {
					return &pb.GetChartDataReply{Data: echar[dataType]}, nil
				}
			}
		} else {
			log.Error(err.Error())
		}

		_, err = cosCli.Object.GetToFile(context.Background(), cosExperimentPath + xmlName, fileLocalDir+xmlName, nil)
		if err != nil {
			log.Error(err.Error())
		}
		_, err = cosCli.Object.GetToFile(context.Background(), cosExperimentPath+kpointsName, fileLocalDir+kpointsName, nil)
		if err != nil {
			log.Error(err.Error())
			return nil, GetCosFileFailed.Error()
		}

		poscarDataBuff, err := getPoscarFromXml(fileLocalDir + xmlName)
		if err != nil {
			log.Error(err.Error())
			return nil, ParseXmlFailed.Error()
		}
		energyDataBuff, err := getEnergyFromXml(fileLocalDir+xmlName, energyTypeArray[0])
		if err != nil {
			log.Error(err.Error())
			return nil, ParseXmlFailed.Error()
		}
		eigenDataBuff, err := getEigenFromXml(fileLocalDir + xmlName)
		if err != nil {
			log.Error(err.Error())
			return nil, ParseXmlFailed.Error()
		}
		dosDataBuff, err := getDosFromXml(fileLocalDir + xmlName)
		if err != nil {
			log.Error(err.Error())
			return nil, ParseXmlFailed.Error()
		}

		redisData := make(map[string]string)
		redisData["energy"] = energyDataBuff.String()
		redisData["eigen"] = eigenDataBuff.String()
		redisData["dos"] = dosDataBuff.String()
		redisData["poscar"] = poscarDataBuff.String()
		redisData["xmlLastModifiedAt"] = xmlLastModifiedAt.String()

		data, err := json.Marshal(redisData)
		if err != nil {
			log.Error(err.Error())
			return nil, ToJsonFailed.Error()
		}
		if err := cache.Set(cacheKey, string(data), -1); err != nil {
			log.Error(err.Error())
		}
		return &pb.GetChartDataReply{Data: redisData[dataType]}, nil

	} else {
		// 演示项目
		demoFileKey := fmt.Sprintf(chartDemoExPathDir, experimentId)
		chartService := getChartCosService(chartDemoExBucket, chartDemoExRegion)

		//localBasePath := fmt.Sprintf("/users/%d/experiments/%d/", user.Id, experimentId)
		fileLocalDir := chartDataFilePathDir + demoFileKey
		// 创建数据文件所在文件夹
		if err := os.MkdirAll(fileLocalDir, 0755); err != nil {
			log.Error(err.Error())
			return nil, PathErr.Error()
		}

		// 从cos上拉取数据，并放入redis缓存
		if err = downloadXml(demoFileKey, fileLocalDir, demoExUserId, experimentId, chartService); err != nil {
			return nil, err
		}
	}

	redisCli := cache.GetRedis()

	var data, redisKeyPrefix string
	if experimentType == demoExType {
		// 演示项目实验
		redisKeyPrefix = getExDataRedisKey(demoExUserId, experimentId, dataType)
	} else {
		// 用户实验
		redisKeyPrefix = getExDataRedisKey(user.Id, experimentId, dataType)
	}
	if dataType == energyKey {
		data = redisCli.Get(redisKeyPrefix + "." + key).Val()
	} else {
		data = redisCli.Get(redisKeyPrefix).Val()
	}

	return &pb.GetChartDataReply{Data: data}, nil
}

func downloadImportChartData(fileKey, localPath string, chartService *tencentCloud.Cos) (reply *pb.GetImportChartDataReply, err error) {
	fileName := strings.Split(fileKey, "/")[2]
	// 查看xml文件
	xmlGetResult, err := chartService.GetObjectList(fileKey)
	if err != nil {
		log.Error(err.Error())
		return nil, GetCosFileFailed.Error()
	}
	// 该实验不存在vasprun.xml数据文件
	if len(xmlGetResult) == 0 {
		return nil, ExpDataNotFound.Error()
	}

	// redis数据不可用，需要重新从cos上拉取数据放入缓存
	err = chartService.Download(fileKey, localPath+fileName)
	//err = chartService.Download(fileName + kpointsName, localPath + kpointsName)
	if err != nil {
		return nil, err
	}

	// 解析poscar数据
	poscarDataBuff, err := getPoscarFromXml(localPath + fileName)
	if err != nil {
		return nil, ParseXmlFailed.Error()
	}

	// 解析energy中数据
	energyDataBuff, err := getEnergyFromXml(localPath+fileName, energyTypeArray[0])
	if err != nil {
		return nil, ParseXmlFailed.Error()
	}

	// 解析eigen数据
	eigenDataBuff, err := getEigenFromXml(localPath + fileName)
	if err != nil {
		return nil, ParseXmlFailed.Error()
	}

	// 解析dos数据
	dosDataBuff, err := getDosFromXml(localPath + fileName)
	if err != nil {
		return nil, ParseXmlFailed.Error()
	}

	return &pb.GetImportChartDataReply{
		PoscarData: poscarDataBuff.String(),
		EnergyData: energyDataBuff.String(),
		EigenData:  eigenDataBuff.String(),
		DosData:    dosDataBuff.String(),
	}, nil
}

func downloadXml(fileKey, localPath string, userId, experimentId int64, chartService *tencentCloud.Cos) (err error) {
	// 查看xml文件
	xmlGetResult, err := chartService.GetObjectList(fileKey + xmlName)
	if err != nil {
		log.Error(err.Error())
		return GetCosFileFailed.Error()
	}
	// 该实验不存在vasprun.xml数据文件
	if len(xmlGetResult) == 0 {
		return ExpDataNotFound.Error()
	}
	// 查看KPOINTS文件
	kpointsGetResult, err := chartService.GetObjectList(fileKey + kpointsName)
	if err != nil {
		log.Error(err.Error())
		return GetCosFileFailed.Error()
	}
	// 该实验不存在KPOINTS数据文件
	if len(kpointsGetResult) == 0 {
		return ExpDataNotFound.Error()
	}

	xmlMetaData := xmlGetResult[0]
	// 这里用的UTC时间，不是东八区，腾讯云cos返回的LastModified时间也是UTC时间
	xmlLastModifiedAt, err := time.Parse(cosTimeFormat, xmlMetaData.LastModified)
	if err != nil {
		return err
	}
	redisCli := cache.GetRedis()

	// 查看redis是否存在该数据
	timeKey := getExDataRedisKey(userId, experimentId, timeData)
	cacheTimeStr := redisCli.Get(timeKey).Val()
	if cacheTimeStr != "" {
		cacheTime, err := time.Parse(redisTimeFormat, cacheTimeStr)
		if err != nil {
			return err
		}
		// 判断redis中数据是否过时
		if !xmlLastModifiedAt.After(cacheTime) {
			// 数据没有过时，不需要重新下载数据文件
			return nil
		}
		delRedisChart(userId, experimentId, redisCli)
	}

	// redis数据不可用，需要重新从cos上拉取数据放入缓存
	err = chartService.Download(fileKey+xmlName, localPath+xmlName)
	err = chartService.Download(fileKey+kpointsName, localPath+kpointsName)

	if err != nil {
		return err
	}

	// 缓存进redis
	redisCli.Set(timeKey, xmlLastModifiedAt.String(), keyExpiration)

	// 解析poscar数据
	poscarDataBuff, err := getPoscarFromXml(localPath + xmlName)
	if err != nil {
		return ParseXmlFailed.Error()
	}
	poscarRedisKey := getExDataRedisKey(userId, experimentId, poscarKey)
	// 过期时间设置为0即永久保存
	redisCli.Set(poscarRedisKey, poscarDataBuff.String(), keyExpiration)

	// 解析energy中数据
	for _, k := range energyTypeArray {
		energyDataBuff, err := getEnergyFromXml(localPath+xmlName, k)
		if err != nil {
			return ParseXmlFailed.Error()
		}
		energyRedisKey := getExDataRedisKey(userId, experimentId, energyKey+"."+k)
		redisCli.Set(energyRedisKey, energyDataBuff.String(), keyExpiration)
	}

	// 解析eigen数据
	eigenDataBuff, err := getEigenFromXml(localPath + xmlName)
	if err != nil {
		return ParseXmlFailed.Error()
	}
	eigenRedisKey := getExDataRedisKey(userId, experimentId, eigenKey)
	redisCli.Set(eigenRedisKey, eigenDataBuff.String(), keyExpiration)

	// 解析dos数据
	dosDataBuff, err := getDosFromXml(localPath + xmlName)
	if err != nil {
		return ParseXmlFailed.Error()
	}
	dosRedisKey := getExDataRedisKey(userId, experimentId, dosKey)
	redisCli.Set(dosRedisKey, dosDataBuff.String(), keyExpiration)

	return
}

func getChartCosService(bucket, region string) (cosService *tencentCloud.Cos) {
	// 创建cos client
	cosService = &tencentCloud.Cos{
		Credential: &tencentCloud.Credential{
			AppId:     tencentCloudAppID,
			SecretId:  tencentCloudSecretID,
			SecretKey: tencentCloudSecretKey,
		},
		Region: region,
		Bucket: bucket,
	}
	return cosService
}

func parseCosPath(cosBasePath string) (bucket, region, dataDir string) {
	// CosBasePath的形式为 “cos://tefs-cos-0f4398-1300241787.cos.ap-nanjing.myqcloud.com/users/151/experiments/2027/”
	cosBasePath = strings.TrimPrefix(cosBasePath, "cos://")
	slashSeqIdx := strings.Index(cosBasePath, "/")

	baseUrl := cosBasePath[:slashSeqIdx]
	// 获取桶的名字, 例:tefs-cos-0f4398-1300241787
	bucket = strings.Split(baseUrl, ".")[0]
	// 获取桶所在区域, 例:ap-nanjing
	region = strings.Split(baseUrl, ".")[2]
	// users/151/experiments/2027/
	dataDir = cosBasePath[slashSeqIdx+1:]
	return
}

func parseImportCosPath(cosBasePath string) (userDir, fileName string) {
	// CosBasePath的形式为 “user_${USER_ID}/Echar/vasprun_${USER_ID}_${timeTemp}.xml”
	slashSeqIdx := strings.Index(cosBasePath, "/")
	userDir = cosBasePath[:slashSeqIdx]

	// Echar/vasprun_${USER_ID}_${timeTemp}.xml
	fileName = cosBasePath[slashSeqIdx+1:]
	return
}

func dataTypeParamValid(dataType string) error {
	if len(dataType) == 0 {
		return ErrDataType.Error()
	}
	for _, dt := range dataTypeArray {
		if dt == dataType {
			return nil
		}
	}
	return ErrDataType.Error()
}

func keyParamValid(key string) (res string, err error) {
	if len(key) == 0 {
		res = energyTypeArray[0]
	} else {
		// 参数key是否合法
		keyValid := false
		for _, t := range energyTypeArray {
			if key == t {
				keyValid = true
				res = key
			}
		}
		if !keyValid {
			return "", ErrKey.Error()
		}
	}
	return res, nil
}

// 获取数据在redis的key
func getExDataRedisKey(userId, experimentId int64, key string) string {
	return fmt.Sprintf(chartRedisKey, userId, experimentId) + key
}

func delRedisChart(userId, experimentId int64, redisCli *redis.Client) {
	keyPrefix := getExDataRedisKey(userId, experimentId, "")
	keyPrefixEnergy := getExDataRedisKey(userId, experimentId, energyKey+".")
	redisCli.Del(keyPrefix+poscarKey, keyPrefix+eigenKey, keyPrefix+dosKey, keyPrefix+timeData,
		keyPrefixEnergy+energyTypeArray[0], keyPrefixEnergy+energyTypeArray[1], keyPrefixEnergy+energyTypeArray[2])
}
