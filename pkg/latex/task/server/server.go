package main

import (
	"TEFS-BE/pkg/cache"
	"TEFS-BE/pkg/database"
	"TEFS-BE/pkg/latex/task"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"strings"
)

func setup() {
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
	database.Setup()
	cache.Setup()
	task.Setup()
	zerolog.SetGlobalLevel(zerolog.Level(0))
	log.Setup()
}

func main() {
	setup()
	if err := task.Worker.Launch(); err != nil {
		log.Fatal(fmt.Sprintf("failed to task worker serve: %v", err))
	}
}
