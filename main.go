package main

import (
	"log"

	"github.com/Parthiba-Hazra/golivesync/internal/server"
)

func main() {
	if err := server.StartServer(); err != nil {
		log.Fatalln(err.Error())
	}
}
