// main.go
package main

import (
	"log"
	"os"

	"github.com/zaynkorai/gemini-fullstack-langgraph-quickstart/api"
)

func main() {

	port := os.Getenv("PORT")
	s := api.NewServer()
	frontendBuildDir := "../frontend/dist"

	s.SetupFrontend(frontendBuildDir)

	if err := s.Start(port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
