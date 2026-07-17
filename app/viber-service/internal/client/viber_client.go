package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrRateLimited      = errors.New("viber api rate limited")
	ErrInvalidToken     = errors.New("invalid viber auth token")
)

type SendMessageInput struct {
	AuthToken   string
	RecipientID string
	Text        string
	SenderName  string
}

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
	Context   string `json:"context,omitempty"`
}

type ViberClient interface {
	SendMessage(ctx context.Context, in SendMessageInput) error
	VerifyWebhook(signatureHeader string, body []byte, authToken string) error
	ParseWebhook(body []byte) ([]InboundEvent, error)
	ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error
}

type viberClientImpl struct {
	httpClient *http.Client
	apiBase    string
}

var _ ViberClient = (*viberClientImpl)(nil)

func NewViberClient(timeoutSec int) ViberClient {
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &viberClientImpl{
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		apiBase:    "https://chatapi.viber.com/pa",
	}
}

func (c *viberClientImpl) SendMessage(ctx context.Context, in SendMessageInput) error {
	token := strings.TrimSpace(in.AuthToken)
	if token == "" {
		return ErrInvalidToken
	}
	sender := strings.TrimSpace(in.SenderName)
	if sender == "" {
		sender = "Mosonic"
	}
	payload := map[string]interface{}{
		"receiver": in.RecipientID,
		"type":     "text",
		"text":     in.Text,
		"sender": map[string]string{
			"name": sender,
		},
	}
	return c.postJSON(ctx, c.apiBase+"/send_message", token, payload)
}

func (c *viberClientImpl) postJSON(ctx context.Context, endpoint, authToken string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal viber payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Viber-Auth-Token", authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("viber api request: %w", err)
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
		Status        int    `json:"status"`
		StatusMessage string `json:"status_message"`
	}
	_ = json.Unmarshal(raw, &envelope)

	// Viber uses status 0 for success; status 2 = invalid auth token.
	if envelope.Status == 2 {
		return ErrInvalidToken
	}
	if resp.StatusCode >= 300 || envelope.Status != 0 {
		if envelope.StatusMessage != "" {
			return fmt.Errorf("viber api status %d: %s", envelope.Status, envelope.StatusMessage)
		}
		return fmt.Errorf("viber api http %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (c *viberClientImpl) VerifyWebhook(signatureHeader string, body []byte, authToken string) error {
	authToken = strings.TrimSpace(authToken)
	if authToken == "" {
		return errors.New("viber auth_token is not configured")
	}
	mac := hmac.New(sha256.New, []byte(authToken))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	got := strings.TrimSpace(strings.ToLower(signatureHeader))
	if subtleConstantTimeHexEqual(got, expected) {
		return nil
	}
	return ErrInvalidSignature
}

func subtleConstantTimeHexEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func (c *viberClientImpl) ParseWebhook(body []byte) ([]InboundEvent, error) {
	var cb struct {
		Event        string `json:"event"`
		MessageToken int64  `json:"message_token"`
		Context      string `json:"context"`
		User         *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"user"`
		Sender *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"sender"`
		Message *struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &cb); err != nil {
		return nil, fmt.Errorf("parse webhook: %w", err)
	}

	out := make([]InboundEvent, 0, 1)
	switch strings.ToLower(strings.TrimSpace(cb.Event)) {
	case "webhook":
		// Viber handshake when set_webhook is called — acknowledge only.
		return out, nil
	case "subscribed", "conversation_started":
		userID := ""
		if cb.User != nil {
			userID = strings.TrimSpace(cb.User.ID)
		}
		if userID == "" {
			return out, nil
		}
		out = append(out, InboundEvent{Type: EventTypeFollow, UserID: userID})
		ctx := strings.TrimSpace(cb.Context)
		if ctx != "" {
			out = append(out, InboundEvent{
				Type:    EventTypeMessage,
				UserID:  userID,
				Text:    ctx,
				Context: ctx,
			})
		}
	case "unsubscribed":
		userID := ""
		if cb.User != nil {
			userID = strings.TrimSpace(cb.User.ID)
		}
		if userID != "" {
			out = append(out, InboundEvent{Type: EventTypeUnfollow, UserID: userID})
		}
	case "message":
		userID := ""
		if cb.Sender != nil {
			userID = strings.TrimSpace(cb.Sender.ID)
		}
		text := ""
		if cb.Message != nil {
			text = strings.TrimSpace(cb.Message.Text)
		}
		if userID == "" || text == "" {
			return out, nil
		}
		out = append(out, InboundEvent{
			Type:      EventTypeMessage,
			UserID:    userID,
			MessageID: strconv.FormatInt(cb.MessageToken, 10),
			Text:      text,
		})
	}
	return out, nil
}

func (c *viberClientImpl) ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error {
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

var uriPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{2,50}$`)

// BotDeepLink builds viber://pa?chatURI=<uri>&context=<payload>
func BotDeepLink(uri, contextPayload string) (string, error) {
	normalized, err := NormalizeURI(uri)
	if err != nil {
		return "", err
	}
	q := url.Values{}
	q.Set("chatURI", normalized)
	contextPayload = strings.TrimSpace(contextPayload)
	if contextPayload != "" {
		q.Set("context", contextPayload)
	}
	return "viber://pa?" + q.Encode(), nil
}

// NormalizeURI validates the public account URI (chatURI).
func NormalizeURI(uri string) (string, error) {
	trimmed := strings.TrimSpace(uri)
	trimmed = strings.TrimPrefix(trimmed, "@")
	if trimmed == "" {
		return "", errors.New("uri query parameter is required")
	}
	if !uriPattern.MatchString(trimmed) {
		return "", errors.New("uri must be 2–50 chars: letters, digits, ., _, -")
	}
	return trimmed, nil
}
