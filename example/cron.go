package main

import (
	"context"
	"fmt"
	"time"

	"github.com/eininst/gcron"
)

func main() {
	// 1. 创建 Cron 实例，可使用默认选项
	c := gcron.New()

	// 2. 注册一个每 5 秒执行一次的任务
	c.Task("*/5 * * * * * *", func(ctx context.Context) error {
		fmt.Println("Task runs every 5 seconds:", time.Now())
		return nil
	})

	// 3. 启动调度并阻塞等待信号
	c.Spin()
	// 收到信号后(默认SIGTERM)，会优雅关闭所有正在执行的任务
}
