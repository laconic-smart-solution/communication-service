package client

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrRateLimited      = errors.New("wechat api rate limited")
	ErrInvalidToken     = errors.New("invalid wechat access token")
)

type SendMessageInput struct {
	AccessToken string
	RecipientID string // OpenID
	Text        string
}

type CreateTempQRInput struct {
	AccessToken string
	SceneStr    string
	ExpireSec   int
}

type CreateTempQRResult struct {
	Ticket        string
	ExpireSeconds int
	URL           string
	ShowQRCodeURL string
}

const (
	EventTypeMessage  = "message"
	EventTypeFollow   = "follow"
	EventTypeUnfollow = "unfollow"
	EventTypeScan     = "scan"
)

type InboundEvent struct {
	Type      string `json:"type"`
	UserID    string `json:"user_id"` // OpenID
	MessageID string `json:"message_id,omitempty"`
	Text      string `json:"text,omitempty"`
	EventKey  string `json:"event_key,omitempty"` // scene_str for SCAN / subscribe
}

type WeChatClient interface {
	GetAccessToken(ctx context.Context, appID, appSecret string) (string, int, error)
	SendMessage(ctx context.Context, in SendMessageInput) error
	CreateTempQR(ctx context.Context, in CreateTempQRInput) (*CreateTempQRResult, error)
	FetchQRCodePNG(ctx context.Context, ticket string) ([]byte, error)
	VerifyWebhook(signature, timestamp, nonce, token string) error
	ParseWebhookXML(body []byte) ([]InboundEvent, error)
	ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error
}

type wechatClientImpl struct {
	httpClient *http.Client
}

var _ WeChatClient = (*wechatClientImpl)(nil)

func NewWeChatClient(timeoutSec int) WeChatClient {
	if timeoutSec < 1 {
		timeoutSec = 30
	}
	return &wechatClientImpl{
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *wechatClientImpl) GetAccessToken(ctx context.Context, appID, appSecret string) (string, int, error) {
	q := url.Values{}
	q.Set("grant_type", "client_credential")
	q.Set("appid", appID)
	q.Set("secret", appSecret)
	apiURL := "https://api.weixin.qq.com/cgi-bin/token?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", 0, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("wechat token request: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&out); err != nil {
		return "", 0, fmt.Errorf("decode token response: %w", err)
	}
	if out.ErrCode != 0 || out.AccessToken == "" {
		return "", 0, fmt.Errorf("wechat token error %d: %s", out.ErrCode, out.ErrMsg)
	}
	return out.AccessToken, out.ExpiresIn, nil
}

func (c *wechatClientImpl) SendMessage(ctx context.Context, in SendMessageInput) error {
	payload := map[string]interface{}{
		"touser":  in.RecipientID,
		"msgtype": "text",
		"text":    map[string]string{"content": in.Text},
	}
	apiURL := "https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=" + url.QueryEscape(in.AccessToken)
	return c.postJSON(ctx, apiURL, payload)
}

func (c *wechatClientImpl) CreateTempQR(ctx context.Context, in CreateTempQRInput) (*CreateTempQRResult, error) {
	expire := in.ExpireSec
	if expire <= 0 {
		expire = 600
	}
	if expire > 2592000 {
		expire = 2592000
	}
	payload := map[string]interface{}{
		"expire_seconds": expire,
		"action_name":    "QR_STR_SCENE",
		"action_info": map[string]interface{}{
			"scene": map[string]string{"scene_str": in.SceneStr},
		},
	}
	apiURL := "https://api.weixin.qq.com/cgi-bin/qrcode/create?access_token=" + url.QueryEscape(in.AccessToken)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wechat qr create: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Ticket        string `json:"ticket"`
		ExpireSeconds int    `json:"expire_seconds"`
		URL           string `json:"url"`
		ErrCode       int    `json:"errcode"`
		ErrMsg        string `json:"errmsg"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&out); err != nil {
		return nil, err
	}
	if out.ErrCode != 0 || out.Ticket == "" {
		return nil, fmt.Errorf("wechat qr create error %d: %s", out.ErrCode, out.ErrMsg)
	}
	return &CreateTempQRResult{
		Ticket:        out.Ticket,
		ExpireSeconds: out.ExpireSeconds,
		URL:           out.URL,
		ShowQRCodeURL: "https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=" + url.QueryEscape(out.Ticket),
	}, nil
}

func (c *wechatClientImpl) FetchQRCodePNG(ctx context.Context, ticket string) ([]byte, error) {
	apiURL := "https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=" + url.QueryEscape(ticket)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch qr png: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("showqrcode status %d: %s", resp.StatusCode, string(b))
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

func (c *wechatClientImpl) postJSON(ctx context.Context, apiURL string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("wechat api request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.ErrCode == 40001 || out.ErrCode == 42001 {
		return ErrInvalidToken
	}
	if out.ErrCode == 45009 || out.ErrCode == 45011 {
		return ErrRateLimited
	}
	if out.ErrCode != 0 {
		return fmt.Errorf("wechat api error %d: %s", out.ErrCode, out.ErrMsg)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("wechat api status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (c *wechatClientImpl) VerifyWebhook(signature, timestamp, nonce, token string) error {
	parts := []string{token, timestamp, nonce}
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	want := hex.EncodeToString(sum[:])
	if !strings.EqualFold(strings.TrimSpace(signature), want) {
		return ErrInvalidSignature
	}
	return nil
}

type xmlMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgID        int64    `xml:"MsgId"`
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
	Ticket       string   `xml:"Ticket"`
}

func (c *wechatClientImpl) ParseWebhookXML(body []byte) ([]InboundEvent, error) {
	var msg xmlMsg
	if err := xml.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("parse wechat xml: %w", err)
	}
	openID := strings.TrimSpace(msg.FromUserName)
	if openID == "" {
		return nil, errors.New("missing FromUserName")
	}

	mt := strings.ToLower(strings.TrimSpace(msg.MsgType))
	switch mt {
	case "text":
		return []InboundEvent{{
			Type:      EventTypeMessage,
			UserID:    openID,
			MessageID: fmt.Sprintf("%d", msg.MsgID),
			Text:      msg.Content,
		}}, nil
	case "event":
		ev := strings.ToLower(strings.TrimSpace(msg.Event))
		key := strings.TrimSpace(msg.EventKey)
		switch ev {
		case "subscribe":
			// qrscene_<scene_str> when scan-to-subscribe
			if strings.HasPrefix(key, "qrscene_") {
				key = strings.TrimPrefix(key, "qrscene_")
			}
			return []InboundEvent{{
				Type:     EventTypeFollow,
				UserID:   openID,
				EventKey: key,
			}}, nil
		case "unsubscribe":
			return []InboundEvent{{
				Type:   EventTypeUnfollow,
				UserID: openID,
			}}, nil
		case "scan":
			return []InboundEvent{{
				Type:     EventTypeScan,
				UserID:   openID,
				EventKey: key,
			}}, nil
		default:
			return nil, nil
		}
	default:
		return nil, nil
	}
}

func (c *wechatClientImpl) ForwardEvents(ctx context.Context, callbackURL string, events []InboundEvent) error {
	payload, err := json.Marshal(map[string]interface{}{"events": events})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("callback status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
