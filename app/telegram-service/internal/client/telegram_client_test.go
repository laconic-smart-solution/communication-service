package client

import "testing"

func TestParseWebhookReturnsFollowMessageUnfollow(t *testing.T) {
	c := NewTelegramClient(5)

	followBody := []byte(`{
		"update_id": 1,
		"my_chat_member": {
			"chat": {"id": 100},
			"from": {"id": 200},
			"new_chat_member": {"status": "member"}
		}
	}`)
	events, err := c.ParseWebhook(followBody)
	if err != nil {
		t.Fatalf("ParseWebhook follow error = %v", err)
	}
	if len(events) != 1 || events[0].Type != EventTypeFollow || events[0].UserID != "200" {
		t.Fatalf("follow events = %+v", events)
	}

	msgBody := []byte(`{
		"update_id": 2,
		"message": {
			"message_id": 9,
			"from": {"id": 300},
			"chat": {"id": 300},
			"text": "hello"
		}
	}`)
	events, err = c.ParseWebhook(msgBody)
	if err != nil {
		t.Fatalf("ParseWebhook message error = %v", err)
	}
	if len(events) != 1 || events[0].Type != EventTypeMessage || events[0].Text != "hello" || events[0].MessageID != "9" {
		t.Fatalf("message events = %+v", events)
	}

	startBody := []byte(`{
		"update_id": 3,
		"message": {
			"message_id": 10,
			"from": {"id": 400},
			"chat": {"id": 400},
			"text": "/start payload"
		}
	}`)
	events, err = c.ParseWebhook(startBody)
	if err != nil {
		t.Fatalf("ParseWebhook start error = %v", err)
	}
	if len(events) != 2 || events[0].Type != EventTypeFollow || events[1].Type != EventTypeMessage {
		t.Fatalf("start events = %+v", events)
	}

	unfollowBody := []byte(`{
		"update_id": 4,
		"my_chat_member": {
			"chat": {"id": 100},
			"from": {"id": 200},
			"new_chat_member": {"status": "kicked"}
		}
	}`)
	events, err = c.ParseWebhook(unfollowBody)
	if err != nil {
		t.Fatalf("ParseWebhook unfollow error = %v", err)
	}
	if len(events) != 1 || events[0].Type != EventTypeUnfollow {
		t.Fatalf("unfollow events = %+v", events)
	}
}

func TestNormalizeBotUsername(t *testing.T) {
	got, err := NormalizeBotUsername("MyBot_name")
	if err != nil {
		t.Fatalf("NormalizeBotUsername() error = %v", err)
	}
	if got != "@MyBot_name" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeBotUsername("ab"); err == nil {
		t.Fatal("expected short username error")
	}
}

func TestBotDeepLink(t *testing.T) {
	u, err := BotDeepLink("@MyCoolBot", "welcome")
	if err != nil {
		t.Fatalf("BotDeepLink() error = %v", err)
	}
	if u != "https://t.me/MyCoolBot?start=welcome" {
		t.Fatalf("got %q", u)
	}
}
