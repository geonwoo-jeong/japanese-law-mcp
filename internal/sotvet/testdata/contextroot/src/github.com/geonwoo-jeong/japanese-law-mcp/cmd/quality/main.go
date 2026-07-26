package main

import (
	. "context"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(Background(), os.Interrupt)
	_ = ctx
	stop()

	_ = Background() // want `SOT-ENG-010`

	todoContext, stopTODO := signal.NotifyContext(TODO(), os.Interrupt) // want `SOT-ENG-010`
	_ = todoContext
	stopTODO()

	root := Background // want `SOT-ENG-010`
	aliasedContext, stopAliased := signal.NotifyContext(root(), os.Interrupt)
	_ = aliasedContext
	stopAliased()

	func() {
		nestedContext, stopNested := signal.NotifyContext(Background(), os.Interrupt) // want `SOT-ENG-010`
		_ = nestedContext
		stopNested()
	}()
}

func unrelatedRoot() {
	_ = Background() // want `SOT-ENG-010`
}
