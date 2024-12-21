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

type Context struct {
	context.Context
}
type Function func(ctx context.Context) error

type job struct {
	CronLine   string
	Fc         Function
	Expression *cronexpr.Expression
	LastTime   time.Time
}

type Cron interface {
	Handler(expr string, fc Function)
	Spin()
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
		RedisUrl: "",
		Name:     "JOB",
	}
	options.Apply(opts)

	return &cron{
		jobs:    []*job{},
		stop:    make(chan int, 1),
		options: *options,
	}
}

func (c *cron) Handler(expr string, fc Function) {
	expression, err := cronexpr.Parse(expr)
	if err != nil {
		glog.Fatal(err)
	}

	c.jobs = append(c.jobs, &job{
		CronLine:   expr,
		Fc:         fc,
		Expression: expression,
		LastTime:   time.Now().Truncate(time.Second),
	})
}

func (c *cron) Spin() {
	ctx := context.Background()

	if c.options.RedisUrl != "" {
		c.rcli = NewRedisClient(c.options.RedisUrl, len(c.jobs))
	}

	for _, j := range c.jobs {
		ctxCancel, cancel := context.WithCancel(ctx)
		c.cancels = append(c.cancels, cancel)

		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.execute(ctxCancel, j)
		}()
	}

	go func() {
		quit := make(chan os.Signal)

		if len(c.options.Signals) == 0 {
			signal.Notify(quit, syscall.SIGTERM)
		} else {
			signal.Notify(quit, c.options.Signals...)
		}

		<-quit
		glog.Printf("\033[33mShutdown...\033[0m")
		c.Shutdown()
	}()

	glog.Printf("Running... (%v jobs)", len(c.jobs))

	<-c.stop

	glog.Printf("\033[32mGraceful shutdown success!\033[0m")
}

func (c *cron) Shutdown() {
	defer func() { c.stop <- 1 }()

	for _, cancel := range c.cancels {
		cancel()
	}

	c.wg.Wait()
}

func getFunctionName(fn interface{}) string {
	pc := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	if pc == nil {
		return ""
	}
	return pc.Name()
}

func callFc(fcName string, try func()) {
	defer func() {
		if r := recover(); r != nil {
			glog.Printf("\033[31merror by [%v]: %v\033[0m", fcName, r)
		}
	}()

	try()
}

func (c *cron) execute(ctx context.Context, j *job) {
	firstNextTime := j.Expression.Next(time.Now())
	ticker := time.NewTicker(time.Until(firstNextTime))

	fcName := getFunctionName(j.Fc)
	name := c.options.Name
	ex := time.Second * 5

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			callFc(fcName, func() {
				nextTime := j.Expression.Next(time.Now())
				ticker.Reset(time.Until(nextTime))

				if c.rcli != nil {
					jobKey := fmt.Sprintf("%v_%v_%v", name,
						fcName, nextTime.Format(TIME_FORMAT))

					ok, _er := c.rcli.SetNX(ctx, jobKey, 1, ex).Result()
					if _er != nil {
						panic(_er)
					}
					if ok {
						er := j.Fc(ctx)

						if er != nil {
							panic(er)
						}
					}
				} else {
					er := j.Fc(ctx)

					if er != nil {
						panic(er)
					}
				}
			})
		}
	}
}
