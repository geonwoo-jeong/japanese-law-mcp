package processexitbad

import . "os"

func dotImportedExit() {
	Exit(1) // want `SOT-ENG-014`
}
