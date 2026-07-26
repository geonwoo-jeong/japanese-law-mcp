package main

import (
	"log"
	"os"
)

func main() {
	if false {
		os.Exit(1)
		log.Fatal("開発用コマンドを終了します") // want `SOT-ENG-014`

		exit := os.Exit // want `SOT-ENG-014`
		exit(1)
	}
}

func helper() {
	os.Exit(1) // want `SOT-ENG-014`
}
