package travel

import "strings"

func IsWeatherIntent(text string) bool {
	normalized := normalizeText(text)
	keywords := []string{
		"天气", "气温", "温度", "下雨", "降雨", "晴", "阴", "多云", "穿什么", "带什么衣服",
		"适合去", "适不适合去", "会不会下雨", "热不热", "冷不冷", "天气如何",
	}
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func IsBookingIntent(text string) bool {
	normalized := normalizeText(text)
	keywords := []string{
		"订票", "买票", "预订", "购票", "抢票", "帮我订", "帮我买",
		"机票", "航班", "飞机票", "高铁", "火车", "动车", "列车", "12306",
	}
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}
