package main

import (
	"TEFS-BE/pkg/admin/auth"
	pb "TEFS-BE/pkg/admin/proto"
	admin "TEFS-BE/pkg/admin/service"
	"TEFS-BE/pkg/admin/task"
	ws "TEFS-BE/pkg/admin/ws"
	"TEFS-BE/pkg/cache"
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/tencentCloud"
	"TEFS-BE/pkg/tencentCloud/ses"
	"TEFS-BE/pkg/tencentCloud/sms"
	"TEFS-BE/pkg/utils/email"
	"context"
	"flag"
	"fmt"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	vaspkitServer = ws.VaspkitServer{}
)

func setup() {
	// 1.Set up log level
	zerolog.SetGlobalLevel(zerolog.Level(0))
	// 2.Set up configuration
	config := "./config/admin-local.yaml"
	viper.SetConfigFile(config)
	content, err := ioutil.ReadFile(config)
	if err != nil {
		panic(fmt.Sprintf("Read config file fail: %s", err.Error()))
	}
	// Replace environment variables
	err = viper.ReadConfig(strings.NewReader(os.ExpandEnv(string(content))))
	if err != nil {
		panic(fmt.Sprintf("Parse config file fail: %s", err.Error()))
	}
	// log
	log.Setup()
	// 3.Set up redis
	cache.Setup()
	// 4.Set up sms
	sms.Setup()
	// 5.Set up mysql
	database.Setup()
	// 6 Set up AUTH QQ wx login
	auth.Setup()
	// 7 Set up email
	email.Setup()
	ses.Setup()
	// task
	task.Setup()
	// 8 cos
	admin.Setup()
	// ocr
	tencentCloud.SetupOcr()
	// vaspkit
	vaspkitCvmIP := viper.GetString("vaspkit.cvm.ip")
	vaspkitCvmPort := viper.GetInt("vaspkit.cvm.port")
	vaspkitCvmUser := viper.GetString("vaspkit.cvm.user")
	vaspkitCvmPassword := viper.GetString("vaspkit.cvm.password")
	vaspkitServer.Setup(vaspkitCvmIP, vaspkitCvmUser,vaspkitCvmPassword, vaspkitCvmPort)
}

// AuthInterceptor 认证拦截器，对以authorization为头部，形式为`bearer token`的Token进行验证
func AuthInterceptor(ctx context.Context) (context.Context, error) {
	var ctxKey string = "user"
	pairs := metadata.Pairs("token", "")
	if err := grpc.SetTrailer(ctx, pairs); err != nil {
		log.Error(err.Error())
	}

	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		log.Error(err.Error())
		return ctx, nil
		//return nil, admin.GetTokenFailed.Error()
	}
	user, tokenClaims, isExpired, err := auth.ParseJwtToken(token)
	if err != nil {
		return nil, admin.ParseTokenFailed.Error()
	}
	if user.Id <= 0 {
		return nil, admin.NotFoundUser.Error()
	}

	// redis token json
	redisCli := cache.GetRedis()
	tokenRedisKey := auth.GetTokenRedisKey(user.Id)
	oldTokenRedisKey := tokenRedisKey + ".old"
	storageTokens, err := redisCli.MGet(tokenRedisKey, oldTokenRedisKey).Result()
	if err != nil {
		log.Error(err.Error())
		return nil, admin.GetStorageTokenFailed.Error()
	}
	curStorageToken := storageTokens[0]
	oldStorageToken := storageTokens[1]

	// 存储token过期
	if curStorageToken == nil {
		return nil, admin.InvalidToken.Error()
	}

	// 存储token未过期
	if token != curStorageToken.(string) {
		// token过期任有30s宽限期
		if oldStorageToken == nil || token != oldStorageToken.(string) {
			return nil, admin.InvalidToken.Error()
		}
		newCtx := context.WithValue(ctx, ctxKey, user)
		return newCtx, nil
	} else {
		// token 未过期,刷新存储token有效期
		if !isExpired {
			if err := auth.RefreshTokenTime(user.Id); err != nil {
				log.Error(err.Error())
			}
			newCtx := context.WithValue(ctx, ctxKey, user)
			return newCtx, nil
		}
		// token过期, 存储token未过期。设置过期token 30s宽限期
		ok, err := redisCli.SetNX(oldTokenRedisKey, token, time.Second * 30).Result()
		if err != nil {
			log.Error(err.Error())
		}
		// 设置成功，更新token
		if ok {
			newToken, err := auth.CreateJwtToken(tokenClaims["nickname"], tokenClaims["figure_url"],
				tokenClaims["account"], tokenClaims["way"], user.Id)
			if err != nil {
				log.Error(err.Error())
				return nil, admin.RefreshToken1.Error()
			}
			if err := auth.SaveToken(user.Id, newToken); err != nil {
				return nil, admin.RefreshToken2.Error()
			}
			pairs := metadata.Pairs("token", newToken)
			if err := grpc.SetTrailer(ctx, pairs); err != nil {
				log.Error(err.Error())
			}
		}

		// 使用context.WithValue添加了值后，可以用Value(key)方法获取值
		newCtx := context.WithValue(ctx, ctxKey, user)
		return newCtx, nil
	}
}

func mai1n() {
	// tefs 统一后台grpc服务

	// 初始化组件
	setup()

	port := viper.Get("webport")

	// grpc service
	s := grpc.NewServer(
		// 拦截器
		grpc.StreamInterceptor(
			grpc_auth.StreamServerInterceptor(AuthInterceptor),
		),
		grpc.UnaryInterceptor(
			grpc_auth.UnaryServerInterceptor(AuthInterceptor),
		),
	)
	pb.RegisterAdminServer(s, admin.Service{})
	reflection.Register(s)

	// grpc web
	wrappedGrpc := grpcweb.WrapServer(s,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
	)
	var serveAddr string
	flag.StringVar(&serveAddr,
		"address",
		fmt.Sprintf(":%d", port),
		fmt.Sprintf(":serve address - e.g. :%d", port))
	flag.Parse()

	grpcweb.WithCorsForRegisteredEndpointsOnly(true)

	// 添加restful api处理函数
	http.HandleFunc("/api/login/wx/callback", auth.WxLoinCallBack)
	http.HandleFunc("/api/login/qq/callback", auth.QQLoginCallback)
	http.HandleFunc("/api/ws/vaspkit", vaspkitServer.WsVaspkit)
	http.HandleFunc("/api/ws/fileTree", vaspkitServer.WsFileTree)
	http.HandleFunc("/api/ws/fileContent", vaspkitServer.WsFileContent)
	//http.HandleFunc("/api/cos/tmpSecret", vaspkitServer.GetCosTmpSecret)
	//http.Handle("/", http.FileServer(http.Dir("./demo")))

	srv := &http.Server{
		Addr:         serveAddr,
		WriteTimeout: time.Second * 32,
		ReadTimeout:  time.Second * 32,
		IdleTimeout:  time.Second * 60,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if wrappedGrpc.IsGrpcWebRequest(r) {
				// NOT WORKING!
				w.Header().Set("Access-Control-Allow-Origin", "*")
				wrappedGrpc.ServeHTTP(w, r)
				return
			}

			// Fall back to other servers.
			http.DefaultServeMux.ServeHTTP(w, r)
		}),
	}
	log.Info(fmt.Sprintf("start admin service, 127.0.0.1:%d", port))
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(fmt.Sprintf("failed to serve: %v", err))
	}
}

func main() {
	// tefs 统一后台grpc服务
	setup()
	lis, err := net.Listen("tcp", ":8013")
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to listen: %v", err))
	}
	s := grpc.NewServer(
		// 拦截器
		grpc.StreamInterceptor(
			grpc_auth.StreamServerInterceptor(AuthInterceptor),
		),
		grpc.UnaryInterceptor(
			grpc_auth.UnaryServerInterceptor(AuthInterceptor),
		),
	)
	pb.RegisterAdminServer(s, admin.Service{})
	reflection.Register(s)
	log.Info("start admin service, 127.0.0.1:8013")
	if err := s.Serve(lis); err != nil {
		log.Fatal(fmt.Sprintf("failed to serve: %v", err))
	}
}