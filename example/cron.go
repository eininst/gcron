package main

import (
	"context"
	"fmt"
	"github.com/eininst/gcron"
	"syscall"
)

func main() {
	cron := gcron.New(gcron.WithSignal(syscall.SIGTERM, syscall.SIGINT))

	//I will execute every 5 seconds.
	cron.Task("*/5 * * * * * *", func(ctx context.Context) error {
		fmt.Println("done")
		return nil
	})

	cron.Spin()
}
