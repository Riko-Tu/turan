package tencentCloud

import (
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/spf13/viper"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ocr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ocr/v20181119"
)

var ocrClient *ocr.Client

const (
	ocrEndpoint = "ocr.tencentcloudapi.com"

	// region, ocr目前不支持南京, 选择广州
	ocrRegion = "ap-guangzhou"
)

func SetupOcr() {
	secretId := viper.GetString("tencentCloud.secretId")
	secretKey := viper.GetString("tencentCloud.secretKey")

	credential := common.NewCredential(secretId, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = ocrEndpoint
	var err error
	ocrClient, err = ocr.NewClient(credential, ocrRegion, cpf)
	if err != nil {
		log.Fatal(fmt.Sprintf("get tencentcloud OCR client failed:%s", err.Error()))
	}
}

// 识别结果
type CharacterRet struct {
	*ocr.GeneralBasicOCRResponse
	err error
}

// 图片文字识别
func Character(imageUrls []string) (characterRetItems []*CharacterRet) {
	if len(imageUrls) == 0 {
		return
	}

	for _, imageUrl := range imageUrls {
		request := ocr.NewGeneralBasicOCRRequest()
		request.ImageUrl = common.StringPtr(imageUrl)
		response, err := ocrClient.GeneralBasicOCR(request)

		characterRet := &CharacterRet{
			GeneralBasicOCRResponse: response,
			err:                     err,
		}
		characterRetItems = append(characterRetItems, characterRet)
	}
	return
}

// 获取解析字符串
func GetDetectedText(characterRetItems []*CharacterRet) (detectedTextItems []*string) {
	for _, characterRet := range characterRetItems {
		if characterRet.err != nil {
			continue
		}
		textDetections := characterRet.Response.TextDetections
		for _, textDetection := range textDetections {
			detectedTextItems = append(detectedTextItems, textDetection.DetectedText)
		}
	}
	return
}
