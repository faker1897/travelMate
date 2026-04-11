package main

import (
	"SuperBizAgent/internal/ai/agent/chat_pipeline"
	"SuperBizAgent/internal/travel"
	"SuperBizAgent/utility/mem"
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	id := "111"
	sessionMemory := mem.GetSimpleMemory(id)
	sessionMemory.UpdateTravelProfile("你好")
	travelProfile := sessionMemory.GetTravelProfile()
	userMessage := &chat_pipeline.UserMessage{
		ID:                   id,
		Query:                "你好",
		History:              sessionMemory.GetMessages(),
		TravelProfileSummary: travelProfile.Summary(),
		MissingSlotsSummary:  summarizeMissingSlotsForCmd(travelProfile),
		TravelStageSummary:   travelProfile.StageSummary(),
		ResponseTemplateHint: travelProfile.ResponseTemplateHint(),
		StructuredOutputHint: travelProfile.StructuredOutputHint(),
	}
	runner, err := chat_pipeline.BuildChatAgent(ctx)
	if err != nil {
		panic(err)
	}
	// 第一次对话
	out, err := runner.Invoke(ctx, userMessage)
	if err != nil {
		panic(err)
	}
	answer := out.Content
	fmt.Println("Q: 你好")
	fmt.Println("A:", answer)
	sessionMemory.SetMessages(schema.UserMessage("你好"))
	sessionMemory.SetMessages(schema.AssistantMessage(out.Content, nil))
	// 第二次对话
	sessionMemory.UpdateTravelProfile("现在是几点")
	travelProfile = sessionMemory.GetTravelProfile()
	userMessage = &chat_pipeline.UserMessage{
		ID:                   id,
		Query:                "现在是几点",
		History:              sessionMemory.GetMessages(),
		TravelProfileSummary: travelProfile.Summary(),
		MissingSlotsSummary:  summarizeMissingSlotsForCmd(travelProfile),
		TravelStageSummary:   travelProfile.StageSummary(),
		ResponseTemplateHint: travelProfile.ResponseTemplateHint(),
		StructuredOutputHint: travelProfile.StructuredOutputHint(),
	}
	out, err = runner.Invoke(ctx, userMessage)
	if err != nil {
		panic(err)
	}
	answer = out.Content
	fmt.Println("----------------")
	fmt.Println("Q: 现在是几点")
	fmt.Println("A:", answer)
}

func summarizeMissingSlotsForCmd(profile travel.Profile) string {
	missing := profile.MissingSlots()
	if len(missing) == 0 {
		return "主要旅行信息已经比较完整，可以继续细化行程。"
	}
	return "仍建议补充：" + strings.Join(missing, "、")
}
