package tools

import (
	"SuperBizAgent/internal/ai/retriever"
	"context"
	"encoding/json"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type QueryInternalDocsInput struct {
	Query string `json:"query" jsonschema:"description=The query string to search in the local travel knowledge base for destinations, attractions, transport, food, and itinerary hints"`
}

func NewQueryInternalDocsTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"query_internal_docs",
		"Use this tool to search the local travel knowledge base. It performs RAG retrieval over destination notes and travel guides, and is useful for finding city overviews, attractions, food suggestions, transport advice, and itinerary inspiration stored in uploaded documents.",
		func(ctx context.Context, input *QueryInternalDocsInput, opts ...tool.Option) (output string, err error) {
			rr, err := retriever.NewMilvusRetriever(ctx)
			if err != nil {
				log.Fatal(err)
			}
			resp, err := rr.Retrieve(ctx, input.Query)
			if err != nil {
				log.Fatal(err)
			}
			respBytes, _ := json.Marshal(resp)
			output = string(respBytes)
			return output, nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}
