package main

import (
	pb "TEFS-BE/pkg/laboratory/proto"
	laboratory "TEFS-BE/pkg/laboratory/service"
	"TEFS-BE/pkg/log"
	tc "TEFS-BE/pkg/tencentCloud"
	"TEFS-BE/pkg/tencentCloud/batchCompute"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	qvpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"strings"
)

const port = 50051

func setup() {
	//config.SetLaboratoryEnv()
	// env config
	viper.SetEnvPrefix("TEFS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.AutomaticEnv()
	_ = viper.BindEnv("tencentCloud.secretId")
	_ = viper.BindEnv("tencentCloud.secretKey")
	_ = viper.BindEnv("tencentCloud.region")
	_ = viper.BindEnv("tencentCloud.appId")
	_ = viper.BindEnv("tencentCloud.cos.bucket")
	_ = viper.BindEnv("tencentCloud.securityGroupId")
	_ = viper.BindEnv("tencentCloud.account")
	_ = viper.BindEnv("tencentCloud.projectId")

	// Set up log
	zerolog.SetGlobalLevel(zerolog.Level(0))
	log.Setup()

	// batchCompute set up
	batchCompute.Setup()
	laboratory.Setup()
}

var cidrBlocks = []string{"146.56.192.59", "118.195.139.156", "81.71.9.72"}
var protocol = "tcp"
var addOpenPorts = []string{"5000-8000"}

// 更新安全组
func updateSecurityGroup() (e error) {
	vpc := tc.Vpc{
		Credential: &tc.Credential{
			SecretId:  viper.GetString("tencentCloud.secretId"),
			SecretKey: viper.GetString("tencentCloud.secretKey"),
		},
		Region: viper.GetString("tencentCloud.region"),
	}
	securityGroupId := viper.GetString("tencentCloud.securityGroupId")
	accept := "ACCEPT"
	securityGroupPolicySet, e := vpc.QuerySecurityGroupPolicies(securityGroupId)
	if e != nil {
		return
	}
	openPolicy := make(map[string]bool)
	for _, c := range cidrBlocks {
		for _, p := range addOpenPorts {
			key := fmt.Sprintf("%s:%s:%s", c, protocol, p)
			openPolicy[key] = false
		}
	}

	ingress := securityGroupPolicySet.Ingress
	for _, v := range ingress {
		if *v.Action != "ACCEPT" {
			continue
		}
		findKey := fmt.Sprintf("%s:%s:%s", *v.CidrBlock, *v.Protocol, *v.Port)
		if _, ok := openPolicy[findKey]; ok {
			openPolicy[findKey] = true
		}
	}
	ingressPolicyList := []*qvpc.SecurityGroupPolicy{}
	policyType := "ingress"
	for k, v := range openPolicy {
		if !v {
			tmp := strings.Split(k, ":")
			ingressPolicyList = append(ingressPolicyList, &qvpc.SecurityGroupPolicy{
				Protocol:  &tmp[1],
				Port:      &tmp[2],
				CidrBlock: &tmp[0],
				Action:    &accept,
			}, )
		}
	}
	if len(ingressPolicyList) > 0 {
		if e = vpc.AddSecurityGroupPolicies(securityGroupId, policyType, ingressPolicyList); e != nil {
			return
		}
	}
	return
}

func main() {
	// tefs laboratory grpc服务
	setup()

	// 更新安全组，失败重试，最大3次
	for i := 0; i < 3; i++ {
		if err := updateSecurityGroup(); err != nil {
			fmt.Println(err.Error())
			log.Error(err.Error())
			continue
		}
		break
	}

	// 开启nfs服务
	for i := 0; i < 3; i++ {
		if err := laboratory.StartNFS(); err != nil {
			fmt.Println(err.Error())
			log.Error(err.Error())
			continue
		}
		break
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to listen: %v", err))
	}
	s := grpc.NewServer()
	pb.RegisterLaboratoryServer(s, laboratory.Service{})
	reflection.Register(s)
	log.Info(fmt.Sprintf("start laboratory service, 127.0.0.1:%d", port))
	if err := s.Serve(lis); err != nil {
		log.Fatal(fmt.Sprintf("failed to serve: %v", err))
	}
}
