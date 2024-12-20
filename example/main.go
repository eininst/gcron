package main

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorhill/cronexpr"
	"github.com/panjf2000/ants/v2"
	"time"
)

func main() {
	w := &redis.Options{}
	println(w)
	pool, _ := ants.NewPool(10)
	println(pool)
	// 定义一个 Cron 表达式
	expr := "*/5 * * * * *" // 每 5 分钟执行一次

	// 解析 Cron 表达式
	cron, err := cronexpr.Parse(expr)
	if err != nil {
		fmt.Printf("Error parsing cron expression: %v\n", err)
		return
	}

	// 获取当前时间
	now := time.Now()

	// 计算下一个执行时间
	nextTime := cron.Next(now)
	fmt.Printf("Current time: %v\n", now)
	fmt.Printf("Next execution time: %v\n", nextTime)

	// 获取未来的多个执行时间
	for i := 0; i < 5; i++ {
		nextTime = cron.Next(nextTime)
		fmt.Printf("Future execution time %d: %v\n", i+1, nextTime)
	}
}
