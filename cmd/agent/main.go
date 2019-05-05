package main

import (
	"../../pkg/agent"
	"log"
	"os"
)


func main() {
	server, err := agent.NewServer(&agent.DefaultConfig)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if err := server.Run(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}