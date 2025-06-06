package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/fatih/color"
)

const GraphEnd = "__END__"

type GraphNodeFunc[S any] func(ctx context.Context, state S) (S, string, error)

type EdgeConfig[S any] struct {
	IsConditional  bool
	ToNode         string
	RouterFunc     GraphNodeFunc[S]
	ConditionalMap map[string]string
}

type Graph[S any] struct {
	nodes      map[string]GraphNodeFunc[S]
	edges      map[string]EdgeConfig[S]
	entryPoint string
}

// NewGraph now takes a type parameter
func NewGraph[S any]() *Graph[S] {
	return &Graph[S]{
		nodes: make(map[string]GraphNodeFunc[S]),
		edges: make(map[string]EdgeConfig[S]),
	}
}

// AddNode now takes a generic GraphNodeFunc
func (g *Graph[S]) AddNode(name string, nodeFunc GraphNodeFunc[S]) {
	g.nodes[name] = nodeFunc
}

func (g *Graph[S]) SetEntryPoint(name string) {
	g.entryPoint = name
}

func (g *Graph[S]) SetFinishPoint(name string) {
	g.AddEdge(name, GraphEnd)
}

func (g *Graph[S]) AddEdge(fromNode, toNode string) {
	g.edges[fromNode] = EdgeConfig[S]{
		IsConditional: false,
		ToNode:        toNode,
	}
}

func (g *Graph[S]) AddConditionalEdges(fromNode string, routerFunc GraphNodeFunc[S], conditionalMap map[string]string) {
	g.edges[fromNode] = EdgeConfig[S]{
		IsConditional:  true,
		RouterFunc:     routerFunc,
		ConditionalMap: conditionalMap,
	}
}

func (g *Graph[S]) Compile() *Graph[S] {
	return g
}

// Execute now takes and returns the generic state type S
func (g *Graph[S]) Execute(ctx context.Context, initialState S, maxIterations int) (S, error) {
	currentState := initialState
	currentNodeName := g.entryPoint

	if _, ok := g.nodes[currentNodeName]; !ok {
		return currentState, fmt.Errorf("entry point node '%s' not found", currentNodeName)
	}

	fmt.Printf("\n--- Starting Workflow Execution ---\nInitial State: %+v\n\n", currentState)

	for i := 0; i < maxIterations; i++ {
		if currentNodeName == GraphEnd {
			fmt.Println("Workflow reached END. Terminating.")
			break
		}

		fmt.Printf("Executing node: %s\n", currentNodeName)

		nodeFunc, ok := g.nodes[currentNodeName]
		if !ok {
			return currentState, fmt.Errorf("node '%s' not found in graph definition", currentNodeName)
		}

		// Node function now directly works with the generic state type S
		updatedState, _, err := nodeFunc(ctx, currentState)
		if err != nil {
			return currentState, fmt.Errorf("error executing node '%s': %w", currentNodeName, err)
		}
		currentState = updatedState

		fmt.Println(color.CyanString("Finished running: %s", currentNodeName))

		edgeConfig, edgeExists := g.edges[currentNodeName]
		if !edgeExists {
			fmt.Printf("Node '%s' has no outgoing edges. Implicitly ending path.\n", currentNodeName)
			currentNodeName = GraphEnd
			continue
		}

		var routingDecision string
		if edgeConfig.IsConditional {
			// Router function also directly works with the generic state type S
			_, decisionFromRouterFunc, routerErr := edgeConfig.RouterFunc(ctx, currentState)
			if routerErr != nil {
				return currentState, fmt.Errorf("error executing router function for node '%s': %w", currentNodeName, routerErr)
			}
			routingDecision = decisionFromRouterFunc

			fmt.Printf("Node '%s' is conditional. Router function decided: '%s'\n", currentNodeName, routingDecision)

			nextNode, ok := edgeConfig.ConditionalMap[routingDecision]
			if !ok {
				return currentState, fmt.Errorf("conditional edge from '%s' has no mapping for decision '%s'", currentNodeName, routingDecision)
			}
			currentNodeName = nextNode
		} else {
			currentNodeName = edgeConfig.ToNode
		}

		fmt.Printf("Transitioning to node: %s\n\n", currentNodeName)

		if i == maxIterations-1 && currentNodeName != GraphEnd {
			log.Printf("Warning: Workflow reached max iterations (%d) without reaching END. Terminating to prevent infinite loop.\n", maxIterations)
			break
		}
	}

	fmt.Printf("\n--- Workflow Execution Finished ---\nFinal State: %+v\n", currentState)
	return currentState, nil
}
