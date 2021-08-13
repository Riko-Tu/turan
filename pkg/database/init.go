package database

import (
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/jinzhu/gorm"

	// mysql初始化导入
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/spf13/viper"
)

var db *gorm.DB
var NotFoundErr = gorm.ErrRecordNotFound

func Setup() {
	var err error
	host := viper.GetString("database.mysql.host")
	user := viper.GetString("database.mysql.user")
	password := viper.GetString("database.mysql.password")
	name := viper.GetString("database.mysql.name")
	charset := viper.GetString("database.mysql.charset")
	isDevelopment := viper.GetBool("isDevelopment")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=%s&parseTime=True&loc=Local", user, password, host, name, charset)
	log.Debug(dsn)
	db, err = gorm.Open("mysql", dsn)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to connect mysql %s", err.Error()))
	} else {
		db.DB().SetMaxIdleConns(viper.GetInt("database.mysql.pool.min"))
		db.DB().SetMaxOpenConns(viper.GetInt("database.mysql.pool.max"))
		if isDevelopment {
			db.LogMode(true)
		}
	}
	log.Info("Successfully connect to database")
}

func GetDb() *gorm.DB {
	return db
}

func Shutdown() error {
	log.Info("Closing database's connections")
	return db.Close()
}
