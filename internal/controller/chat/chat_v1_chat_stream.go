package chat

import (
	"SuperBizAgent/api/chat/v1"
	"SuperBizAgent/internal/ai/agent/chat_pipeline"
	"SuperBizAgent/utility/mem"
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

func (c *ControllerV1) ChatStream(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error) {
	id := req.Id
	msg := req.Question
	sessionMemory := mem.GetSimpleMemory(id)
	sessionMemory.UpdateTravelProfile(msg)
	travelProfile := sessionMemory.GetTravelProfile()
	supplementalContext := buildTravelRuntimeContext(ctx, msg, travelProfile)

	ctx = context.WithValue(ctx, "client_id", req.Id)
	client, err := c.service.Create(ctx, g.RequestFromCtx(ctx))
	if err != nil {
		return nil, err
	}

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

	var fullResponse strings.Builder

	runner, err := chat_pipeline.BuildChatAgent(ctx)
	if err == nil {
		primaryCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()
		log.Printf("[chat_stream] primary agent started, session=%s, query=%q", id, msg)
		var sr *schema.StreamReader[*schema.Message]
		sr, err = runner.Stream(primaryCtx, userMessage)
		if err == nil {
			defer sr.Close()

			defer func() {
				completeResponse := fullResponse.String()
				if completeResponse != "" {
					sessionMemory.SetMessages(schema.UserMessage(msg))
					sessionMemory.SetMessages(schema.AssistantMessage(completeResponse, nil))
				}
			}()

			for {
				chunk, recvErr := sr.Recv()
				if errors.Is(recvErr, io.EOF) {
					log.Printf("[chat_stream] primary stream completed, session=%s, chars=%d", id, len(fullResponse.String()))
					client.SendToClient("done", "Stream completed")
					return &v1.ChatStreamRes{}, nil
				}
				if recvErr != nil {
					log.Printf("[chat_stream] primary stream recv failed, session=%s, err=%v", id, recvErr)
					client.SendToClient("error", recvErr.Error())
					return &v1.ChatStreamRes{}, nil
				}
				normalizedChunk := normalizeAssistantText(chunk.Content)
				fullResponse.WriteString(normalizedChunk)
				client.SendToClient("message", normalizedChunk)
			}
		}
	}

	if err != nil {
		log.Printf("[chat_stream] primary agent failed, falling back to direct chat, session=%s, err=%v", id, err)
		fallbackCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		out, fallbackErr := chat_pipeline.InvokeDirectChat(fallbackCtx, userMessage)
		if fallbackErr != nil {
			log.Printf("[chat_stream] fallback failed, session=%s, err=%v", id, fallbackErr)
			client.SendToClient("error", fallbackErr.Error())
			return nil, fallbackErr
		}
		content := normalizeAssistantText(out.Content)
		fullResponse.WriteString(content)
		sessionMemory.SetMessages(schema.UserMessage(msg))
		sessionMemory.SetMessages(schema.AssistantMessage(content, nil))
		log.Printf("[chat_stream] fallback completed, session=%s, chars=%d", id, len(content))
		client.SendToClient("message", content)
		client.SendToClient("done", "Stream completed")
		return &v1.ChatStreamRes{}, nil
	}

	return &v1.ChatStreamRes{}, nil
}
