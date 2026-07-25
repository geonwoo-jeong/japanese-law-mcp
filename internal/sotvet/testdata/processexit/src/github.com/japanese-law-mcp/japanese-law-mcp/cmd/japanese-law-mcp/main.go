package main

import "os"

func main() {
	if false {
		os.Exit(0)
	}

	func() {
		os.Exit(1) // want `SOT-ENG-014`
	}()
}

func helperExit() {
	os.Exit(1) // want `SOT-ENG-014`
}
