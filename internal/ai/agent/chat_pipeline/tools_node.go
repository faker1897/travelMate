package chat_pipeline

import (
	"SuperBizAgent/internal/ai/tools"
	"context"

	"github.com/cloudwego/eino/components/tool"
)

func newSearchTool(ctx context.Context) (bt tool.BaseTool, err error) {
	return tools.NewWebSearchTool(ctx)
}
