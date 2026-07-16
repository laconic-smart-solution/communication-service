package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrRateLimited      = errors.New("line api rate limited")
	ErrInvalidToken     = errors.New("invalid line channel token")
)

type SendMessageInput struct {
	AccessToken string
	RecipientID string
	Text        string
}

// Event types forwarded to the calling app.
const (
	EventTypeMessage  = "message"
	EventTypeFollow   = "follow"
	EventTypeUnfollow = "unfollow"
)

type InboundEvent struct {
	Type       string `json:"type"`
	UserID     string `json:"user_id"`
	ReplyToken string `json:"reply_token,omitempty"`
	MessageID  string `json:"message_id,omitempty"`
	Text       string `json:"text,omitempty"`
}

type LINEClient interface {
	SendMessage(ctx context.Context, in SendMessageInput) error
	VerifyWebhook(signature string, body []byte, channelSecret string) error
	ParseWebhook(body []byte) ([]InboundEvent, error)
	ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error
}

type lineClientImpl struct {
	httpClient *http.Client
}

var _ LINEClient = (*lineClientImpl)(nil)

func NewLINEClient(timeoutSec int) LINEClient {
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &lineClientImpl{
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *lineClientImpl) SendMessage(ctx context.Context, in SendMessageInput) error {
	payload := map[string]interface{}{
		"to": in.RecipientID,
		"messages": []map[string]string{
			{"type": "text", "text": in.Text},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal push message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.line.me/v2/bot/message/push", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+in.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("line api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrInvalidToken
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("line api status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *lineClientImpl) VerifyWebhook(signature string, body []byte, channelSecret string) error {
	if channelSecret == "" {
		return errors.New("line channel secret is not configured")
	}
	if signature == "" {
		return ErrInvalidSignature
	}

	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}
	return nil
}

func (c *lineClientImpl) ParseWebhook(body []byte) ([]InboundEvent, error) {
	var payload struct {
		Events []struct {
			Type       string `json:"type"`
			ReplyToken string `json:"replyToken"`
			Message    struct {
				ID   string `json:"id"`
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"message"`
			Source struct {
				UserID string `json:"userId"`
			} `json:"source"`
		} `json:"events"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse webhook: %w", err)
	}

	out := make([]InboundEvent, 0, len(payload.Events))
	for _, ev := range payload.Events {
		switch ev.Type {
		case EventTypeMessage:
			if ev.Message.Type != "text" {
				continue
			}
			out = append(out, InboundEvent{
				Type:       ev.Type,
				UserID:     ev.Source.UserID,
				ReplyToken: ev.ReplyToken,
				MessageID:  ev.Message.ID,
				Text:       ev.Message.Text,
			})
		case EventTypeFollow, EventTypeUnfollow:
			// A follow is LINE's "user added the OA as a friend" signal; there is
			// nothing to accept, so it is forwarded for the calling app to greet.
			out = append(out, InboundEvent{
				Type:       ev.Type,
				UserID:     ev.Source.UserID,
				ReplyToken: ev.ReplyToken,
			})
		}
	}
	return out, nil
}

func (c *lineClientImpl) ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error {
	payload, err := json.Marshal(map[string]interface{}{
		"events": events,
	})
	if err != nil {
		return fmt.Errorf("marshal callback payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create callback request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("callback request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("callback status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
