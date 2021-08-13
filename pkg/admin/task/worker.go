package task

import (
	"fmt"
	"github.com/RichardKnop/logging"
	"github.com/RichardKnop/machinery/v1"
	mchConf "github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/log"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/spf13/viper"
	"io"
	"os"
	"time"
)

var (
	// 任务执行着
	Worker *machinery.Worker

	// 任务控制器
	taskCenter *machinery.Server

	// 注册的任务
	tasks map[string]interface{}

	queue string

	tmpExperimentBaseDir string
	tmpOszicarDir string

)

func init() {
	tasks = make(map[string]interface{})
	tasks[MonitoringExperimentEnvFunc] = MonitoringExperimentEnv
	tasks[MonitoringExperimentComputeFunc] = MonitoringExperimentCompute
	tasks[DeleteExperimentFunc] = DeleteExperiment
}

// 初始化设置
func Setup() {
	redisPassword := viper.GetString("redis.auth")
	redisHost := viper.GetString("redis.host")
	brokerDB := viper.GetString("redis.taskBrokerDB")
	resultBackendDB := viper.GetString("redis.taskResultBackendDB")
	tmpExperimentBaseDir = viper.GetString("tmpExperimentBaseDir")
	tmpOszicarDir = viper.GetString("tmpOszicarBaseDir")

	queue = viper.GetString("task.queue")             // 队列名
	concurrency := viper.GetInt("task.concurrency")    // 并发数
	consumerTag := viper.GetString("task.consumerTag") // 标签

	broker := fmt.Sprintf("redis://%s@%s/%s", redisPassword, redisHost, brokerDB)
	resultBackend := fmt.Sprintf("redis://%s@%s/%s", redisPassword, redisHost, resultBackendDB)

	// log设置,开发输出到屏幕。其他输出到配置文件，按天切割，保留n天（配置）
	development := viper.GetBool("isDevelopment")
	var out, errOut io.Writer
	if development {
		out = os.Stdout
		errOut = os.Stderr
	} else {
		logPath := viper.GetString("log.path")
		maxCount := viper.GetUint("log.maxCount")
		writer, err := rotatelogs.New(
			logPath+".%Y%m%d",
			rotatelogs.WithLinkName(logPath),          // 生成软链，指向最新日志文件
			rotatelogs.WithRotationTime(time.Hour*24), // 日志切割时间间隔
			rotatelogs.WithRotationCount(maxCount),    // 日志最大保存个数
		)
		if err != nil {
			panic(err)
		}
		out = writer
		errOut = writer
	}
	logger := logging.New(out, errOut, new(logging.ColouredFormatter))
	log.DEBUG = logger[logging.DEBUG]
	log.INFO = logger[logging.INFO]
	log.WARNING = logger[logging.WARNING]
	log.ERROR = logger[logging.ERROR]
	log.FATAL = logger[logging.FATAL]

	cnf := &mchConf.Config{
		Broker:        broker,
		DefaultQueue:  queue,
		ResultBackend: resultBackend,
	}

	var err error
	taskCenter, err = machinery.NewServer(cnf)
	if err != nil {
		panic(err)
	}
	if err := taskCenter.RegisterTasks(tasks); err != nil {
		panic(err)
	}
	Worker = taskCenter.NewWorker(consumerTag, concurrency)
}
