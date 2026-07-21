package service

import "testing"

func TestQRGenerate(t *testing.T) {
	svc := NewQRService()
	res, err := svc.Generate("MyCoolBot", "", 128)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if res.BotUsername != "@MyCoolBot" {
		t.Fatalf("BotUsername = %q", res.BotUsername)
	}
	if res.BotURL != "https://t.me/MyCoolBot" {
		t.Fatalf("BotURL = %q", res.BotURL)
	}
	if len(res.PNG) < 50 {
		t.Fatalf("PNG too small: %d", len(res.PNG))
	}
}
