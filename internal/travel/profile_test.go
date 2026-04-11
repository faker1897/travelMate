package travel

import (
	"strings"
	"testing"
)

func TestExtractProfile(t *testing.T) {
	profile := ExtractProfile("我想五一去杭州玩3天，预算3000元，情侣出行，想吃好吃的顺便拍照。")
	if profile.Destination != "杭州" {
		t.Fatalf("expected destination 杭州, got %q", profile.Destination)
	}
	if profile.TravelDate != "五一" {
		t.Fatalf("expected travel date 五一, got %q", profile.TravelDate)
	}
	if profile.Days != 3 {
		t.Fatalf("expected 3 days, got %d", profile.Days)
	}
	if profile.Budget != "3000元" {
		t.Fatalf("expected budget 3000元, got %q", profile.Budget)
	}
	if profile.Companions != "情侣" {
		t.Fatalf("expected companions 情侣, got %q", profile.Companions)
	}
	if len(profile.Interests) == 0 {
		t.Fatalf("expected interests to be extracted")
	}
}

func TestMergeProfile(t *testing.T) {
	base := Profile{Destination: "北京", Interests: []string{"历史人文"}}
	update := Profile{Days: 4, Interests: []string{"美食"}}
	merged := MergeProfile(base, update)
	if merged.Destination != "北京" {
		t.Fatalf("expected destination to stay 北京, got %q", merged.Destination)
	}
	if merged.Days != 4 {
		t.Fatalf("expected days to update to 4, got %d", merged.Days)
	}
	if len(merged.Interests) != 2 {
		t.Fatalf("expected merged interests, got %v", merged.Interests)
	}
}

func TestResponseMode(t *testing.T) {
	if mode := (Profile{}).ResponseMode(); mode != "discover_destination" {
		t.Fatalf("expected discover_destination, got %q", mode)
	}

	mode := (Profile{
		Destination: "杭州",
		Days:        3,
		TravelDate:  "五一",
		Companions:  "情侣",
		Interests:   []string{"美食"},
	}).ResponseMode()
	if mode != "recommend_plan" {
		t.Fatalf("expected recommend_plan, got %q", mode)
	}
}

func TestStructuredOutputHint(t *testing.T) {
	profile := Profile{
		Destination: "杭州",
		Days:        3,
		TravelDate:  "五一",
		Companions:  "情侣",
		Interests:   []string{"美食"},
	}
	hint := profile.StructuredOutputHint()
	if !strings.Contains(hint, "## 推荐行程") {
		t.Fatalf("expected structured itinerary section, got %q", hint)
	}
	if !strings.Contains(hint, "### Day 1") || !strings.Contains(hint, "### Day 3") {
		t.Fatalf("expected day-based plan hint, got %q", hint)
	}
}

func TestExtractBookingProfile(t *testing.T) {
	profile := ExtractProfile("我想订 5月1日上海到杭州的高铁")
	if profile.TransportMode != TransportTrain {
		t.Fatalf("expected train transport, got %q", profile.TransportMode)
	}
	if profile.DepartureCity != "上海" {
		t.Fatalf("expected departure city 上海, got %q", profile.DepartureCity)
	}
	if profile.ArrivalCity != "杭州" {
		t.Fatalf("expected arrival city 杭州, got %q", profile.ArrivalCity)
	}
	if profile.Destination != "杭州" {
		t.Fatalf("expected destination 杭州, got %q", profile.Destination)
	}
	if profile.DepartureDate != "5月1日" {
		t.Fatalf("expected departure date 5月1日, got %q", profile.DepartureDate)
	}
}

func TestFormatBookingReplyNeedsDepartureCity(t *testing.T) {
	profile := ExtractProfile("帮我订明天去杭州的高铁")
	reply := FormatBookingReply(profile)
	if !strings.Contains(reply, "还需要补充的信息") {
		t.Fatalf("expected missing fields prompt, got %q", reply)
	}
	if !strings.Contains(reply, "出发地") {
		t.Fatalf("expected departure city to be required, got %q", reply)
	}
}

func TestFormatBookingReplyForTrainReady(t *testing.T) {
	profile := ExtractProfile("我想订 5月1日上海到杭州的高铁")
	reply := FormatBookingReply(profile)
	if !strings.Contains(reply, "搜索火车") {
		t.Fatalf("expected train search link, got %q", reply)
	}
	if !strings.Contains(reply, "12306") {
		t.Fatalf("expected 12306 helper link, got %q", reply)
	}
}

func TestFormatBookingReplyForFlightWithoutDate(t *testing.T) {
	profile := ExtractProfile("帮我看下北京到成都的机票")
	reply := FormatBookingReply(profile)
	if !strings.Contains(reply, "搜索机票") {
		t.Fatalf("expected flight search link, got %q", reply)
	}
	if !strings.Contains(reply, "暂未提供") {
		t.Fatalf("expected missing date note, got %q", reply)
	}
}
