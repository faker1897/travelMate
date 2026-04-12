package main

import (
	"SuperBizAgent/internal/ai/agent/plan_execute_replan"
	"context"
	"fmt"
)

func main() {
	ctx := context.Background()
	query := `
"你是一名旅行顾问入口助手。"
"请优先查询本地旅游知识库，需要时补充联网搜索。"
"如果用户信息不足，就先询问目的地、出行时间、天数、预算、同行人和偏好。"
"如果信息足够，就先给出方向性建议，再说明你还能继续细化成分日行程。"`
	resp, detail, err := plan_execute_replan.BuildPlanAgent(ctx, query)
	if err != nil {
		panic(err)
	}
	fmt.Println("----- Final Response -----")
	fmt.Println(resp)
	fmt.Println("----- Final detail -----")
	fmt.Println(detail)
}
