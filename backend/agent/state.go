package agent

type Message interface {
	GetContent() string
	Type() string
}

type HumanMessage struct {
	Content string
}

func (m HumanMessage) GetContent() string {
	return m.Content
}

func (m HumanMessage) Type() string {
	return "human"
}

type AIMessage struct {
	Content string
}

func (m AIMessage) GetContent() string {
	return m.Content
}

func (m AIMessage) Type() string {
	return "ai"
}

type Query struct {
	Query     string `json:"query"`
	Rationale string `json:"rationale"`
}

type SourceSegment struct {
	Value    string `json:"value"`
	ShortURL string `json:"short_url"`
	LinkID   string `json:"link_id"`
}

type OverallState struct {
	Messages                []Message
	SearchQueries           []Query
	WebResearchResults      []string
	SourcesGathered         []SourceSegment
	InitialSearchQueryCount int
	MaxResearchLoops        int
	ResearchLoopCount       int
	ReasoningModel          string

	IsSufficient       bool
	KnowledgeGap       string
	FollowUpQueries    []string
	NumberOfRanQueries int
}

type SearchQueryList struct {
	Query []Query `json:"query"`
}

type Reflection struct {
	IsSufficient    bool     `json:"is_sufficient"`
	KnowledgeGap    string   `json:"knowledge_gap"`
	FollowUpQueries []string `json:"follow_up_queries"`
}
