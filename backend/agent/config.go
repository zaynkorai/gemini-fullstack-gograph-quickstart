package agent

import (
	"os"
	"strconv"
)

type RunnableConfig struct {
	Configurable map[string]interface{}
}

type Configuration struct {
	QueryGeneratorModel    string
	ReasoningModel         string
	NumberOfInitialQueries int
	MaxResearchLoops       int
}

func NewConfiguration() *Configuration {
	return &Configuration{
		QueryGeneratorModel:    "gemini-2.0-flash",
		ReasoningModel:         "gemini-2.5-flash-preview-04-17",
		NumberOfInitialQueries: 3,
		MaxResearchLoops:       2,
	}
}

func (c *Configuration) FromRunnableConfig(config *RunnableConfig) *Configuration {
	if config == nil {
		config = &RunnableConfig{}
	}

	getString := func(envVar, configKey string, defaultValue string) string {
		if val := os.Getenv(envVar); val != "" {
			return val
		}
		if configurableVal, ok := config.Configurable[configKey].(string); ok && configurableVal != "" {
			return configurableVal
		}
		return defaultValue
	}

	getInt := func(envVar, configKey string, defaultValue int) int {
		if val := os.Getenv(envVar); val != "" {
			if intVal, err := strconv.Atoi(val); err == nil {
				return intVal
			}
		}
		if configurableVal, ok := config.Configurable[configKey].(float64); ok {
			return int(configurableVal)
		}
		return defaultValue
	}

	c.QueryGeneratorModel = getString("QUERY_GENERATOR_MODEL", "query_generator_model", c.QueryGeneratorModel)
	c.ReasoningModel = getString("REASONING_MODEL", "reasoning_model", c.ReasoningModel)
	c.NumberOfInitialQueries = getInt("NUMBER_OF_INITIAL_QUERIES", "number_of_initial_queries", c.NumberOfInitialQueries)
	c.MaxResearchLoops = getInt("MAX_RESEARCH_LOOPS", "max_research_loops", c.MaxResearchLoops)

	return c
}
