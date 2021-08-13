package history

import (
	"TEFS-BE/pkg/cache"
	"encoding/json"
	"fmt"
)

type Transform struct {
	Insert    string `json:"insert"`
	NumDelete int64  `json:"num_delete"`
	Position  int64  `json:"position"`
}

type Change struct {
	Document  map[string]string `json:"document"`
	Time      int64             `json:"time"`
	Transform Transform         `json:"transform"`
	UserId    int64             `json:"user_id"`
	LatexId   int64             `json:"latex_id"`
	Operation string            `json:"operation"`
}

func (c Change)toRedis() error {
	if c.LatexId == 0 {
		return fmt.Errorf("latex id is 0")
	}
	key := fmt.Sprintf(LatexHistoryQueue, c.LatexId)
	cli := cache.GetRedis()
	data, _ := json.Marshal(c)
	return cli.RPush(key, data).Err()
}

const LatexHistoryQueue = "latex_history.%d"

func (c *Change)TransFormToCache(changeInfo []byte) error {
	if err := json.Unmarshal(changeInfo, c); err != nil {
		return err
	}
	c.Operation = "TransForm"
	return c.toRedis()
}

// 文件操作
const (
	Delete = "delete"
	Upload = "upload"
	Move   = "move"
	Rename = "rename"
)

func (c *Change)FileChangeToCache() error {
	return c.toRedis()
}

