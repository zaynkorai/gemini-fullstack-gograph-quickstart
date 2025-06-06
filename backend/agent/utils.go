package agent

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type GroundingChunk struct {
	Web struct {
		URI   string `json:"uri"`
		Title string `json:"title"`
	} `json:"web"`
}

type GroundingSupport struct {
	Segment struct {
		StartIndex int `json:"start_index"`
		EndIndex   int `json:"end_index"`
	} `json:"segment"`
	GroundingChunkIndices []int `json:"grounding_chunk_indices"`
}

type GroundingMetadata struct {
	GroundingSupports []GroundingSupport `json:"grounding_supports"`
	GroundingChunks   []GroundingChunk   `json:"grounding_chunks"`
}

type LLMResponse struct {
	Candidates []struct {
		GroundingMetadata GroundingMetadata `json:"grounding_metadata"`
	} `json:"candidates"`
	Text string `json:"text"`
}

func GetResearchTopic(messages []Message) string {
	if len(messages) == 1 {
		return messages[len(messages)-1].GetContent()
	}

	var researchTopic strings.Builder
	for _, message := range messages {
		if _, ok := message.(HumanMessage); ok {
			researchTopic.WriteString(fmt.Sprintf("User: %s\n", message.GetContent()))
		} else if _, ok := message.(AIMessage); ok {
			researchTopic.WriteString(fmt.Sprintf("Assistant: %s\n", message.GetContent()))
		}
	}
	return researchTopic.String()
}

func ResolveURLs(urlsToResolve []GroundingChunk, id int) map[string]string {
	prefix := "https://vertexaisearch.cloud.google.com/id/"
	resolvedMap := make(map[string]string)

	for idx, chunk := range urlsToResolve {
		url := chunk.Web.URI
		if _, exists := resolvedMap[url]; !exists {
			resolvedMap[url] = fmt.Sprintf("%s%d-%d", prefix, id, idx)
		}
	}
	return resolvedMap
}

func InsertCitationMarkers(text string, citationsList []map[string]interface{}) string {
	sort.Slice(citationsList, func(i, j int) bool {
		endI := citationsList[i]["end_index"].(int)
		endJ := citationsList[j]["end_index"].(int)
		if endI != endJ {
			return endI > endJ
		}
		startI := citationsList[i]["start_index"].(int)
		startJ := citationsList[j]["start_index"].(int)
		return startI > startJ
	})

	modifiedText := text
	for _, citationInfo := range citationsList {
		endIdx := citationInfo["end_index"].(int)
		markerToInsert := ""
		segments, ok := citationInfo["segments"].([]interface{})
		if !ok {
			continue
		}
		for _, seg := range segments {
			segmentMap, ok := seg.(map[string]interface{})
			if !ok {
				continue
			}
			label, _ := segmentMap["label"].(string)
			shortURL, _ := segmentMap["short_url"].(string)
			markerToInsert += fmt.Sprintf(" [%s](%s)", label, shortURL)
		}

		modifiedText = modifiedText[:endIdx] + markerToInsert + modifiedText[endIdx:]
	}

	return modifiedText
}

func GetCitations(response *LLMResponse, resolvedURLsMap map[string]string) []map[string]interface{} {
	citations := []map[string]interface{}{}

	if response == nil || len(response.Candidates) == 0 {
		return citations
	}

	candidate := response.Candidates[0]
	if len(candidate.GroundingMetadata.GroundingSupports) == 0 {
		return citations
	}

	for _, support := range candidate.GroundingMetadata.GroundingSupports {
		citation := make(map[string]interface{})

		if support.Segment.EndIndex == 0 && support.Segment.StartIndex == 0 && support.Segment.EndIndex <= support.Segment.StartIndex {
			continue
		}

		citation["start_index"] = support.Segment.StartIndex
		citation["end_index"] = support.Segment.EndIndex

		segmentsData := []map[string]interface{}{}
		for _, ind := range support.GroundingChunkIndices {
			if ind >= 0 && ind < len(candidate.GroundingMetadata.GroundingChunks) {
				chunk := candidate.GroundingMetadata.GroundingChunks[ind]
				resolvedURL, ok := resolvedURLsMap[chunk.Web.URI]
				if !ok {
					continue
				}

				titleParts := strings.Split(chunk.Web.Title, ".")
				label := chunk.Web.Title
				if len(titleParts) > 1 {
					label = titleParts[0]
				}

				segmentsData = append(segmentsData, map[string]interface{}{
					"label":     label,
					"short_url": resolvedURL,
					"value":     chunk.Web.URI,
				})
			}
		}
		citation["segments"] = segmentsData
		citations = append(citations, citation)
	}
	return citations
}

func GetCurrentDate() string {
	return time.Now().Format("January 2, 2006")
}
