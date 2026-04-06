package main

import (
	"log"

	"github.com/marang/photo-roller/cmd/photoroller"
)

func main() {
	if err := photoroller.Execute(); err != nil {
		log.Fatal(err)
	}
}
