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
	"strings"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrRateLimited      = errors.New("whatsapp api rate limited")
	ErrInvalidToken     = errors.New("invalid whatsapp access token")
)

type SendMessageInput struct {
	AccessToken   string
	PhoneNumberID string
	RecipientID   string // WhatsApp user phone / wa_id (digits)
	Text          string
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
}

type WhatsAppClient interface {
	SendMessage(ctx context.Context, in SendMessageInput) error
	VerifyWebhookSignature(signatureHeader string, body []byte, appSecret string) error
	ParseWebhook(body []byte) ([]InboundEvent, error)
	ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error
}

type whatsAppClientImpl struct {
	httpClient *http.Client
	apiBase    string
	apiVersion string
}

var _ WhatsAppClient = (*whatsAppClientImpl)(nil)

func NewWhatsAppClient(timeoutSec int) WhatsAppClient {
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &whatsAppClientImpl{
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		apiBase:    "https://graph.facebook.com",
		apiVersion: "v21.0",
	}
}

func (c *whatsAppClientImpl) SendMessage(ctx context.Context, in SendMessageInput) error {
	token := strings.TrimSpace(in.AccessToken)
	phoneID := strings.TrimSpace(in.PhoneNumberID)
	to := normalizePhone(in.RecipientID)
	if token == "" {
		return ErrInvalidToken
	}
	if phoneID == "" {
		return errors.New("phone_number_id is required")
	}
	if to == "" {
		return errors.New("recipient_id is required")
	}

	endpoint := fmt.Sprintf("%s/%s/%s/messages", c.apiBase, c.apiVersion, url.PathEscape(phoneID))
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "text",
		"text": map[string]interface{}{
			"preview_url": false,
			"body":        in.Text,
		},
	}
	return c.postJSON(ctx, endpoint, token, payload)
}

func (c *whatsAppClientImpl) postJSON(ctx context.Context, endpoint, accessToken string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal whatsapp payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp api request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))

	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrInvalidToken
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("whatsapp api http %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (c *whatsAppClientImpl) VerifyWebhookSignature(signatureHeader string, body []byte, appSecret string) error {
	appSecret = strings.TrimSpace(appSecret)
	if appSecret == "" {
		return errors.New("whatsapp app_secret is not configured")
	}
	got := strings.TrimSpace(signatureHeader)
	got = strings.TrimPrefix(got, "sha256=")
	got = strings.ToLower(got)

	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

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

func (c *whatsAppClientImpl) ParseWebhook(body []byte) ([]InboundEvent, error) {
	var payload struct {
		Object string `json:"object"`
		Entry  []struct {
			Changes []struct {
				Value struct {
					Messages []struct {
						From string `json:"from"`
						ID   string `json:"id"`
						Type string `json:"type"`
						Text *struct {
							Body string `json:"body"`
						} `json:"text"`
					} `json:"messages"`
					Contacts []struct {
						WaID string `json:"wa_id"`
					} `json:"contacts"`
				} `json:"value"`
				Field string `json:"field"`
			} `json:"changes"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse webhook: %w", err)
	}

	out := make([]InboundEvent, 0)
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "" && change.Field != "messages" {
				continue
			}
			for _, msg := range change.Value.Messages {
				from := normalizePhone(msg.From)
				if from == "" {
					continue
				}
				text := ""
				if msg.Text != nil {
					text = strings.TrimSpace(msg.Text.Body)
				}
				if text == "" {
					continue
				}
				// First inbound message acts as "follow" for connect UX.
				out = append(out, InboundEvent{Type: EventTypeFollow, UserID: from})
				out = append(out, InboundEvent{
					Type:      EventTypeMessage,
					UserID:    from,
					MessageID: strings.TrimSpace(msg.ID),
					Text:      text,
				})
			}
		}
	}
	return out, nil
}

func (c *whatsAppClientImpl) ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error {
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

var phonePattern = regexp.MustCompile(`^\d{8,15}$`)

func normalizePhone(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "+")
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	trimmed = strings.ReplaceAll(trimmed, "-", "")
	if !phonePattern.MatchString(trimmed) {
		return ""
	}
	return trimmed
}

// ChatDeepLink builds https://wa.me/<phone>?text=<payload>
func ChatDeepLink(businessPhone, textPayload string) (string, error) {
	phone, err := NormalizeBusinessPhone(businessPhone)
	if err != nil {
		return "", err
	}
	u := "https://wa.me/" + phone
	textPayload = strings.TrimSpace(textPayload)
	if textPayload != "" {
		u += "?text=" + url.QueryEscape(textPayload)
	}
	return u, nil
}

// NormalizeBusinessPhone validates E.164 digits without +.
func NormalizeBusinessPhone(phone string) (string, error) {
	normalized := normalizePhone(phone)
	if normalized == "" {
		return "", errors.New("phone query parameter is required (E.164 digits, no +)")
	}
	return normalized, nil
}
