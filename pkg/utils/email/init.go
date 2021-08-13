package email

import "github.com/spf13/viper"

type _EmailConfig struct {
	user     string
	password string
	host     string
	port     int
}

var emailConfig *_EmailConfig

func Setup() {
	emailConfig = &_EmailConfig{
		user:     viper.GetString("email.qq.user"),
		password: viper.GetString("email.qq.password"),
		host:     viper.GetString("email.qq.host"),
		port:     viper.GetInt("email.qq.port"),
	}
}
