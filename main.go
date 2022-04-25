package main

import (
	"log"

	"github.com/magicniko/rp/cli"
)

func main() {
	app := cli.New()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
