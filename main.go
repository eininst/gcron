package gcron

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorhill/cronexpr"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var glog = log.New(os.Stderr, "[CRON] ", log.Lmsgprefix|log.Ldate|log.Ltime)

const TIME_FORMAT = "20060102150405"

type Function func(ctx context.Context) error

type job struct {
	CronLine   string
	Fc         Function
	FcName     string // 优化点1：存储函数名，避免每次反射
	Expression *cronexpr.Expression
}

type Cron interface {
	Task(expr string, fc Function)
	Spin()
	Shutdown()
}

type cron struct {
	jobs    []*job
	stop    chan int
	cancels []context.CancelFunc
	wg      sync.WaitGroup
	rcli    *redis.Client
	options Options
}

func New(opts ...Option) Cron {
	options := &Options{
		RedisUrl:     "",
		Name:         "JOB",
		LockExpire:   time.Second * 5,  // 默认5秒
		ExitWaitTime: time.Second * 10, //默认10秒
	}
	options.Apply(opts)

	return &cron{
		jobs:    make([]*job, 0),
		stop:    make(chan int, 1),
		options: *options,
	}
}

func (c *cron) Task(expr string, fc Function) {
	expression, err := cronexpr.Parse(expr)
	if err != nil {
		glog.Fatal(err)
	}

	fcName := getFunctionName(fc)

	c.jobs = append(c.jobs, &job{
		CronLine:   expr,
		Fc:         fc,
		FcName:     fcName,
		Expression: expression,
	})
}

func (c *cron) Spin() {
	ctx := context.Background()

	// 如果有 Redis URL，则初始化 Redis 客户端
	if c.options.RedisUrl != "" {
		c.rcli = NewRedisClient(c.options.RedisUrl, len(c.jobs))
	}

	// 创建每个任务的 Goroutine
	for _, j := range c.jobs {
		ctxCancel, cancel := context.WithCancel(ctx)
		c.cancels = append(c.cancels, cancel)

		c.wg.Add(1)
		go func(job *job) {
			defer c.wg.Done()
			c.execute(ctxCancel, job)
		}(j)
	}

	// 捕捉信号，用于优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		if len(c.options.Signals) == 0 {
			signal.Notify(quit, syscall.SIGTERM)
		} else {
			signal.Notify(quit, c.options.Signals...)
		}
		<-quit
		glog.Printf("Shutdown...")
		c.Shutdown()
	}()

	glog.Printf("Running... (%v jobs)", len(c.jobs))
	<-c.stop
}

func (c *cron) Shutdown() {
	defer func() { c.stop <- 1 }()
	for _, cancel := range c.cancels {
		cancel()
	}

	// 等待剩余任务处理完毕
	ctx, cancel := context.WithTimeout(context.Background(), c.options.ExitWaitTime)
	defer cancel()

	done := make(chan struct{}, 1)
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		glog.Printf("Graceful shutdown success")
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			glog.Println("Shutdown timeout exceeded")
		}
	}
}

func getFunctionName(fn Function) string {
	// 这里不再使用反射，直接打印内存地址也行，或者让调用方自行传函数名
	// 如果一定要通过反射获取，可以保留，但只在 Task() 阶段调用一次
	return runtime.FuncForPC(runtime.FuncForPC((reflect.ValueOf(fn).Pointer())).Entry()).Name()
}

func callFc(fcName string, try func()) {
	defer func() {
		if r := recover(); r != nil {
			glog.Printf("\033[31merror by [%v]: %v\033[0m", fcName, r)
		}
	}()
	try()
}

// execute 改用循环 + time.Sleep 模式，避免频繁重置 ticker
func (c *cron) execute(ctx context.Context, j *job) {
	for {
		// 计算下一次执行时间
		now := time.Now()
		nextTime := j.Expression.Next(now)
		duration := nextTime.Sub(now)
		if duration < 0 {
			duration = 0
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(duration):
			// 到达执行时间，安全执行任务
			callFc(j.FcName, func() {
				// 如果使用 Redis 互斥
				if c.rcli != nil {
					jobKey := fmt.Sprintf("%v_%v_%v",
						c.options.Name,
						j.FcName,
						nextTime.Format(TIME_FORMAT))

					ok, err := c.rcli.SetNX(ctx, jobKey, 1, c.options.LockExpire).Result()
					if err != nil {
						panic(err)
					}
					if ok {
						if err := j.Fc(ctx); err != nil {
							panic(err)
						}
					}
				} else {
					// 无 Redis 直接执行
					if err := j.Fc(ctx); err != nil {
						panic(err)
					}
				}
			})
		}
	}
}
