package log

import (
	"fmt"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

const timeFormat = "2006-01-02 15:04:05"

var log zerolog.Logger

// 日志初始化设置
func Setup() {
	development := viper.GetBool("isDevelopment")
	var output zerolog.ConsoleWriter
	if development {
		output = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: timeFormat}
	} else {
		logPath := viper.GetString("log.path")
		maxCount := viper.GetUint("log.maxCount")
		writer, err := rotatelogs.New(
			logPath +".%Y%m%d",
			rotatelogs.WithLinkName(logPath),          // 生成软链，指向最新日志文件
			//rotatelogs.WithMaxAge(time.Second*60*3),  // 文件最大保存时间
			rotatelogs.WithRotationTime(time.Hour*24), 	// 日志切割时间间隔
			rotatelogs.WithRotationCount(maxCount), 	// 日志最大保存个数
		)
		if err != nil {
			panic(err)
		}
		output = zerolog.ConsoleWriter{Out: writer, TimeFormat: timeFormat}
	}

	zerolog.CallerSkipFrameCount = 3
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf(" | %s", i))
	}
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf(" | %s", i)
	}
	output.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf(" %s:", i)
	}
	output.FormatFieldValue = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%s ", i))
	}
	output.FormatCaller = func(i interface{}) string {
		var c string
		if cc, ok := i.(string); ok {
			c = cc
		}
		if len(c) > 0 {
			cwd, err := os.Getwd()
			if err == nil {
				c = strings.TrimPrefix(c, cwd)
				c = strings.TrimPrefix(c, "/")
			}
		}
		return "| " + c
	}
	log = zerolog.New(output).With().Timestamp().Logger()
}

//Debug : Level 0
func Debug(msg string) {
	log.Debug().Caller().Msg(msg)
}

//Info : Level 1
func Info(msg string) {
	log.Info().Caller().Msg(msg)
}

//Warn : Level 2
func Warn(msg string) {
	log.Warn().Caller().Msg(msg)
}

//Error : Level 3
func Error(msg string) {
	log.Error().Caller().Msg(msg)
}

//Fatal : Level 4
func Fatal(msg string) {
	log.Fatal().Caller().Msg(msg)
}
