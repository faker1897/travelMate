package chat

import (
	"SuperBizAgent/internal/ai/tools"
	"SuperBizAgent/internal/travel"
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

func buildTravelRuntimeContext(ctx context.Context, query string, profile travel.Profile) string {
	sections := make([]string, 0, 2)

	if weatherContext := buildWeatherRuntimeContext(ctx, query, profile); weatherContext != "" {
		sections = append(sections, weatherContext)
	}
	if bookingContext := buildBookingRuntimeContext(query, profile); bookingContext != "" {
		sections = append(sections, bookingContext)
	}

	return strings.Join(sections, "\n\n")
}

func buildWeatherRuntimeContext(ctx context.Context, query string, profile travel.Profile) string {
	if !travel.IsWeatherIntent(query) {
		return ""
	}

	location := strings.TrimSpace(profile.Destination)
	if location == "" {
		location = strings.TrimSpace(profile.EffectiveArrivalCity())
	}
	if location == "" {
		return "## 实时天气补充\n- 用户提到了天气，但当前还没有明确目的地；回答时请先确认具体城市，再把天气影响融入行程建议。"
	}

	weatherCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	log.Printf("[travel_context] weather enrichment started, location=%s", location)
	output, err := tools.QueryWeatherForecast(weatherCtx, location, 3)
	if err != nil {
		log.Printf("[travel_context] weather enrichment failed, location=%s, err=%v", location, err)
		return fmt.Sprintf("## 实时天气补充\n- 原计划查询 %s 的天气，但请求失败；回答时请说明暂未获取到实时天气，并继续提供常规规划建议。", location)
	}
	log.Printf("[travel_context] weather enrichment completed, location=%s, success=%t", location, output.Success)

	var builder strings.Builder
	builder.WriteString("## 实时天气补充\n")
	if !output.Success {
		builder.WriteString(fmt.Sprintf("- 目的地：%s\n", location))
		builder.WriteString(fmt.Sprintf("- 查询结果：%s\n", output.Message))
		builder.WriteString("- 回答要求：说明天气暂未成功获取，同时继续完成旅行规划，不要卡在天气查询上。\n")
		return builder.String()
	}

	locationName := output.ResolvedName
	if locationName == "" {
		locationName = location
	}
	builder.WriteString(fmt.Sprintf("- 目的地：%s\n", locationName))
	if output.CurrentWeather != nil {
		builder.WriteString(fmt.Sprintf("- 当前天气：%s，%.1f°C，体感 %.1f°C，风速 %.1f km/h\n",
			output.CurrentWeather.WeatherDescription,
			output.CurrentWeather.TemperatureC,
			output.CurrentWeather.ApparentTemperatureC,
			output.CurrentWeather.WindSpeedKmh,
		))
	}
	if len(output.DailyForecasts) > 0 {
		builder.WriteString("- 未来几天：\n")
		for i, daily := range output.DailyForecasts {
			if i >= 3 {
				break
			}
			builder.WriteString(fmt.Sprintf("  - %s：%s，%.1f°C 到 %.1f°C，降水概率 %d%%\n",
				daily.Date,
				daily.WeatherDescription,
				daily.TempMinC,
				daily.TempMaxC,
				daily.PrecipitationProbability,
			))
		}
	}
	if output.TravelAdvice != "" {
		builder.WriteString(fmt.Sprintf("- 对行程的影响：%s\n", output.TravelAdvice))
	}
	builder.WriteString("- 回答要求：把天气影响融入行程安排、住宿建议、穿衣建议或室内外活动取舍，不要只单独总结天气。\n")
	return builder.String()
}

func buildBookingRuntimeContext(query string, profile travel.Profile) string {
	if !travel.IsBookingIntent(query) {
		return ""
	}

	suggestion := travel.BuildBookingSuggestion(profile)
	var builder strings.Builder
	builder.WriteString("## 预订补充信息\n")
	builder.WriteString("- 当前模式：仅支持整理条件并跳转到外部平台继续搜索，不支持站内下单、支付或退改签。\n")

	if suggestion.TransportMode != "" {
		builder.WriteString(fmt.Sprintf("- 当前交通方式：%s\n", bookingTransportLabel(suggestion.TransportMode)))
	}
	if suggestion.DepartureCity != "" || suggestion.ArrivalCity != "" {
		builder.WriteString(fmt.Sprintf("- 当前路线：%s -> %s\n",
			fallbackText(suggestion.DepartureCity, "待补充出发地"),
			fallbackText(suggestion.ArrivalCity, "待补充到达地"),
		))
	}
	if suggestion.DepartureDate != "" {
		builder.WriteString(fmt.Sprintf("- 出发日期：%s\n", suggestion.DepartureDate))
	}
	if suggestion.ReturnDate != "" {
		builder.WriteString(fmt.Sprintf("- 返程日期：%s\n", suggestion.ReturnDate))
	}

	if suggestion.Status != "search_ready" {
		builder.WriteString("- 当前状态：信息还不够，回答时应继续追问缺失字段，而不是直接结束在预订话题。\n")
		if len(suggestion.MissingFields) > 0 {
			builder.WriteString("- 仍需补充：")
			builder.WriteString(strings.Join(suggestion.MissingFields, "、"))
			builder.WriteString("\n")
		}
		return builder.String()
	}

	builder.WriteString("- 当前状态：信息足够，可以在回答末尾附上预订跳转入口，同时继续完成行程或交通建议。\n")
	if suggestion.SearchQuery != "" {
		builder.WriteString(fmt.Sprintf("- 主搜索词：%s\n", suggestion.SearchQuery))
	}
	if suggestion.AlternateQuery != "" {
		builder.WriteString(fmt.Sprintf("- 备用搜索词：%s\n", suggestion.AlternateQuery))
	}
	if len(suggestion.Links) > 0 {
		builder.WriteString("- 可直接附上的跳转链接：\n")
		for _, link := range suggestion.Links {
			builder.WriteString(fmt.Sprintf("  - [%s](%s)\n", link.Label, link.URL))
		}
	}
	builder.WriteString("- 回答要求：把这些链接整理到“预订建议”部分，并同时给出交通选择建议，不要只输出链接。\n")
	return builder.String()
}

func bookingTransportLabel(mode string) string {
	switch mode {
	case travel.TransportTrain:
		return "高铁/火车"
	case travel.TransportFlight:
		return "机票/航班"
	case travel.TransportUndecided:
		return "待确认"
	default:
		return "待确认"
	}
}

func fallbackText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
