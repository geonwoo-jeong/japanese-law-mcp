package contextrootbad

import (
	"context"
	"os"
	"os/signal"
)

func notifyOutsideCommandMain() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt) // want `SOT-ENG-010`
	_ = ctx
	stop()
}
