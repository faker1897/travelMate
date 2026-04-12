package plan_execute_replan

import (
	"SuperBizAgent/internal/ai/models"
	"SuperBizAgent/internal/ai/tools"
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func NewExecutor(ctx context.Context) (adk.Agent, error) {
	searchTool, err := tools.NewWebSearchTool(ctx)
	if err != nil {
		return nil, err
	}
	toolList := []tool.BaseTool{searchTool}
	toolList = append(toolList, tools.NewQueryInternalDocsTool())
	toolList = append(toolList, tools.NewGetCurrentTimeTool())
	toolList = append(toolList, tools.NewWeatherForecastTool())
	execModel, err := models.OpenAIForDeepSeekV3Quick(ctx)
	if err != nil {
		return nil, err
	}
	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: execModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolList,
			},
		},
		MaxIterations: 999999,
	})
}
