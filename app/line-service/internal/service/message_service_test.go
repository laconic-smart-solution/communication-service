package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/request"
)

type mockLINEClient struct {
	multicastCalls int
	multicastErrs  []error
}

func (m *mockLINEClient) SendMessage(context.Context, client.SendMessageInput) error {
	return nil
}

func (m *mockLINEClient) SendMulticast(_ context.Context, _ client.SendMulticastInput) error {
	m.multicastCalls++
	if len(m.multicastErrs) > 0 {
		err := m.multicastErrs[0]
		m.multicastErrs = m.multicastErrs[1:]
		return err
	}
	return nil
}

func (m *mockLINEClient) VerifyWebhook(string, []byte, string) error { return nil }
func (m *mockLINEClient) ParseWebhook([]byte) ([]client.InboundEvent, error) {
	return nil, nil
}
func (m *mockLINEClient) ForwardEvents(context.Context, string, []client.InboundEvent) error {
	return nil
}

func newTestMessageService(line client.LINEClient, batch BatchConfig) *messageServiceImpl {
	svc := NewMessageService(logrus.New(), line, batch).(*messageServiceImpl)
	svc.sleep = func(context.Context, time.Duration) error { return nil }
	return svc
}

func TestSendBatchChunksByMulticastLimit(t *testing.T) {
	mock := &mockLINEClient{}
	svc := newTestMessageService(mock, BatchConfig{
		MinInterval:   0,
		MaxRecipients: 10000,
		MaxRetries:    3,
		BackoffBase:   time.Millisecond,
	})

	ids := make([]string, 501)
	for i := range ids {
		ids[i] = fmt.Sprintf("U%032d", i)
	}

	res, err := svc.SendBatch(context.Background(), "token", request.SendBatchMessageRequest{
		RecipientIDs: ids,
		Text:         "hello all",
	})
	if err != nil {
		t.Fatalf("SendBatch() error = %v", err)
	}
	if mock.multicastCalls != 2 {
		t.Fatalf("multicastCalls = %d, want 2", mock.multicastCalls)
	}
	if res.Total != 501 || res.Sent != 501 || res.Failed != 0 || res.ChunkCount != 2 {
		t.Fatalf("response = %+v", res)
	}
}

func TestSendBatchRetriesRateLimitThenSucceeds(t *testing.T) {
	mock := &mockLINEClient{
		multicastErrs: []error{client.ErrRateLimited, client.ErrRateLimited, nil},
	}
	svc := newTestMessageService(mock, BatchConfig{
		MinInterval:   0,
		MaxRecipients: 10000,
		MaxRetries:    3,
		BackoffBase:   time.Millisecond,
	})

	res, err := svc.SendBatch(context.Background(), "token", request.SendBatchMessageRequest{
		RecipientIDs: []string{"U1", "U2", "U1", " ", "U3"},
		Text:         "announce",
	})
	if err != nil {
		t.Fatalf("SendBatch() error = %v", err)
	}
	if mock.multicastCalls != 3 {
		t.Fatalf("multicastCalls = %d, want 3 (2 rate-limit + 1 success)", mock.multicastCalls)
	}
	if res.Total != 3 || res.Sent != 3 || res.Failed != 0 {
		t.Fatalf("response = %+v", res)
	}
}

func TestSendBatchRecordsFailedChunkAfterRetries(t *testing.T) {
	mock := &mockLINEClient{
		multicastErrs: []error{
			client.ErrRateLimited,
			client.ErrRateLimited,
			client.ErrRateLimited,
			client.ErrRateLimited,
		},
	}
	svc := newTestMessageService(mock, BatchConfig{
		MinInterval:   0,
		MaxRecipients: 10000,
		MaxRetries:    3,
		BackoffBase:   time.Millisecond,
	})

	res, err := svc.SendBatch(context.Background(), "token", request.SendBatchMessageRequest{
		RecipientIDs: []string{"Ufail1", "Ufail2"},
		Text:         "announce",
	})
	if err != nil {
		t.Fatalf("SendBatch() error = %v", err)
	}
	if res.Sent != 0 || res.Failed != 2 || len(res.FailedRecipientIDs) != 2 {
		t.Fatalf("response = %+v", res)
	}
}

func TestSendBatchRejectsTooManyRecipients(t *testing.T) {
	mock := &mockLINEClient{}
	svc := newTestMessageService(mock, BatchConfig{
		MinInterval:   0,
		MaxRecipients: 2,
		MaxRetries:    1,
		BackoffBase:   time.Millisecond,
	})

	_, err := svc.SendBatch(context.Background(), "token", request.SendBatchMessageRequest{
		RecipientIDs: []string{"U1", "U2", "U3"},
		Text:         "announce",
	})
	if err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("error = %v, want max recipients error", err)
	}
}

func TestSendBatchRejectsInvalidToken(t *testing.T) {
	mock := &mockLINEClient{
		multicastErrs: []error{client.ErrInvalidToken},
	}
	svc := newTestMessageService(mock, BatchConfig{
		MinInterval:   0,
		MaxRecipients: 100,
		MaxRetries:    1,
		BackoffBase:   time.Millisecond,
	})

	_, err := svc.SendBatch(context.Background(), "token", request.SendBatchMessageRequest{
		RecipientIDs: []string{"U1"},
		Text:         "announce",
	})
	if err == nil || err.Error() != "invalid line channel token" {
		t.Fatalf("error = %v, want invalid line channel token", err)
	}
}
