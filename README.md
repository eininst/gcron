# gcron

`A simple scheduling framework based on cronexpr, supporting distributed execution with Redis.`

## ⚙ Installation

```text
go get -u github.com/eininst/gcron
```

### Base example
```go
package main

import (
	"context"
	"fmt"
	"github.com/eininst/gcron"
)

func main() {
	cron := gcron.New()

	//I will execute every 5 seconds.
	cron.Handler("*/5 * * * * * *", func(ctx context.Context) error {
		fmt.Println("done")
		return nil
	})

	cron.Spin()
}
```


### Multiple instances example
```go
package main

import (
	"context"
	"fmt"
	"github.com/eininst/gcron"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	//Simulate starting multiple instances
	crons := []gcron.Cron{}
	
	for i := 0; i < 10; i++ {
		go func() {
			cron := gcron.New(gcron.WithRedisUrl("redis://127.0.0.1:6379/0"))
			
			//I will execute every 5 seconds.
			cron.Handler("*/5 * * * * * *", func(ctx context.Context) error {
				fmt.Println("done")
				return nil
			})

			crons = append(crons, cron)
			cron.Spin()
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	for _, cron := range crons {
		cron.Shutdown()
	}
}

```
> See [example](/example)

## License

*MIT*
