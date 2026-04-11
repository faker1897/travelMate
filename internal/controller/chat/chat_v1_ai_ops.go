package chat

import (
	"SuperBizAgent/api/chat/v1"
	"SuperBizAgent/internal/ai/agent/chat_pipeline"
	"SuperBizAgent/internal/ai/agent/plan_execute_replan"
	"SuperBizAgent/internal/travel"
	"SuperBizAgent/utility/mem"
	"context"
	"errors"
	"log"
	"time"

	"github.com/cloudwego/eino/schema"
)

func (c *ControllerV1) AIOps(ctx context.Context, req *v1.AIOpsReq) (res *v1.AIOpsRes, err error) {
	userQuestion := req.Question
	storedQuestion := req.Question
	if userQuestion == "" {
		userQuestion = "请作为旅行顾问开启一次咨询，先简短欢迎我，然后用 4 到 5 个关键问题了解我的目的地、出行时间、游玩天数、预算、同行人和偏好，不要直接给完整行程。"
		storedQuestion = "请开始旅行咨询"
	}
	profileSummary := "尚未收集到明确的旅行需求。"
	missingSummary := "建议先收集目的地、出行时间、游玩天数、预算、同行人和玩法偏好。"
	stageSummary := "当前阶段：先做旅行需求澄清。"
	responseTemplate := "优先欢迎用户，再提出 4 到 5 个关键问题，帮助确认目的地、出行时间、天数、预算、同行人和偏好。"
	structuredOutput := "优先输出澄清问题；当信息足够时，再按 总体判断 / 推荐行程 / 住宿建议 / 交通建议 / 美食与体验 / 避坑提醒 的结构回答。"
	supplementalContext := ""
	if req.Id != "" {
		sessionMemory := mem.GetSimpleMemory(req.Id)
		sessionMemory.UpdateTravelProfile(storedQuestion)
		travelProfile := sessionMemory.GetTravelProfile()
		profileSummary = travelProfile.Summary()
		missingSummary = summarizeMissingSlots(travelProfile)
		stageSummary = travelProfile.StageSummary()
		responseTemplate = travelProfile.ResponseTemplateHint()
		structuredOutput = travelProfile.StructuredOutputHint()
		supplementalContext = buildTravelRuntimeContext(ctx, userQuestion, travelProfile)
	} else {
		inferredProfile := travel.ExtractProfile(storedQuestion)
		supplementalContext = buildTravelRuntimeContext(ctx, userQuestion, inferredProfile)
	}
	query := `
	你是一名旅行顾问入口助手，需要根据用户输入决定下一步咨询方式。

	你的任务：
	1. 优先调用 query_internal_docs 查询本地旅游知识库，了解可能相关的目的地、玩法和旅行信息。
	2. 如果知识库不够支撑回答，可以调用联网搜索工具补充更广泛的信息。
	3. 如果用户询问近期天气、未来几天适不适合去、穿衣建议或天气对行程的影响，优先调用 query_weather_forecast。
	4. 涉及时间判断时，先调用 get_current_time。
	5. 如果用户信息不足，不要直接生成完整 itinerary，而是先提出 4 到 5 个最关键的澄清问题。
	6. 如果用户已经给了较明确的目的地或需求，就先给一版方向性建议，再告诉用户你还可以继续展开成详细行程。
	7. 如果额外补充信息里已经带有实时天气结果，要把天气融入行程、住宿、出行和穿衣建议，而不是只复述天气。
	8. 如果额外补充信息里已经带有预订跳转信息，要把链接整理进“预订建议”部分，并继续完成旅行规划，不要只贴链接。

输出要求：
- 使用中文
- 允许使用 Markdown
- 语气像专业但亲切的旅行顾问
- 不要编造预订结果、实时票价或未确认的营业信息

当前已收集到的旅行需求：
` + profileSummary + `

当前仍建议补充的信息：
` + missingSummary + `

当前建议的对话阶段：
` + stageSummary + `

当前推荐的回答结构：
` + responseTemplate + `

	当前推荐的最终输出版式：
	` + structuredOutput + `

	当前额外补充信息（实时天气 / 预订跳转 / 其他运行时信息；如果为空可忽略）：
	` + supplementalContext + `

	用户当前输入：
	` + userQuestion

	primaryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	log.Printf("[travel_advisor] primary plan agent started, session=%s, question=%q", req.Id, userQuestion)
	resp, detail, err := plan_execute_replan.BuildPlanAgent(primaryCtx, query)
	if err != nil {
		log.Printf("[travel_advisor] primary plan agent failed, falling back to direct chat, session=%s, err=%v", req.Id, err)
		userMessage := &chat_pipeline.UserMessage{
			ID:                   req.Id,
			Query:                userQuestion,
			TravelProfileSummary: profileSummary,
			MissingSlotsSummary:  missingSummary,
			TravelStageSummary:   stageSummary,
			ResponseTemplateHint: responseTemplate,
			StructuredOutputHint: structuredOutput,
			SupplementalContext:  supplementalContext,
		}
		if req.Id != "" {
			sessionMemory := mem.GetSimpleMemory(req.Id)
			userMessage.History = sessionMemory.GetMessages()
		}
		fallbackCtx, fallbackCancel := context.WithTimeout(ctx, 20*time.Second)
		defer fallbackCancel()
		out, fallbackErr := chat_pipeline.InvokeDirectChat(fallbackCtx, userMessage)
		if fallbackErr != nil {
			log.Printf("[travel_advisor] fallback failed, session=%s, err=%v", req.Id, fallbackErr)
			return nil, fallbackErr
		}
		resp = normalizeAssistantText(out.Content)
		detail = nil
	}
	if resp == "" {
		return nil, errors.New("内部错误")
	}
	resp = normalizeAssistantText(resp)
	log.Printf("[travel_advisor] response completed, session=%s, chars=%d", req.Id, len(resp))
	if req.Id != "" {
		sessionMemory := mem.GetSimpleMemory(req.Id)
		sessionMemory.SetMessages(schema.UserMessage(storedQuestion))
		sessionMemory.SetMessages(schema.AssistantMessage(resp, nil))
	}
	res = &v1.AIOpsRes{
		Result: resp,
		Detail: detail,
	}
	return res, nil

}
