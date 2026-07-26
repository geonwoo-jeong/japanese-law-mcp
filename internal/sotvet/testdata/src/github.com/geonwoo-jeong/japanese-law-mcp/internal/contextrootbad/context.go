package contextrootbad

import ctx "context"

func roots() {
	_ = ctx.Background() // want `SOT-ENG-010`
	_ = ctx.TODO()       // want `SOT-ENG-010`
}
