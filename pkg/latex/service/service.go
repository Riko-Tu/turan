package main

import (
	"TEFS-BE/pkg/admin/auth"
	admin "TEFS-BE/pkg/admin/service"
	"TEFS-BE/pkg/cache"
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/latex"
	"TEFS-BE/pkg/latex/route"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/tencentCloud/ses"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func setup() {
	zerolog.SetGlobalLevel(zerolog.Level(0))
	config := "./config/admin-local.yaml"
	viper.SetConfigFile(config)
	content, err := ioutil.ReadFile(config)
	if err != nil {
		panic(fmt.Sprintf("Read config file fail: %s", err.Error()))
	}
	err = viper.ReadConfig(strings.NewReader(os.ExpandEnv(string(content))))
	if err != nil {
		panic(fmt.Sprintf("Parse config file fail: %s", err.Error()))
	}
	log.Setup()
	cache.Setup()
	database.Setup()
	auth.Setup()
	admin.Setup()
	latex.Setup()
	ses.Setup()
}

// @title TEFS latex API
// @version V0.1
// @description TEFS latex
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@bullteam.cn
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /api/doc
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	setup()
	port := viper.GetString("latex.port")
	fmt.Println(port)
	engine := gin.Default()
	// cors设置
	engine.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"POST, GET, OPTIONS, PUT, DELETE, UPDATE"},
		AllowHeaders: []string{"Origin, X-Requested-With, Content-Type, Accept, Authorization"},
		ExposeHeaders: []string{"Content-Length, Access-Control-Allow-Origin, " +
			"Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		MaxAge: 12 * time.Hour,
	}))

	router.Setup(engine)
	log.Fatal(engine.Run(fmt.Sprintf(":%s", port)).Error())
}
