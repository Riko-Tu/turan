package task

import (
	"TEFS-BE/pkg/cache"
	"context"
	"fmt"
	"github.com/RichardKnop/machinery/v1/backends/result"
	mchTasks "github.com/RichardKnop/machinery/v1/tasks"
	"time"
)

const LatexHistoryHandleJobKey = "LatexHistoryHandle"

const ETA = (60 * 5) + 10

func SendLatexHistoryHandleJob(ctx context.Context, retryCount int, name string, latexId int64, eta *time.Time, args ...mchTasks.Arg) (*result.AsyncResult, error) {

	// redis设置发送任务锁，设置锁的超时时间
	redisCli := cache.GetRedis()
	success, err := redisCli.SetNX(fmt.Sprintf("latex.%d.versionLocak", latexId), 1, time.Second * 240).Result()
	if err != nil {
		return nil, err
	}
	if !success {
		return nil, nil
	}

	task, err := mchTasks.NewSignature(name, args)
	if err != nil {
		return nil, err
	}
	task.RetryCount = retryCount
	task.ETA = eta
	task.UUID = fmt.Sprintf("latex_%d", latexId)
	task.Headers = make(map[string]interface{})
	task.RoutingKey = queue
	return taskCenter.SendTaskWithContext(ctx, task)
}

func LatexHistoryHandleJob(latexId int64) error {
	fmt.Println("start task")
	fmt.Println(latexId)
	time.Sleep(time.Second * time.Duration(10))
	fmt.Println("end task")
	return nil
}
