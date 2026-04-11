package travel

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	TransportUndecided = "undecided"
	TransportFlight    = "flight"
	TransportTrain     = "train"
)

type BookingLink struct {
	Label string
	URL   string
}

type BookingSuggestion struct {
	Status         string
	TransportMode  string
	DepartureCity  string
	ArrivalCity    string
	DepartureDate  string
	ReturnDate     string
	MissingFields  []string
	SearchQuery    string
	AlternateQuery string
	Links          []BookingLink
	Disclaimer     string
}

func (p Profile) BookingStatus() string {
	if p.TransportMode == "" && p.DepartureCity == "" && p.ArrivalCity == "" && p.DepartureDate == "" {
		return ""
	}
	if len(p.MissingBookingSlots()) > 0 {
		return "need_booking_fields"
	}
	return "search_ready"
}

func (p Profile) MissingBookingSlots() []string {
	missing := make([]string, 0, 4)
	if strings.TrimSpace(p.TransportMode) == "" || p.TransportMode == TransportUndecided {
		missing = append(missing, "交通方式")
	}
	if strings.TrimSpace(p.DepartureCity) == "" {
		missing = append(missing, "出发地")
	}
	if strings.TrimSpace(p.EffectiveArrivalCity()) == "" {
		missing = append(missing, "到达地")
	}
	return missing
}

func (p Profile) BookingSummary() string {
	if p.BookingStatus() == "" {
		return "尚未收集到明确的交通预订信息。"
	}

	parts := make([]string, 0, 5)
	if mode := transportModeLabel(p.TransportMode); mode != "" {
		parts = append(parts, fmt.Sprintf("交通方式：%s", mode))
	}
	if p.DepartureCity != "" {
		parts = append(parts, fmt.Sprintf("出发地：%s", p.DepartureCity))
	}
	if arrival := p.EffectiveArrivalCity(); arrival != "" {
		parts = append(parts, fmt.Sprintf("到达地：%s", arrival))
	}
	if departureDate := p.EffectiveDepartureDate(); departureDate != "" {
		parts = append(parts, fmt.Sprintf("出发日期：%s", departureDate))
	}
	if p.ReturnDate != "" {
		parts = append(parts, fmt.Sprintf("返程日期：%s", p.ReturnDate))
	}
	return strings.Join(parts, "\n")
}

func BuildBookingSuggestion(p Profile) BookingSuggestion {
	arrival := p.EffectiveArrivalCity()
	departureDate := p.EffectiveDepartureDate()
	status := p.BookingStatus()

	suggestion := BookingSuggestion{
		Status:        status,
		TransportMode: p.TransportMode,
		DepartureCity: p.DepartureCity,
		ArrivalCity:   arrival,
		DepartureDate: departureDate,
		ReturnDate:    p.ReturnDate,
		MissingFields: p.MissingBookingSlots(),
		Disclaimer:    "当前仅支持跳转到外部平台继续搜索或预订，不提供站内下单、支付、退改签处理；价格和余票请以跳转后的平台实时结果为准。",
	}
	if status != "search_ready" {
		return suggestion
	}

	switch p.TransportMode {
	case TransportTrain:
		suggestion.SearchQuery = fmt.Sprintf("%s %s到%s 高铁", departureDate, p.DepartureCity, arrival)
		suggestion.AlternateQuery = fmt.Sprintf("12306 %s到%s %s", p.DepartureCity, arrival, departureDate)
		suggestion.Links = []BookingLink{
			{Label: "搜索火车", URL: baiduSearchLink(suggestion.SearchQuery)},
			{Label: "搜索 12306 入口", URL: baiduSearchLink(suggestion.AlternateQuery)},
		}
	case TransportFlight:
		suggestion.SearchQuery = fmt.Sprintf("%s %s到%s 机票", departureDate, p.DepartureCity, arrival)
		suggestion.AlternateQuery = fmt.Sprintf("%s %s到%s 航班", departureDate, p.DepartureCity, arrival)
		suggestion.Links = []BookingLink{
			{Label: "搜索机票", URL: baiduSearchLink(suggestion.SearchQuery)},
			{Label: "搜索航班", URL: baiduSearchLink(suggestion.AlternateQuery)},
		}
	}
	return suggestion
}

func FormatBookingReply(p Profile) string {
	suggestion := BuildBookingSuggestion(p)
	var builder strings.Builder

	builder.WriteString("## 预订助手\n")
	builder.WriteString("- 我可以先帮你整理预订条件，再跳转到外部平台继续搜索。\n")

	if mode := transportModeLabel(suggestion.TransportMode); mode != "" {
		builder.WriteString(fmt.Sprintf("- 当前交通方式：%s\n", mode))
	}
	if suggestion.DepartureCity != "" || suggestion.ArrivalCity != "" {
		builder.WriteString(fmt.Sprintf("- 当前行程：%s -> %s\n", emptyFallback(suggestion.DepartureCity, "待补充出发地"), emptyFallback(suggestion.ArrivalCity, "待补充到达地")))
	}
	if suggestion.DepartureDate != "" {
		builder.WriteString(fmt.Sprintf("- 出发日期：%s\n", suggestion.DepartureDate))
	} else {
		builder.WriteString("- 出发日期：暂未提供，当前会先给你通用搜索入口\n")
	}
	if suggestion.ReturnDate != "" {
		builder.WriteString(fmt.Sprintf("- 返程日期：%s\n", suggestion.ReturnDate))
	}

	if suggestion.Status != "search_ready" {
		builder.WriteString("\n## 还需要补充的信息\n")
		for _, field := range suggestion.MissingFields {
			builder.WriteString(fmt.Sprintf("- %s\n", field))
		}
		builder.WriteString("\n## 你可以直接这样回复我\n")
		builder.WriteString("- 出发地：\n")
		builder.WriteString("- 到达地：\n")
		builder.WriteString("- 出发日期（可选，但补充后结果更准）：\n")
		builder.WriteString("- 交通方式：高铁 / 机票\n")
		builder.WriteString("\n## 说明\n")
		builder.WriteString("- " + suggestion.Disclaimer + "\n")
		return builder.String()
	}

	builder.WriteString("\n## 跳转预订\n")
	for _, link := range suggestion.Links {
		builder.WriteString(fmt.Sprintf("- [%s](%s)\n", link.Label, link.URL))
	}
	if suggestion.SearchQuery != "" {
		builder.WriteString(fmt.Sprintf("- 搜索条件：`%s`\n", suggestion.SearchQuery))
	}
	if suggestion.AlternateQuery != "" {
		builder.WriteString(fmt.Sprintf("- 备用搜索词：`%s`\n", suggestion.AlternateQuery))
	}
	builder.WriteString("\n## 说明\n")
	builder.WriteString("- " + suggestion.Disclaimer + "\n")
	return builder.String()
}

func (p Profile) EffectiveArrivalCity() string {
	if p.ArrivalCity != "" {
		return p.ArrivalCity
	}
	return p.Destination
}

func (p Profile) EffectiveDepartureDate() string {
	if p.DepartureDate != "" {
		return p.DepartureDate
	}
	return p.TravelDate
}

func transportModeLabel(mode string) string {
	switch mode {
	case TransportTrain:
		return "高铁/火车"
	case TransportFlight:
		return "机票/航班"
	case TransportUndecided:
		return "待确认"
	default:
		return ""
	}
}

func baiduSearchLink(query string) string {
	return "https://www.baidu.com/s?wd=" + url.QueryEscape(query)
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
