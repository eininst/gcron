package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/eininst/gcron"
)

func main() {
	c := gcron.New(
		// 配置 Redis 地址，启用分布式互斥
		gcron.WithRedisUrl("redis://127.0.0.1:6379/0"),
		// 设置任务名称前缀
		gcron.WithName("JOB_DEMO"),

		gcron.WithSignals(syscall.SIGINT, syscall.SIGTERM),
	)

	c.Task("*/3 * * * * * *", func(ctx context.Context) error {
		fmt.Println("Task runs, press Ctrl+C to stop...")
		return nil
	})

	c.Spin()
}
