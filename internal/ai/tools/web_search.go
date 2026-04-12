package tools

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type WebSearchInput struct {
	Query     string `json:"query" jsonschema:"description=Search query used to find travel information on the web"`
	TimeRange string `json:"time_range,omitempty" jsonschema:"description=Optional time range filter. Supported values: d, w, m, y, or empty string"`
}

type WebSearchOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Query   string `json:"query,omitempty"`
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Summary string `json:"summary"`
	} `json:"results,omitempty"`
}

func NewWebSearchTool(ctx context.Context) (tool.BaseTool, error) {
	searchClient, err := duckduckgo.NewSearch(ctx, &duckduckgo.Config{
		Timeout:    60 * time.Second,
		MaxResults: 5,
		Region:     duckduckgo.RegionCN,
	})
	if err != nil {
		return nil, err
	}

	return utils.InferOptionableTool(
		"duckduckgo_text_search",
		"Search the web for travel-related information such as destination references, attraction summaries, transport hints, and general up-to-date travel context. If the network is unstable, this tool may return a graceful failure message instead of throwing an error.",
		func(ctx context.Context, input *WebSearchInput, opts ...tool.Option) (string, error) {
			query := strings.TrimSpace(input.Query)
			if query == "" {
				output := WebSearchOutput{
					Success: false,
					Message: "Search query is empty. Please provide a concrete keyword or destination.",
				}
				return marshalWebSearchOutput(output)
			}

			resp, err := searchClient.TextSearch(ctx, &duckduckgo.TextSearchRequest{
				Query:     query,
				TimeRange: duckduckgo.TimeRange(input.TimeRange),
			})
			if err != nil {
				log.Printf("web search failed for query %q: %v", query, err)
				output := WebSearchOutput{
					Success: false,
					Query:   query,
					Message: "Web search is temporarily unavailable or timed out. Please continue with local travel knowledge and clearly say the online search failed.",
				}
				return marshalWebSearchOutput(output)
			}

			output := WebSearchOutput{
				Success: true,
				Query:   query,
				Message: resp.Message,
			}
			for _, item := range resp.Results {
				if item == nil {
					continue
				}
				output.Results = append(output.Results, struct {
					Title   string `json:"title"`
					URL     string `json:"url"`
					Summary string `json:"summary"`
				}{
					Title:   item.Title,
					URL:     item.URL,
					Summary: item.Summary,
				})
			}
			return marshalWebSearchOutput(output)
		},
	)
}

func marshalWebSearchOutput(output WebSearchOutput) (string, error) {
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
