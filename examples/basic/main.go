package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gostafa/inputstats"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		cancel()
	}()

	statsCh, err := inputstats.Start(ctx, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	for stats := range statsCh {
		fmt.Printf(
			"keys=%d moves=%d left=%d right=%d\n",
			stats.KeyboardClicks,
			stats.MouseMoves,
			stats.LeftClicks,
			stats.RightClicks,
		)
	}
}
