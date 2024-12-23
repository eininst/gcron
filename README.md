# gcron

[![Go Reference](https://pkg.go.dev/badge/github.com/eininst/gcron.svg)](https://pkg.go.dev/github.com/eininst/gcron)
[![License](https://img.shields.io/github/license/eininst/gcron.svg)](LICENSE)

基于 Go 语言实现的定时任务调度框架，支持使用类似 **cron** 表达式定义任务，并提供简单易用的 **Task** 注册、任务执行与优雅停止能力。  
通过 **gcron**，你可以快速实现周期性任务调度、定时脚本执行，或搭配 Redis 分布式锁实现分布式环境下的任务互斥调度。

## 功能特性

1. **Cron 表达式**
    - 使用类似 cron 表达式（如 `0 * * * * *`）定义任务触发时间，支持秒级别调度。
    - 内置 [github.com/gorhill/cronexpr](https://github.com/gorhill/cronexpr) 做解析，更加灵活。

2. **任务注册 (Task)**
    - 通过 `Task(expr, func)` 注册定时任务，可添加任意多个任务。
    - 每个任务均独立在 Goroutine 中执行，互不影响。

3. **分布式互斥（可选）**
    - 提供可选 Redis 配置，通过 `SetNX` 实现分布式锁，确保多实例环境下同一任务只在一个节点执行。
    - 根据是否设置 `RedisUrl` 启用或关闭此特性。

4. **优雅关闭 (Graceful Shutdown)**
    - 内部捕捉系统信号（可定制），在关闭时会取消所有任务并等待执行结束。
    - 避免任务正在运行时被强行中断。

5. **简洁易扩展**
    - 仅依赖少量库 (如 `cronexpr`、`go-redis/redis`)，集成简单。
    - 支持通过选项自定义名称、信号列表、Redis 地址等，满足多种应用场景。

---

## 安装

```bash
go get github.com/eininst/gcron
```

## 使用示例
下面展示 gcron 的几个主要用例，包括基础调度和分布式场景下的互斥任务。

### 1. 基本用法
```go
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
```

* 表达式 `"*/5 * * * * *"` 表示每 5 秒执行一次任务。
* 若需要多个任务，可多次调用 `c.Task(expr, func)`。

### 2. 分布式互斥场景
在多节点环境下，如果不想让相同任务在多个节点同时执行，可启用 Redis 分布式锁：
```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/eininst/gcron"
)

func main() {
    c := gcron.New(
        // 配置 Redis 地址，启用分布式互斥
        gcron.WithRedisUrl("redis://127.0.0.1:6379/0"),
        // 设置任务名称前缀
        gcron.WithName("JOB_DEMO"),
    )

    // 这个任务在多个节点都部署时，只有一个节点会真正执行
    c.Task("* * * * * * *", func(ctx context.Context) error {
        fmt.Println("Distributed Cron Task:", time.Now())
        return nil
    })

    c.Spin()
}
```
* 内部会通过 `SetNX` + 过期时间来做互斥控制。
* 如果你没设置 `WithRedisUrl`，则不会启用互斥特性。


### 3. 自定义信号与手动关闭
如果想监听 SIGINT、SIGTERM 等自定义信号，或`手动调用 Shutdown()`，可参考以下示例：

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/eininst/gcron"
)

func main() {
    c := gcron.New(
        gcron.WithSignals(syscall.SIGINT, syscall.SIGTERM),
    )

    c.Task("*/3 * * * * * *", func(ctx context.Context) error {
        fmt.Println("Task runs, press Ctrl+C to stop...")
        return nil
    })

    // 手动监听 SIGINT 并关闭
    go func() {
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT)
        <-quit
        fmt.Println("Received SIGINT, shutting down gcron...")
        c.Shutdown()
    }()

    c.Spin()
}
```
* `WithSignals(...)` 可替换默认只监听 SIGTERM 的行为。
* `Shutdown()` 会取消所有任务上下文并等待它们结束后退出。


> See [example](/example)

## 开发与贡献
欢迎参与本项目开发和提交 Issue/PR：

* Issue: 遇到问题或有新需求，可在 Issues 反馈
* PR: 修复 Bug 或实现新特性后，可发起 Pull Request 贡献给社区

## License

*MIT*