package contextrootbad

import (
	"context"
	"os"
	"os/signal"
)

var packageRoot = context.Background // want `SOT-ENG-010`

func functionValueRoot() {
	localRoot := context.TODO // want `SOT-ENG-010`
	_ = localRoot()
}

func notifyWithFunctionValue() {
	root := context.Background // want `SOT-ENG-010`
	ctx, stop := signal.NotifyContext(root(), os.Interrupt)
	_ = ctx
	stop()
}
