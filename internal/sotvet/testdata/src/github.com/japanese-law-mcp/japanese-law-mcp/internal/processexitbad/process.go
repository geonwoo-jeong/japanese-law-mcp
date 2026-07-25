package processexitbad

import (
	standardlog "log"
	system "os"
)

func terminate(code int) {
	system.Exit(code)               // want `SOT-ENG-014`
	standardlog.Fatal("失敗")         // want `SOT-ENG-014`
	standardlog.Fatalf("失敗: %d", 1) // want `SOT-ENG-014`
	standardlog.Fatalln("失敗")       // want `SOT-ENG-014`
}
