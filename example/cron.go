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
