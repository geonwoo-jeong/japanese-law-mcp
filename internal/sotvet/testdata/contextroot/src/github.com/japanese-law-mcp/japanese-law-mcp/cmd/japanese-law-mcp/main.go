package main

import (
	"context"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	_ = ctx
	stop()
}

func unrelatedRoot() {
	_ = context.TODO() // want `SOT-ENG-010`
}
