package main

import (
	"TEFS-BE/pkg/creator/api/router"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net"
	"os/exec"
	"runtime"
	"time"
)

var openUrl = "http://81.71.9.72:8080/login?port=%d"
//var openUrl = "http://127.0.0.1:8081/login?port=%d"
//var openUrl = "http://118.195.139.156:8080/login?port=%d"
//var openUrl = "https://tefscloud.com/login?port=%d"

// 获取一个随机未占用端口
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// 打开浏览器访问网址
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		//cmd = exec.Command("cmd.exe", "/c", "set http_proxy=http://127.0.0.1:12639")
		//cmd = exec.Command("cmd.exe", "/c", "set https_proxy=http://127.0.0.1:12639")
		cmd = exec.Command("cmd.exe", "/c", "start", url)
	case "darwin":
		//cmd = exec.Command("export http_proxy=http://127.0.0.1:12639")
		//cmd = exec.Command("export https_proxy=http://127.0.0.1:12639")
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("don't know how to open things on %s platform", runtime.GOOS)
	}
	return cmd.Run()
}

// @title Tefs 用户安装腾讯云 API
// @version V0.1
// @description Tefs 用户创建腾讯云环境
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@bullteam.cn
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /api
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	engine := gin.Default()
	// cors设置
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"POST, GET, OPTIONS, PUT, DELETE, UPDATE"},
		AllowHeaders:     []string{"Origin, X-Requested-With, Content-Type, Accept, Authorization"},
		ExposeHeaders:    []string{"Content-Length, Access-Control-Allow-Origin, " +
			"Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		MaxAge: 12 * time.Hour,
	}))


	
	router.Setup(engine)
	port, err := getFreePort()
	if err != nil {
		log.Error(err.Error())
	}
	url := fmt.Sprintf(openUrl, port)
	if err := openBrowser(url); err != nil {
		log.Fatal(err.Error())
	}
	log.Fatal(engine.Run(fmt.Sprintf(":%d", port)).Error())
}
