package main

import (
	"TEFS-BE/pkg/latex/route"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)



// @title TEFS latex API
// @version V0.1
// @description TEFS latex
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@bullteam.cn
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /api/latex
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
	log.Fatal(engine.Run(fmt.Sprintf(":%d", 5006)).Error())
}
