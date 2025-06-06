package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ChatGoogleGenerativeAI struct {
	Model            string
	Temperature      float64
	MaxRetries       int
	APIKey           string
	StructuredOutput bool
}

func (llm *ChatGoogleGenerativeAI) WithStructuredOutput(outputSchema interface{}) *ChatGoogleGenerativeAI {
	newLLM := *llm
	newLLM.StructuredOutput = true
	return &newLLM
}

func (llm *ChatGoogleGenerativeAI) Invoke(prompt string) (AIMessage, error) {
	if llm.StructuredOutput {
		if strings.Contains(prompt, "query_writer_instructions") {
			return AIMessage{Content: `{"query": [{"Query": "mock query 1", "Rationale": "mock reason 1"}, {"Query": "mock query 2", "Rationale": "mock reason 2"}]}`}, nil
		}
		if strings.Contains(prompt, "reflection_instructions") {
			return AIMessage{Content: `{"is_sufficient": false, "knowledge_gap": "more info needed", "follow_up_queries": ["follow up query 1", "follow up query 2"]}`}, nil
		}
		return AIMessage{Content: `{}`}, nil
	}
	return AIMessage{Content: "This is a mock LLM response to: " + prompt}, nil
}

type GeminiClient struct {
	APIKey string
}

type GeminiGenerateContentConfig struct {
	Tools       []map[string]any `json:"tools"`
	Temperature float64          `json:"temperature"`
}

type GeminiCandidate struct {
	GroundingMetadata GroundingMetadata `json:"grounding_metadata"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	Text       string            `json:"text"`
}

func (c *GeminiClient) GenerateContent(model string, contents string, config GeminiGenerateContentConfig) (GeminiResponse, error) {
	mockGroundingChunks := []GroundingChunk{
		{Web: struct {
			URI   string `json:"uri"`
			Title string `json:"title"`
		}{URI: "https://example.com/doc1", Title: "Document One.pdf"}},
		{Web: struct {
			URI   string `json:"uri"`
			Title string `json:"title"`
		}{URI: "https://example.com/doc2", Title: "Another Doc.html"}},
	}
	mockGroundingSupports := []GroundingSupport{
		{Segment: struct {
			StartIndex int `json:"start_index"`
			EndIndex   int `json:"end_index"`
		}{StartIndex: 10, EndIndex: 20}, GroundingChunkIndices: []int{0}},
		{Segment: struct {
			StartIndex int `json:"start_index"`
			EndIndex   int `json:"end_index"`
		}{StartIndex: 30, EndIndex: 45}, GroundingChunkIndices: []int{1}},
	}

	return GeminiResponse{
		Text: "Mock web research result for: " + contents + " [1] [2]",
		Candidates: []GeminiCandidate{
			{GroundingMetadata: GroundingMetadata{
				GroundingSupports: mockGroundingSupports,
				GroundingChunks:   mockGroundingChunks,
			}},
		},
	}, nil
}

type Nodes struct {
	config            *Configuration
	geminiClient      *GeminiClient
	queryGeneratorLLM *ChatGoogleGenerativeAI
	reasoningLLM      *ChatGoogleGenerativeAI
}

func NewNodes(config *Configuration, apiKey string) *Nodes {
	return &Nodes{
		config:       config,
		geminiClient: &GeminiClient{APIKey: apiKey},
		queryGeneratorLLM: &ChatGoogleGenerativeAI{
			Model:       config.QueryGeneratorModel,
			Temperature: 1.0,
			MaxRetries:  2,
			APIKey:      apiKey,
		},
		reasoningLLM: &ChatGoogleGenerativeAI{
			Model:       config.ReasoningModel,
			Temperature: 1.0,
			MaxRetries:  2,
			APIKey:      apiKey,
		},
	}
}

func (n *Nodes) GenerateQueryNode(ctx context.Context, state *OverallState) (*OverallState, string, error) {
	if state.InitialSearchQueryCount == 0 {
		state.InitialSearchQueryCount = n.config.NumberOfInitialQueries
	}

	structured_llm := n.queryGeneratorLLM.WithStructuredOutput(SearchQueryList{})

	current_date := GetCurrentDate()
	researchTopic := GetResearchTopic(state.Messages)

	formatted_prompt := fmt.Sprintf(QueryWriterInstructions,
		state.InitialSearchQueryCount, researchTopic, current_date)

	result, err := structured_llm.Invoke(formatted_prompt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate query: %w", err)
	}

	var sqList SearchQueryList
	err = json.Unmarshal([]byte(result.Content), &sqList)
	if err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal SearchQueryList: %w", err)
	}

	state.Messages = append(state.Messages, AIMessage{Content: result.Content})
	state.InitialSearchQueryCount = n.config.NumberOfInitialQueries
	state.SearchQueries = sqList.Query
	return state, "web_research", nil
}

func (n *Nodes) WebResearchNode(ctx context.Context, state *OverallState) (*OverallState, string, error) {
	var allSourcesGathered []SourceSegment
	var allWebResearchResult []string

	if state.SearchQueries == nil {
		state.SearchQueries = []Query{}
	}

	queriesToProcess := state.SearchQueries
	state.SearchQueries = []Query{} // Clear for tracking actual ran queries in this loop

	for idx, query := range queriesToProcess {
		formatted_prompt := fmt.Sprintf(WebSearcherInstructions, query.Query, GetCurrentDate())

		response, err := n.geminiClient.GenerateContent(
			n.config.QueryGeneratorModel,
			formatted_prompt,
			GeminiGenerateContentConfig{
				Tools:       []map[string]any{{"Google Search": map[string]any{}}},
				Temperature: 0,
			},
		)
		if err != nil {
			return nil, "", fmt.Errorf("error during web search for query '%s': %w", query.Query, err)
		}

		resolved_urls := ResolveURLs(response.Candidates[0].GroundingMetadata.GroundingChunks, idx)
		citations := GetCitations(&LLMResponse{
			Candidates: []struct {
				GroundingMetadata GroundingMetadata `json:"grounding_metadata"`
			}{
				{GroundingMetadata: response.Candidates[0].GroundingMetadata},
			},
			Text: response.Text,
		}, resolved_urls)
		modified_text := InsertCitationMarkers(response.Text, citations)

		for _, citation := range citations {
			if segments, ok := citation["segments"].([]map[string]interface{}); ok {
				for _, segment := range segments {
					allSourcesGathered = append(allSourcesGathered, SourceSegment{
						Value:    segment["value"].(string),
						ShortURL: segment["short_url"].(string),
						LinkID:   fmt.Sprintf("%d", idx),
					})
				}
			}
		}
		allWebResearchResult = append(allWebResearchResult, modified_text)
		state.SearchQueries = append(state.SearchQueries, query) // Tracking all queries that were actually executed
	}

	state.SourcesGathered = append(state.SourcesGathered, allSourcesGathered...)
	state.WebResearchResults = append(state.WebResearchResults, allWebResearchResult...)

	return state, "reflection", nil
}

func (n *Nodes) ReflectionNode(ctx context.Context, state *OverallState) (*OverallState, string, error) {
	state.ResearchLoopCount++

	llm := n.reasoningLLM
	if state.ReasoningModel != "" {
		llm.Model = state.ReasoningModel // Directly set the model if specified in state
	} else {
		llm.Model = n.config.ReasoningModel // Otherwise, use the default from config
	}

	formatted_prompt := fmt.Sprintf(ReflectionInstructions,
		GetResearchTopic(state.Messages), GetCurrentDate(), strings.Join(state.WebResearchResults, "\n\n---\n\n"))

	result, err := llm.WithStructuredOutput(Reflection{}).Invoke(formatted_prompt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to perform reflection: %w", err)
	}

	var reflectionResult Reflection
	err = json.Unmarshal([]byte(result.Content), &reflectionResult)
	if err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal Reflection: %w", err)
	}

	state.IsSufficient = reflectionResult.IsSufficient
	state.KnowledgeGap = reflectionResult.KnowledgeGap
	state.FollowUpQueries = reflectionResult.FollowUpQueries
	state.ResearchLoopCount = state.ResearchLoopCount   // Already incremented
	state.NumberOfRanQueries = len(state.SearchQueries) // Refers to the queries ran in WebResearchNode

	return state, "evaluate_research", nil
}

func (n *Nodes) EvaluateResearchNode(ctx context.Context, state *OverallState) (*OverallState, string, error) {
	max_research_loops := n.config.MaxResearchLoops
	if state.MaxResearchLoops != 0 {
		max_research_loops = state.MaxResearchLoops
	}

	if state.IsSufficient || state.ResearchLoopCount >= max_research_loops {
		return state, "finalize_answer", nil
	} else {
		state.SearchQueries = []Query{}
		for _, q := range state.FollowUpQueries {
			state.SearchQueries = append(state.SearchQueries, Query{Query: q})
		}
		state.FollowUpQueries = []string{} // Clear follow up queries once processed
		return state, "web_research", nil
	}
}

func (n *Nodes) FinalizeAnswerNode(ctx context.Context, state *OverallState) (*OverallState, string, error) {
	llm := n.reasoningLLM
	if state.ReasoningModel != "" {
		llm.Model = state.ReasoningModel // Directly set the model if specified in state
	} else {
		llm.Model = n.config.ReasoningModel // Otherwise, use the default from config
	}
	llm.Temperature = 0

	formatted_prompt := fmt.Sprintf(AnswerInstructions,
		GetResearchTopic(state.Messages), strings.Join(state.WebResearchResults, "\n---\n\n"), GetCurrentDate())

	result, err := llm.Invoke(formatted_prompt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to finalize answer: %w", err)
	}

	uniqueSourcesMap := make(map[string]SourceSegment)
	for _, source := range state.SourcesGathered {
		if strings.Contains(result.Content, source.ShortURL) {
			result.Content = strings.ReplaceAll(result.Content, source.ShortURL, source.Value)
			uniqueSourcesMap[source.Value] = source
		}
	}

	var uniqueSources []SourceSegment
	for _, source := range uniqueSourcesMap {
		uniqueSources = append(uniqueSources, source)
	}

	state.Messages = []Message{AIMessage{Content: result.Content}}
	state.SourcesGathered = uniqueSources

	return state, "__END__", nil
}
