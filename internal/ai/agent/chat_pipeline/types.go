package chat_pipeline

import "github.com/cloudwego/eino/schema"

type UserMessage struct {
	ID                   string            `json:"id"`
	Query                string            `json:"query"`
	History              []*schema.Message `json:"history"`
	TravelProfileSummary string            `json:"travel_profile_summary"`
	MissingSlotsSummary  string            `json:"missing_slots_summary"`
	TravelStageSummary   string            `json:"travel_stage_summary"`
	ResponseTemplateHint string            `json:"response_template_hint"`
	StructuredOutputHint string            `json:"structured_output_hint"`
	SupplementalContext  string            `json:"supplemental_context"`
}
