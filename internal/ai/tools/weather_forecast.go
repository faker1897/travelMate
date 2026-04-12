package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	openMeteoGeocodingURL = "https://geocoding-api.open-meteo.com/v1/search"
	openMeteoForecastURL  = "https://api.open-meteo.com/v1/forecast"
)

type WeatherForecastInput struct {
	Location     string `json:"location" jsonschema:"description=Destination or city name, for example Hangzhou, Beijing, Chengdu, or 西安"`
	ForecastDays int    `json:"forecast_days,omitempty" jsonschema:"description=Number of forecast days to retrieve. Recommended range is 1 to 7"`
}

type WeatherForecastOutput struct {
	Success        bool                   `json:"success"`
	Message        string                 `json:"message"`
	Location       string                 `json:"location,omitempty"`
	ResolvedName   string                 `json:"resolved_name,omitempty"`
	Country        string                 `json:"country,omitempty"`
	Timezone       string                 `json:"timezone,omitempty"`
	CurrentWeather *CurrentWeatherSummary `json:"current_weather,omitempty"`
	DailyForecasts []DailyForecastSummary `json:"daily_forecasts,omitempty"`
	TravelAdvice   string                 `json:"travel_advice,omitempty"`
}

func QueryWeatherForecast(ctx context.Context, location string, forecastDays int) (WeatherForecastOutput, error) {
	location = strings.TrimSpace(location)
	if location == "" {
		return WeatherForecastOutput{
			Success: false,
			Message: "Location is required. Please provide a city or destination name.",
		}, nil
	}

	if forecastDays <= 0 {
		forecastDays = 3
	}
	if forecastDays > 7 {
		forecastDays = 7
	}

	geoResult, err := geocodeLocation(ctx, location)
	if err != nil {
		log.Printf("weather geocoding failed for %q: %v", location, err)
		return WeatherForecastOutput{
			Success:  false,
			Location: location,
			Message:  "Weather service is temporarily unavailable during location lookup. Please continue with local travel knowledge and mention weather could not be retrieved.",
		}, nil
	}
	if geoResult == nil {
		return WeatherForecastOutput{
			Success:  false,
			Location: location,
			Message:  "No matching location was found. Please provide a more specific city or destination name.",
		}, nil
	}

	forecast, err := fetchWeatherForecast(ctx, geoResult.Latitude, geoResult.Longitude, forecastDays)
	if err != nil {
		log.Printf("weather forecast failed for %q: %v", location, err)
		return WeatherForecastOutput{
			Success:      false,
			Location:     location,
			ResolvedName: geoResult.Name,
			Country:      geoResult.Country,
			Message:      "Weather service is temporarily unavailable during forecast lookup. Please continue with local travel knowledge and mention weather could not be retrieved.",
		}, nil
	}

	output := WeatherForecastOutput{
		Success:      true,
		Message:      fmt.Sprintf("Successfully retrieved weather for %s.", geoResult.Name),
		Location:     location,
		ResolvedName: buildResolvedLocationName(geoResult.Name, geoResult.Admin1),
		Country:      geoResult.Country,
		Timezone:     forecast.Timezone,
		CurrentWeather: &CurrentWeatherSummary{
			Time:                 forecast.Current.Time,
			TemperatureC:         forecast.Current.Temperature2M,
			ApparentTemperatureC: forecast.Current.ApparentTemperature2M,
			RelativeHumidity:     forecast.Current.RelativeHumidity2M,
			WindSpeedKmh:         forecast.Current.WindSpeed10M,
			PrecipitationMm:      forecast.Current.Precipitation,
			WeatherDescription:   weatherCodeDescription(forecast.Current.WeatherCode),
		},
		DailyForecasts: make([]DailyForecastSummary, 0, len(forecast.Daily.Time)),
	}

	for i := range forecast.Daily.Time {
		item := DailyForecastSummary{
			Date:                     forecast.Daily.Time[i],
			WeatherDescription:       weatherCodeDescription(forecast.Daily.WeatherCode[i]),
			TempMaxC:                 forecast.Daily.Temperature2MMax[i],
			TempMinC:                 forecast.Daily.Temperature2MMin[i],
			PrecipitationProbability: forecast.Daily.PrecipitationProbabilityMax[i],
			PrecipitationSumMm:       forecast.Daily.PrecipitationSum[i],
			WindSpeedMaxKmh:          forecast.Daily.WindSpeed10MMax[i],
		}
		output.DailyForecasts = append(output.DailyForecasts, item)
	}
	output.TravelAdvice = buildWeatherTravelAdvice(output)

	return output, nil
}

func FormatWeatherReply(output WeatherForecastOutput) string {
	if !output.Success {
		if output.Location != "" {
			return fmt.Sprintf("我暂时没能查到 %s 的天气信息。\n\n原因：%s", output.Location, output.Message)
		}
		return "我暂时没能查到天气信息。"
	}

	locationName := output.ResolvedName
	if locationName == "" {
		locationName = output.Location
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("## %s 天气概览\n", locationName))
	if output.CurrentWeather != nil {
		builder.WriteString(fmt.Sprintf("- 当前天气：%s\n", output.CurrentWeather.WeatherDescription))
		builder.WriteString(fmt.Sprintf("- 当前温度：%.1f°C，体感 %.1f°C\n", output.CurrentWeather.TemperatureC, output.CurrentWeather.ApparentTemperatureC))
		builder.WriteString(fmt.Sprintf("- 湿度：%d%%，风速：%.1f km/h\n", output.CurrentWeather.RelativeHumidity, output.CurrentWeather.WindSpeedKmh))
		if output.CurrentWeather.PrecipitationMm > 0 {
			builder.WriteString(fmt.Sprintf("- 当前降水：%.1f mm\n", output.CurrentWeather.PrecipitationMm))
		}
	}

	if len(output.DailyForecasts) > 0 {
		builder.WriteString("\n## 未来几天\n")
		for i, daily := range output.DailyForecasts {
			if i >= 3 {
				break
			}
			builder.WriteString(fmt.Sprintf("- %s：%s，%.1f°C 到 %.1f°C，降水概率 %d%%\n",
				daily.Date,
				daily.WeatherDescription,
				daily.TempMinC,
				daily.TempMaxC,
				daily.PrecipitationProbability,
			))
		}
	}

	if output.TravelAdvice != "" {
		builder.WriteString("\n## 出行建议\n")
		builder.WriteString("- " + output.TravelAdvice + "\n")
	}

	builder.WriteString("\n如果你愿意，我还可以继续帮你把天气影响融入杭州的具体行程安排。")
	return builder.String()
}

type CurrentWeatherSummary struct {
	Time                 string  `json:"time"`
	TemperatureC         float64 `json:"temperature_c"`
	ApparentTemperatureC float64 `json:"apparent_temperature_c"`
	RelativeHumidity     int     `json:"relative_humidity"`
	WindSpeedKmh         float64 `json:"wind_speed_kmh"`
	PrecipitationMm      float64 `json:"precipitation_mm"`
	WeatherDescription   string  `json:"weather_description"`
}

type DailyForecastSummary struct {
	Date                     string  `json:"date"`
	WeatherDescription       string  `json:"weather_description"`
	TempMaxC                 float64 `json:"temp_max_c"`
	TempMinC                 float64 `json:"temp_min_c"`
	PrecipitationProbability int     `json:"precipitation_probability_max"`
	PrecipitationSumMm       float64 `json:"precipitation_sum_mm"`
	WindSpeedMaxKmh          float64 `json:"wind_speed_max_kmh"`
}

type openMeteoGeocodingResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Country   string  `json:"country"`
		Admin1    string  `json:"admin1"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Timezone  string  `json:"timezone"`
	} `json:"results"`
}

type openMeteoForecastResponse struct {
	Timezone string `json:"timezone"`
	Current  struct {
		Time                  string  `json:"time"`
		Temperature2M         float64 `json:"temperature_2m"`
		RelativeHumidity2M    int     `json:"relative_humidity_2m"`
		ApparentTemperature2M float64 `json:"apparent_temperature"`
		Precipitation         float64 `json:"precipitation"`
		WeatherCode           int     `json:"weather_code"`
		WindSpeed10M          float64 `json:"wind_speed_10m"`
	} `json:"current"`
	Daily struct {
		Time                        []string  `json:"time"`
		WeatherCode                 []int     `json:"weather_code"`
		Temperature2MMax            []float64 `json:"temperature_2m_max"`
		Temperature2MMin            []float64 `json:"temperature_2m_min"`
		PrecipitationProbabilityMax []int     `json:"precipitation_probability_max"`
		PrecipitationSum            []float64 `json:"precipitation_sum"`
		WindSpeed10MMax             []float64 `json:"wind_speed_10m_max"`
	} `json:"daily"`
}

func NewWeatherForecastTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"query_weather_forecast",
		"Get current weather and short-term forecast for a destination. Use this tool when the user asks whether a city is suitable to visit now, what weather to expect in the next few days, what to wear, or whether rain, heat, or wind may affect travel plans.",
		func(ctx context.Context, input *WeatherForecastInput, opts ...tool.Option) (string, error) {
			output, err := QueryWeatherForecast(ctx, input.Location, input.ForecastDays)
			if err != nil {
				return "", err
			}
			return marshalWeatherOutput(output)
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func geocodeLocation(ctx context.Context, location string) (*struct {
	Name      string
	Country   string
	Admin1    string
	Latitude  float64
	Longitude float64
	Timezone  string
}, error) {
	params := url.Values{}
	params.Set("name", location)
	params.Set("count", "1")
	params.Set("language", "zh")
	params.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openMeteoGeocodingURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	respBody, err := doHTTPJSONRequest(req)
	if err != nil {
		return nil, err
	}

	var response openMeteoGeocodingResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, err
	}
	if len(response.Results) == 0 {
		return nil, nil
	}

	first := response.Results[0]
	return &struct {
		Name      string
		Country   string
		Admin1    string
		Latitude  float64
		Longitude float64
		Timezone  string
	}{
		Name:      first.Name,
		Country:   first.Country,
		Admin1:    first.Admin1,
		Latitude:  first.Latitude,
		Longitude: first.Longitude,
		Timezone:  first.Timezone,
	}, nil
}

func fetchWeatherForecast(ctx context.Context, latitude, longitude float64, forecastDays int) (*openMeteoForecastResponse, error) {
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%.4f", latitude))
	params.Set("longitude", fmt.Sprintf("%.4f", longitude))
	params.Set("current", "temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,weather_code,wind_speed_10m")
	params.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,precipitation_sum,wind_speed_10m_max")
	params.Set("forecast_days", fmt.Sprintf("%d", forecastDays))
	params.Set("timezone", "auto")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openMeteoForecastURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	respBody, err := doHTTPJSONRequest(req)
	if err != nil {
		return nil, err
	}

	var response openMeteoForecastResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func doHTTPJSONRequest(req *http.Request) ([]byte, error) {
	httpClient := &http.Client{Timeout: 20 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func buildResolvedLocationName(name, admin1 string) string {
	if admin1 == "" || admin1 == name {
		return name
	}
	return name + ", " + admin1
}

func buildWeatherTravelAdvice(output WeatherForecastOutput) string {
	if output.CurrentWeather == nil || len(output.DailyForecasts) == 0 {
		return ""
	}

	first := output.DailyForecasts[0]
	parts := make([]string, 0, 3)
	if first.PrecipitationProbability >= 60 || first.PrecipitationSumMm >= 8 {
		parts = append(parts, "近期降水概率偏高，行程里尽量减少纯户外长时间停留，并准备雨具。")
	}
	if first.TempMaxC >= 32 {
		parts = append(parts, "白天偏热，建议把暴晒型景点安排到早晚，午间留给室内或休息。")
	}
	if first.TempMinC <= 8 {
		parts = append(parts, "早晚偏凉，建议带一层保暖外套，夜游时尤其注意体感温差。")
	}
	if first.WindSpeedMaxKmh >= 35 {
		parts = append(parts, "风力较明显，高处观景、骑行或海边行程要适当留意体感和安全。")
	}
	if len(parts) == 0 {
		parts = append(parts, "未来几天天气整体较平稳，可以正常安排大多数城市观光和步行类行程。")
	}
	return strings.Join(parts, "")
}

func weatherCodeDescription(code int) string {
	switch code {
	case 0:
		return "晴朗"
	case 1:
		return "大致晴"
	case 2:
		return "局部多云"
	case 3:
		return "阴天"
	case 45, 48:
		return "雾"
	case 51, 53, 55:
		return "毛毛雨"
	case 56, 57:
		return "冻毛毛雨"
	case 61:
		return "小雨"
	case 63:
		return "中雨"
	case 65:
		return "大雨"
	case 66, 67:
		return "冻雨"
	case 71:
		return "小雪"
	case 73:
		return "中雪"
	case 75:
		return "大雪"
	case 77:
		return "冰粒"
	case 80:
		return "阵雨"
	case 81:
		return "较强阵雨"
	case 82:
		return "强阵雨"
	case 85:
		return "阵雪"
	case 86:
		return "强阵雪"
	case 95:
		return "雷暴"
	case 96, 99:
		return "强雷暴伴冰雹"
	default:
		return fmt.Sprintf("天气代码 %d", code)
	}
}

func marshalWeatherOutput(output WeatherForecastOutput) (string, error) {
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
