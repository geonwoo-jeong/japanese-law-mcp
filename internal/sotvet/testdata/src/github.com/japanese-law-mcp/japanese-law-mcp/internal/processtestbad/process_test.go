package processtestbad

import (
	"log"
	"os"
	"testing"
)

func TestForbiddenProcessExit(t *testing.T) {
	t.Parallel()

	if false {
		os.Exit(1)      // want `SOT-ENG-014`
		log.Fatal("失敗") // want `SOT-ENG-014`
	}
}
