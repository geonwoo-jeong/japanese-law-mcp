package processexitbad

import (
	"log"
	"os"
)

var packageExit = os.Exit // want `SOT-ENG-014`

func functionValueExits() {
	localExit := os.Exit // want `SOT-ENG-014`
	localExit(1)

	localFatal := log.Fatal // want `SOT-ENG-014`
	localFatal("失敗")
}
