package client

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook secret token")
	ErrRateLimited      = errors.New("telegram api rate limited")
	ErrInvalidToken     = errors.New("invalid telegram bot token")
)

type SendMessageInput struct {
	BotToken    string
	RecipientID string
	Text        string
}

// Event types forwarded to the calling app (aligned with line-service).
const (
	EventTypeMessage  = "message"
	EventTypeFollow   = "follow"
	EventTypeUnfollow = "unfollow"
)

type InboundEvent struct {
	Type      string `json:"type"`
	UserID    string `json:"user_id"`
	MessageID string `json:"message_id,omitempty"`
	Text      string `json:"text,omitempty"`
}

type TelegramClient interface {
	SendMessage(ctx context.Context, in SendMessageInput) error
	VerifyWebhook(secretHeader, secretToken string) error
	ParseWebhook(body []byte) ([]InboundEvent, error)
	ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error
}

type telegramClientImpl struct {
	httpClient *http.Client
	apiBase    string
}

var _ TelegramClient = (*telegramClientImpl)(nil)

func NewTelegramClient(timeoutSec int) TelegramClient {
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &telegramClientImpl{
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		apiBase:    "https://api.telegram.org",
	}
}

func (c *telegramClientImpl) SendMessage(ctx context.Context, in SendMessageInput) error {
	if strings.TrimSpace(in.BotToken) == "" {
		return ErrInvalidToken
	}
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", c.apiBase, in.BotToken)
	payload := map[string]interface{}{
		"chat_id": in.RecipientID,
		"text":    in.Text,
	}
	return c.postJSON(ctx, endpoint, payload)
}

func (c *telegramClientImpl) postJSON(ctx context.Context, endpoint string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram api request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))

	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrInvalidToken
	}

	var envelope struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		ErrorCode   int    `json:"error_code"`
	}
	_ = json.Unmarshal(raw, &envelope)

	if resp.StatusCode == 429 || envelope.ErrorCode == 429 {
		return ErrRateLimited
	}
	if envelope.ErrorCode == 401 || envelope.ErrorCode == 403 {
		return ErrInvalidToken
	}
	if resp.StatusCode >= 300 || !envelope.OK {
		if envelope.Description != "" {
			return fmt.Errorf("telegram api status %d: %s", resp.StatusCode, envelope.Description)
		}
		return fmt.Errorf("telegram api status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (c *telegramClientImpl) VerifyWebhook(secretHeader, secretToken string) error {
	secretToken = strings.TrimSpace(secretToken)
	if secretToken == "" {
		return errors.New("telegram webhook secret_token is not configured")
	}
	if subtle.ConstantTimeCompare([]byte(secretHeader), []byte(secretToken)) != 1 {
		return ErrInvalidSignature
	}
	return nil
}

func (c *telegramClientImpl) ParseWebhook(body []byte) ([]InboundEvent, error) {
	var update struct {
		Message *struct {
			MessageID int64 `json:"message_id"`
			From      *struct {
				ID int64 `json:"id"`
			} `json:"from"`
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			Text string `json:"text"`
		} `json:"message"`
		MyChatMember *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			From struct {
				ID int64 `json:"id"`
			} `json:"from"`
			NewChatMember struct {
				Status string `json:"status"`
			} `json:"new_chat_member"`
		} `json:"my_chat_member"`
	}
	if err := json.Unmarshal(body, &update); err != nil {
		return nil, fmt.Errorf("parse webhook: %w", err)
	}

	out := make([]InboundEvent, 0, 1)

	if update.MyChatMember != nil {
		userID := strconv.FormatInt(update.MyChatMember.From.ID, 10)
		if userID == "0" {
			userID = strconv.FormatInt(update.MyChatMember.Chat.ID, 10)
		}
		switch update.MyChatMember.NewChatMember.Status {
		case "member", "administrator", "restricted":
			out = append(out, InboundEvent{Type: EventTypeFollow, UserID: userID})
		case "left", "kicked":
			out = append(out, InboundEvent{Type: EventTypeUnfollow, UserID: userID})
		}
	}

	if update.Message != nil {
		text := strings.TrimSpace(update.Message.Text)
		if text == "" {
			return out, nil
		}
		userID := strconv.FormatInt(update.Message.Chat.ID, 10)
		if update.Message.From != nil && update.Message.From.ID != 0 {
			userID = strconv.FormatInt(update.Message.From.ID, 10)
		}
		// /start is Telegram's common "user opened the bot" signal.
		if strings.HasPrefix(text, "/start") {
			out = append(out, InboundEvent{Type: EventTypeFollow, UserID: userID})
		}
		out = append(out, InboundEvent{
			Type:      EventTypeMessage,
			UserID:    userID,
			MessageID: strconv.FormatInt(update.Message.MessageID, 10),
			Text:      text,
		})
	}

	return out, nil
}

func (c *telegramClientImpl) ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error {
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

// BotDeepLink builds https://t.me/<username> optionally with ?start=
func BotDeepLink(botUsername, startPayload string) (string, error) {
	normalized, err := NormalizeBotUsername(botUsername)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(normalized, "@")
	u := "https://t.me/" + url.PathEscape(name)
	startPayload = strings.TrimSpace(startPayload)
	if startPayload != "" {
		u += "?start=" + url.QueryEscape(startPayload)
	}
	return u, nil
}

// NormalizeBotUsername returns @Name after validating Telegram bot username rules.
func NormalizeBotUsername(botUsername string) (string, error) {
	trimmed := strings.TrimSpace(botUsername)
	if trimmed == "" {
		return "", errors.New("bot_username query parameter is required")
	}
	if !strings.HasPrefix(trimmed, "@") {
		trimmed = "@" + trimmed
	}
	name := strings.TrimPrefix(trimmed, "@")
	if len(name) < 5 || len(name) > 32 {
		return "", errors.New("bot_username must be 5–32 characters (Telegram bot username rules)")
	}
	for _, r := range name {
		ok := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_'
		if !ok {
			return "", errors.New("bot_username may only contain letters, digits, and underscores")
		}
	}
	return trimmed, nil
}
