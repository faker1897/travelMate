package chat

import (
	"SuperBizAgent/api/chat/v1"
	"SuperBizAgent/internal/ai/agent/chat_pipeline"
	"SuperBizAgent/internal/travel"
	"SuperBizAgent/utility/mem"
	"context"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

func (c *ControllerV1) Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error) {
	id := req.Id
	msg := req.Question
	sessionMemory := mem.GetSimpleMemory(id)
	sessionMemory.UpdateTravelProfile(msg)
	travelProfile := sessionMemory.GetTravelProfile()
	supplementalContext := buildTravelRuntimeContext(ctx, msg, travelProfile)
	userMessage := &chat_pipeline.UserMessage{
		ID:                   id,
		Query:                msg,
		History:              sessionMemory.GetMessages(),
		TravelProfileSummary: travelProfile.Summary(),
		MissingSlotsSummary:  summarizeMissingSlots(travelProfile),
		TravelStageSummary:   travelProfile.StageSummary(),
		ResponseTemplateHint: travelProfile.ResponseTemplateHint(),
		StructuredOutputHint: travelProfile.StructuredOutputHint(),
		SupplementalContext:  supplementalContext,
	}

	var out *schema.Message

	runner, err := chat_pipeline.BuildChatAgent(ctx)
	if err == nil {
		primaryCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()
		log.Printf("[chat] primary agent started, session=%s, query=%q", id, msg)
		out, err = runner.Invoke(primaryCtx, userMessage)
	}
	if err != nil || out == nil {
		if err != nil {
			log.Printf("[chat] primary agent failed, falling back to direct chat, session=%s, err=%v", id, err)
		}
		fallbackCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		out, err = chat_pipeline.InvokeDirectChat(fallbackCtx, userMessage)
	}
	if err != nil {
		log.Printf("[chat] fallback failed, session=%s, err=%v", id, err)
		return nil, err
	}
	log.Printf("[chat] response completed, session=%s, chars=%d", id, len(out.Content))
	res = &v1.ChatRes{
		Answer: normalizeAssistantText(out.Content),
	}
	sessionMemory.SetMessages(schema.UserMessage(msg))
	sessionMemory.SetMessages(schema.AssistantMessage(res.Answer, nil))

	return res, nil
}

func summarizeMissingSlots(profile travel.Profile) string {
	missing := profile.MissingSlots()
	if len(missing) == 0 {
		return "主要旅行信息已经比较完整，可以继续细化目的地推荐、住宿区域、交通方案和分日行程。"
	}
	return "仍建议补充：" + strings.Join(missing, "、")
}
