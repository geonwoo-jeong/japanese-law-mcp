package contextrootbad

import . "context"

func dotImportedRoots() {
	_ = Background() // want `SOT-ENG-010`
	_ = TODO()       // want `SOT-ENG-010`
}
