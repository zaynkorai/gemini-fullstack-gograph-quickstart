package api

type UserQuery struct {
	Query string `json:"query"`
}

type SearchQueryList struct {
	Query     []string `json:"query" binding:"required"`
	Rationale string   `json:"rationale" binding:"required"`
}

type Reflection struct {
	IsSufficient    bool     `json:"is_sufficient" binding:"required"`
	KnowledgeGap    string   `json:"knowledge_gap" binding:"required"`
	FollowUpQueries []string `json:"follow_up_queries" binding:"required"`
}
