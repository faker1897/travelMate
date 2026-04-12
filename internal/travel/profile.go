package travel

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Profile struct {
	Destination   string
	TravelDate    string
	Days          int
	Budget        string
	Companions    string
	Interests     []string
	Pace          string
	TransportMode string
	DepartureCity string
	ArrivalCity   string
	DepartureDate string
	ReturnDate    string
}

func ExtractProfile(text string) Profile {
	normalized := normalizeText(text)
	profile := Profile{}

	departureCity, arrivalCity := extractRouteCities(normalized)
	if departureCity != "" {
		profile.DepartureCity = departureCity
	}
	if arrivalCity != "" {
		profile.ArrivalCity = arrivalCity
		profile.Destination = arrivalCity
	}
	if destination := extractDestination(normalized); destination != "" && profile.Destination == "" {
		profile.Destination = destination
	}
	if travelDate := extractTravelDate(normalized); travelDate != "" {
		profile.TravelDate = travelDate
		profile.DepartureDate = travelDate
	}
	if days := extractDays(normalized); days > 0 {
		profile.Days = days
	}
	if budget := extractBudget(normalized); budget != "" {
		profile.Budget = budget
	}
	if companions := extractCompanions(normalized); companions != "" {
		profile.Companions = companions
	}
	profile.Interests = extractInterests(normalized)
	if pace := extractPace(normalized); pace != "" {
		profile.Pace = pace
	}
	if transportMode := extractTransportMode(normalized); transportMode != "" {
		profile.TransportMode = transportMode
	}
	if returnDate := extractReturnDate(normalized); returnDate != "" {
		profile.ReturnDate = returnDate
	}
	if profile.ArrivalCity == "" && profile.Destination != "" && IsBookingIntent(normalized) {
		profile.ArrivalCity = profile.Destination
	}

	return profile
}

func MergeProfile(base, update Profile) Profile {
	if update.Destination != "" {
		base.Destination = update.Destination
	}
	if update.TravelDate != "" {
		base.TravelDate = update.TravelDate
	}
	if update.Days > 0 {
		base.Days = update.Days
	}
	if update.Budget != "" {
		base.Budget = update.Budget
	}
	if update.Companions != "" {
		base.Companions = update.Companions
	}
	if update.Pace != "" {
		base.Pace = update.Pace
	}
	if update.TransportMode != "" {
		base.TransportMode = update.TransportMode
	}
	if update.DepartureCity != "" {
		base.DepartureCity = update.DepartureCity
	}
	if update.ArrivalCity != "" {
		base.ArrivalCity = update.ArrivalCity
	}
	if update.DepartureDate != "" {
		base.DepartureDate = update.DepartureDate
	}
	if update.ReturnDate != "" {
		base.ReturnDate = update.ReturnDate
	}
	base.Interests = mergeInterests(base.Interests, update.Interests)
	if base.ArrivalCity == "" && base.Destination != "" && base.TransportMode != "" {
		base.ArrivalCity = base.Destination
	}
	if base.DepartureDate == "" && base.TravelDate != "" && base.TransportMode != "" {
		base.DepartureDate = base.TravelDate
	}
	return base
}

func (p Profile) MissingSlots() []string {
	missing := make([]string, 0, 6)
	if p.Destination == "" {
		missing = append(missing, "目的地")
	}
	if p.TravelDate == "" {
		missing = append(missing, "出行时间")
	}
	if p.Days == 0 {
		missing = append(missing, "游玩天数")
	}
	if p.Budget == "" {
		missing = append(missing, "预算")
	}
	if p.Companions == "" {
		missing = append(missing, "同行人")
	}
	if len(p.Interests) == 0 && p.Pace == "" {
		missing = append(missing, "玩法偏好")
	}
	return missing
}

func (p Profile) Summary() string {
	parts := make([]string, 0, 12)
	if p.Destination != "" {
		parts = append(parts, fmt.Sprintf("目的地：%s", p.Destination))
	}
	if p.TravelDate != "" {
		parts = append(parts, fmt.Sprintf("出行时间：%s", p.TravelDate))
	}
	if p.Days > 0 {
		parts = append(parts, fmt.Sprintf("游玩天数：%d 天", p.Days))
	}
	if p.Budget != "" {
		parts = append(parts, fmt.Sprintf("预算：%s", p.Budget))
	}
	if p.Companions != "" {
		parts = append(parts, fmt.Sprintf("同行人：%s", p.Companions))
	}
	if len(p.Interests) > 0 {
		parts = append(parts, fmt.Sprintf("偏好：%s", strings.Join(p.Interests, "、")))
	}
	if p.Pace != "" {
		parts = append(parts, fmt.Sprintf("节奏：%s", p.Pace))
	}
	if p.TransportMode != "" {
		parts = append(parts, fmt.Sprintf("预订交通：%s", transportModeLabel(p.TransportMode)))
	}
	if p.DepartureCity != "" {
		parts = append(parts, fmt.Sprintf("预订出发地：%s", p.DepartureCity))
	}
	if p.EffectiveArrivalCity() != "" && p.TransportMode != "" {
		parts = append(parts, fmt.Sprintf("预订到达地：%s", p.EffectiveArrivalCity()))
	}
	if p.EffectiveDepartureDate() != "" && p.TransportMode != "" {
		parts = append(parts, fmt.Sprintf("预订出发日期：%s", p.EffectiveDepartureDate()))
	}
	if p.ReturnDate != "" {
		parts = append(parts, fmt.Sprintf("预订返程日期：%s", p.ReturnDate))
	}
	if len(parts) == 0 {
		return "尚未收集到明确的旅行需求。"
	}
	return strings.Join(parts, "\n")
}

func (p Profile) ResponseMode() string {
	if p.Destination == "" {
		return "discover_destination"
	}

	supportingInfoCount := 0
	if p.TravelDate != "" {
		supportingInfoCount++
	}
	if p.Budget != "" {
		supportingInfoCount++
	}
	if p.Companions != "" {
		supportingInfoCount++
	}
	if len(p.Interests) > 0 || p.Pace != "" {
		supportingInfoCount++
	}

	if p.Days == 0 || supportingInfoCount < 2 {
		return "clarify_constraints"
	}
	return "recommend_plan"
}

func (p Profile) StageSummary() string {
	switch p.ResponseMode() {
	case "discover_destination":
		return "当前阶段：先帮用户缩小想去哪里玩，再补齐约束。"
	case "clarify_constraints":
		return "当前阶段：目的地已有方向，但还需要补齐几项关键约束，再继续展开。"
	default:
		return "当前阶段：信息已经足够给出方向性建议和分日行程框架。"
	}
}

func (p Profile) ResponseTemplateHint() string {
	switch p.ResponseMode() {
	case "discover_destination":
		return strings.TrimSpace(`
请优先使用下面的回答结构：
## 先帮你缩小方向
- 用 1 到 2 句话概括适合的旅行方向或城市类型

## 我还想确认几个关键点
- 最多追问 3 到 5 个问题
- 问题必须短、可直接回答

## 你可以直接这样告诉我
- 目的地/城市倾向：
- 什么时候出发：
- 玩几天：
- 预算大概：
- 同行人和偏好：
`)
	case "clarify_constraints":
		return strings.TrimSpace(`
请优先使用下面的回答结构：
## 目前我对你这趟旅行的理解
- 用 2 到 4 条总结已知需求

## 我还需要补充的信息
- 只追问最关键的 2 到 4 项

## 先给你一个方向
- 在信息未补齐前，先给 2 到 4 条高层建议
`)
	default:
		return strings.TrimSpace(`
请优先使用下面的回答结构：
## 你这趟旅行适合怎么玩
- 先给总体判断和玩法定位

## 推荐安排
- 按 Day 1 / Day 2 / Day 3 输出分日框架
- 每天写清楚上午、下午、晚上或核心动线

## 额外建议
- 补充交通、住宿区域、餐饮或避坑提醒

## 如果你愿意，我下一步可以继续细化
- 告诉用户你可以继续展开门票预约、住宿区域、预算拆分或更细日程
`)
	}
}

func (p Profile) StructuredOutputHint() string {
	switch p.ResponseMode() {
	case "discover_destination":
		return strings.TrimSpace(`
请尽量使用下面的固定结构输出：
## 你可能适合的旅行方向
- 给 2 到 4 个方向或城市类型，不要一次铺太多

## 为什么这样推荐
- 结合用户已知信息说明推荐依据

## 我还想确认几个关键点
- 追问 3 到 5 个问题

## 你可以直接这样回复我
- 城市或地区偏好：
- 什么时候出发：
- 玩几天：
- 预算：
- 同行人和偏好：
`)
	case "clarify_constraints":
		return strings.TrimSpace(`
请尽量使用下面的固定结构输出：
## 我目前对这趟旅行的理解
- 总结当前已知需求

## 可以先给你的方向建议
- 先给 2 到 4 条高层建议

## 还需要补充的信息
- 只追问最关键的 2 到 4 项

## 补充完后我可以继续帮你做什么
- 说明可继续细化成住宿区域、交通安排或分日行程
`)
	default:
		dayPlanHint := p.dayPlanHint()
		return strings.TrimSpace(`
请尽量使用下面的固定结构输出：
## 总体判断
- 先概括这趟旅行适合怎么玩、适不适合当前用户

## 推荐行程
` + dayPlanHint + `

## 住宿建议
- 推荐 1 到 3 个适合落脚的区域，并说明理由

## 交通建议
- 说明大交通/市内交通的优先方式

## 美食与体验
- 补充值得安排的餐饮或特色体验

## 避坑提醒
- 给 2 到 4 条真正有用的提醒

## 如果你愿意，我下一步还能继续细化
- 说明可继续展开预算拆分、门票预约、餐厅选择或更细日程
`)
	}
}

func (p Profile) dayPlanHint() string {
	if p.Days <= 0 {
		return `- 如果当前信息足够，可先给一个分日框架草案；如果还不够，就先给不超过 3 段的路线建议`
	}

	if p.Days == 1 {
		return `### Day 1
- 按 上午 / 下午 / 晚上 来安排，尽量保证动线顺`
	}

	if p.Days > 5 {
		return fmt.Sprintf(`- 按 Day 1 到 Day %d 输出，但可以把相近日期合并成重点日程，避免过度冗长`, p.Days)
	}

	parts := make([]string, 0, p.Days)
	for i := 1; i <= p.Days; i++ {
		parts = append(parts, fmt.Sprintf(`### Day %d
- 写清楚当天核心区域、主要景点和节奏`, i))
	}
	return strings.Join(parts, "\n")
}

func normalizeText(text string) string {
	replacer := strings.NewReplacer("，", " ", "。", " ", "、", " ", ",", " ", "\n", " ", "\t", " ")
	return strings.Join(strings.Fields(replacer.Replace(strings.TrimSpace(text))), " ")
}

var cityList = []string{
	"北京", "上海", "广州", "深圳", "杭州", "南京", "苏州", "成都", "重庆", "西安",
	"武汉", "长沙", "青岛", "厦门", "三亚", "昆明", "大理", "丽江", "哈尔滨", "天津",
	"洛阳", "开封", "扬州", "桂林", "拉萨", "敦煌", "福州", "珠海", "汕头", "泉州",
	"香港", "澳门",
}

func extractDestination(text string) string {
	for _, city := range cityList {
		if strings.Contains(text, city) {
			return city
		}
	}

	patterns := []string{
		`想去([^\s]{2,12})`,
		`去([^\s]{2,12})(?:玩|旅游|旅行|逛|看看)`,
		`在([^\s]{2,12})(?:玩|旅游|旅行)`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			candidate := strings.Trim(matches[1], "的呢吧呀啊")
			if len([]rune(candidate)) >= 2 && len([]rune(candidate)) <= 12 {
				return candidate
			}
		}
	}
	return ""
}

func extractRouteCities(text string) (string, string) {
	citiesPattern := strings.Join(cityList, "|")
	patterns := []string{
		fmt.Sprintf(`(?:从)?(%s)(?:到|去|飞|回)(%s)`, citiesPattern, citiesPattern),
		fmt.Sprintf(`(%s)\s*(?:-|—|->|→)\s*(%s)`, citiesPattern, citiesPattern),
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 2 {
			return matches[1], matches[2]
		}
	}
	return "", ""
}

func extractTravelDate(text string) string {
	keywords := []string{
		"五一", "十一", "国庆", "春节", "端午", "中秋", "元旦",
		"周末", "下周", "下个月", "暑假", "寒假", "明天", "后天", "本周末",
	}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return keyword
		}
	}

	patterns := []string{
		`([0-9]{1,2}月[0-9]{1,2}[日号]?)`,
		`([0-9]{4}年[0-9]{1,2}月[0-9]{1,2}[日号]?)`,
		`([0-9]{1,2}月)`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func extractReturnDate(text string) string {
	patterns := []string{
		`返程(?:日期)?(?:是|为)?([0-9]{1,2}月[0-9]{1,2}[日号]?)`,
		`回程(?:日期)?(?:是|为)?([0-9]{1,2}月[0-9]{1,2}[日号]?)`,
		`([0-9]{1,2}月[0-9]{1,2}[日号]?).*(?:返程|回程|返回)`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func extractDays(text string) int {
	reArabic := regexp.MustCompile(`([0-9]{1,2})\s*(?:天|日)`)
	if matches := reArabic.FindStringSubmatch(text); len(matches) > 1 {
		days, err := strconv.Atoi(matches[1])
		if err == nil {
			return days
		}
	}

	reChinese := regexp.MustCompile(`([一二两三四五六七八九十]{1,3})\s*(?:天|日)`)
	if matches := reChinese.FindStringSubmatch(text); len(matches) > 1 {
		if days := parseChineseNumber(matches[1]); days > 0 {
			return days
		}
	}
	return 0
}

func extractBudget(text string) string {
	re := regexp.MustCompile(`(?:预算|人均|总预算)?\s*([0-9]{3,6})\s*元`)
	if matches := re.FindStringSubmatch(text); len(matches) > 1 {
		return matches[1] + "元"
	}
	if strings.Contains(text, "预算不高") || strings.Contains(text, "穷游") {
		return "预算有限"
	}
	if strings.Contains(text, "预算充足") || strings.Contains(text, "不差钱") {
		return "预算充足"
	}
	return ""
}

func extractCompanions(text string) string {
	switch {
	case strings.Contains(text, "情侣"), strings.Contains(text, "对象"), strings.Contains(text, "男朋友"), strings.Contains(text, "女朋友"):
		return "情侣"
	case strings.Contains(text, "亲子"), strings.Contains(text, "带娃"), strings.Contains(text, "孩子"), strings.Contains(text, "宝宝"):
		return "亲子/家庭"
	case strings.Contains(text, "爸妈"), strings.Contains(text, "父母"), strings.Contains(text, "家庭"), strings.Contains(text, "一家人"):
		return "家庭"
	case strings.Contains(text, "闺蜜"), strings.Contains(text, "朋友"), strings.Contains(text, "同学"), strings.Contains(text, "兄弟"):
		return "朋友"
	case strings.Contains(text, "一个人"), strings.Contains(text, "独自"), strings.Contains(text, "自己去"):
		return "独自出行"
	default:
		return ""
	}
}

func extractInterests(text string) []string {
	interestMap := []struct {
		label    string
		keywords []string
	}{
		{label: "自然风景", keywords: []string{"风景", "自然", "山水", "海边", "海岛", "公园"}},
		{label: "美食", keywords: []string{"美食", "小吃", "吃", "餐厅", "夜市"}},
		{label: "历史人文", keywords: []string{"历史", "人文", "古城", "博物馆", "遗址", "寺庙"}},
		{label: "拍照出片", keywords: []string{"拍照", "出片", "打卡", "摄影"}},
		{label: "亲子体验", keywords: []string{"亲子", "带娃", "乐园", "动物园"}},
		{label: "休闲放松", keywords: []string{"休闲", "放松", "慢节奏", "度假", "发呆"}},
		{label: "夜游", keywords: []string{"夜景", "夜游", "夜生活"}},
		{label: "购物", keywords: []string{"购物", "逛街", "买买买"}},
	}

	result := make([]string, 0, len(interestMap))
	for _, item := range interestMap {
		for _, keyword := range item.keywords {
			if strings.Contains(text, keyword) {
				result = append(result, item.label)
				break
			}
		}
	}
	return result
}

func extractPace(text string) string {
	switch {
	case strings.Contains(text, "特种兵"), strings.Contains(text, "高强度"), strings.Contains(text, "紧凑"):
		return "紧凑高效"
	case strings.Contains(text, "慢节奏"), strings.Contains(text, "悠闲"), strings.Contains(text, "休闲"), strings.Contains(text, "轻松"):
		return "轻松慢游"
	default:
		return ""
	}
}

func extractTransportMode(text string) string {
	hasTrain := strings.Contains(text, "高铁") || strings.Contains(text, "火车") || strings.Contains(text, "动车") || strings.Contains(text, "列车") || strings.Contains(text, "12306")
	hasFlight := strings.Contains(text, "机票") || strings.Contains(text, "航班") || strings.Contains(text, "飞机票") || strings.Contains(text, "飞机")
	switch {
	case hasTrain && hasFlight:
		return TransportUndecided
	case hasTrain:
		return TransportTrain
	case hasFlight:
		return TransportFlight
	default:
		return ""
	}
}

func mergeInterests(base, update []string) []string {
	seen := make(map[string]struct{}, len(base)+len(update))
	result := make([]string, 0, len(base)+len(update))
	for _, item := range append(base, update...) {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func parseChineseNumber(raw string) int {
	values := map[rune]int{
		'一': 1,
		'二': 2,
		'两': 2,
		'三': 3,
		'四': 4,
		'五': 5,
		'六': 6,
		'七': 7,
		'八': 8,
		'九': 9,
		'十': 10,
	}

	if raw == "十" {
		return 10
	}
	if strings.HasPrefix(raw, "十") {
		return 10 + values[[]rune(raw)[1]]
	}
	if strings.HasSuffix(raw, "十") {
		return values[[]rune(raw)[0]] * 10
	}
	if strings.Contains(raw, "十") {
		runes := []rune(raw)
		return values[runes[0]]*10 + values[runes[2]]
	}
	if len([]rune(raw)) == 1 {
		return values[[]rune(raw)[0]]
	}
	return 0
}
