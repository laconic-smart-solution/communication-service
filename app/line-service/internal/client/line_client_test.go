package client

import "testing"

func TestParseWebhookReturnsFollowEvents(t *testing.T) {
	body := []byte(`{"events":[
		{"type":"follow","replyToken":"rt-1","source":{"userId":"U123"}},
		{"type":"message","replyToken":"rt-2","message":{"id":"m-1","type":"text","text":"hi"},"source":{"userId":"U456"}},
		{"type":"message","message":{"id":"m-2","type":"sticker"},"source":{"userId":"U789"}},
		{"type":"unfollow","source":{"userId":"U123"}},
		{"type":"join","source":{"userId":"U999"}}
	]}`)

	events, err := NewLINEClient(5).ParseWebhook(body)
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3 (follow, text message, unfollow)", len(events))
	}

	if events[0].Type != EventTypeFollow || events[0].UserID != "U123" || events[0].ReplyToken != "rt-1" {
		t.Errorf("follow event = %+v", events[0])
	}
	if events[1].Type != EventTypeMessage || events[1].Text != "hi" {
		t.Errorf("message event = %+v", events[1])
	}
	if events[2].Type != EventTypeUnfollow || events[2].UserID != "U123" {
		t.Errorf("unfollow event = %+v", events[2])
	}
}
