package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"funny-nova/internal"
)

func main()  {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := internal.NewStorage("./nova.db")

	if storage == nil {
		panic("failed to init storage")
	}

	defer func()  {
		if err := storage.Close(); err != nil {
			fmt.Printf("storage.Close error: %v\n", err)
		}
	}()

	scheduler := internal.NewScheduler()
	scheduler.RegisterNodes()

	queue := internal.NewQueue()
	deamon := internal.NewDeamon(storage, scheduler, queue)

	sigFromChannel := make(chan os.Signal, 1)
	signal.Notify(sigFromChannel, os.Interrupt)

	go func() {
		sig := <-sigFromChannel
		if sig == os.Interrupt {
			cancel()
		}
	}()

	deamon.Run(ctx)
}

