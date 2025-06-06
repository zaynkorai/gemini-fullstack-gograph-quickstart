package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zaynkorai/gemini-fullstack-langgraph-quickstart/agent"
	"github.com/zaynkorai/gemini-fullstack-langgraph-quickstart/api"
)

func main() {
	port := os.Getenv("PORT")
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: GEMINI_API_KEY not set. Using mock LLM responses for the agent.")
		os.Exit(1)
	}

	config := &agent.Configuration{
		QueryGeneratorModel:    "gemini-2.0-flash",
		ReasoningModel:         "gemini-2.0-flash",
		NumberOfInitialQueries: 3,
		MaxResearchLoops:       2,
	}

	workflow, err := agent.NewWorkflow(config, apiKey)
	if err != nil {
		fmt.Printf("Failed to initialize workflow: %v\n", err)
		return
	}

	fmt.Println("\n--- Executing the Research Agent Graph ---")
	initialState := &agent.OverallState{
		Messages: []agent.Message{
			agent.HumanMessage{Content: "What is the capital of France and its history?"},
		},
	}

	ctx := context.Background()
	finalState, err := workflow.Graph.Execute(ctx, initialState, 5) // Max 5 iterations
	if err != nil {
		fmt.Printf("Graph execution error: %v\n", err)
		return
	}
	fmt.Printf("\n--- Workflow Execution Completed ---\nFinal State of the Research Agent:\n%+v\n", finalState)

	s := api.NewServer()
	frontendBuildDir := "../frontend/dist"

	s.SetupFrontend(frontendBuildDir)

	if err := s.Start(port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
