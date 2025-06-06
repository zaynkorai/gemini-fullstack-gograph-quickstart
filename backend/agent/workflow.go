package agent

type Workflow struct {
	Graph *Graph[*OverallState]
}

func NewWorkflow(config *Configuration, apiKey string) (*Workflow, error) {

	nodes := NewNodes(config, apiKey)

	builder := NewGraph[*OverallState]()

	builder.AddNode("GenerateQuery", nodes.GenerateQueryNode)
	builder.AddNode("WebResearch", nodes.WebResearchNode)
	builder.AddNode("Reflection", nodes.ReflectionNode)
	builder.AddNode("FinalizeAnswer", nodes.FinalizeAnswerNode)

	builder.SetEntryPoint("GenerateQuery")

	builder.AddEdge("WebResearch", "Reflection")

	builder.AddEdge("FinalizeAnswer", GraphEnd)

	builder.AddConditionalEdges(
		"reflection",
		nodes.EvaluateResearchNode,
		map[string]string{
			"web_research":    "web_research",
			"finalize_answer": "finalize_answer",
		},
	)

	compiledGraph := builder.Compile()

	return &Workflow{
		Graph: compiledGraph,
	}, nil
}
