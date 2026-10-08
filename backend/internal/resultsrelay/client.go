// Package resultsrelay polls the results relay for match results reported by
// external scoring tools and hands each message to the league package for
// storage as a pending result awaiting admin confirmation.
package resultsrelay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseBytes = 2 * 1024 * 1024

var ErrInvalidDelivery = errors.New("invalid relay delivery")

// Message is a single result payload received from the results relay.
type Message struct {
	Body          string `json:"body,omitempty"`
	MessageID     string `json:"messageId"`
	ReceiptHandle string `json:"receiptHandle"`
	Rejection     string `json:"rejection,omitempty"`
}

// Client wraps the results relay HTTP surface so it can be faked in tests.
type Client interface {
	ReceiveMessages(ctx context.Context) ([]Message, error)
	AcknowledgeMessages(ctx context.Context, messages []Message) error
}

type httpClient struct {
	endpoint string
	client   *http.Client
}

type httpResponse struct {
	Messages []Message `json:"messages"`
}

// NewClient receives messages without deleting them. Acknowledgement is explicit.
func NewClient(endpoint string) (Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("results endpoint must be an HTTP(S) URL without credentials, query or fragment")
	}
	return &httpClient{endpoint: strings.TrimRight(endpoint, "/"), client: &http.Client{
		Timeout:       12 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (c *httpClient) ReceiveMessages(ctx context.Context) ([]Message, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.do(request, maxResponseBytes)
	if err != nil {
		return nil, err
	}
	var payload httpResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, ErrInvalidDelivery
	}
	if payload.Messages == nil || len(payload.Messages) > 10 {
		return nil, ErrInvalidDelivery
	}
	return payload.Messages, nil
}

func (c *httpClient) AcknowledgeMessages(ctx context.Context, messages []Message) error {
	if len(messages) < 1 || len(messages) > 10 {
		return ErrInvalidDelivery
	}
	entries := make([]Message, 0, len(messages))
	ids, receipts := map[string]bool{}, map[string]bool{}
	for _, message := range messages {
		if !message.validReceipt() || ids[message.MessageID] || receipts[message.ReceiptHandle] {
			return ErrInvalidDelivery
		}
		ids[message.MessageID], receipts[message.ReceiptHandle] = true, true
		entries = append(entries, Message{MessageID: message.MessageID, ReceiptHandle: message.ReceiptHandle})
	}
	body, err := json.Marshal(httpResponse{Messages: entries})
	if err != nil {
		return err
	}
	if len(body) > 32*1024 {
		return ErrInvalidDelivery
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/ack", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	body, err = c.do(request, 32*1024)
	if err != nil {
		return err
	}
	var response struct {
		Acknowledged []string `json:"acknowledged"`
		Failed       []string `json:"failed"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return ErrInvalidDelivery
	}
	if len(response.Failed) != 0 || len(response.Acknowledged) != len(entries) {
		return ErrInvalidDelivery
	}
	for _, id := range response.Acknowledged {
		if !ids[id] {
			return ErrInvalidDelivery
		}
		delete(ids, id)
	}
	return nil
}

func (m Message) validReceipt() bool {
	return strings.TrimSpace(m.MessageID) != "" && len(m.MessageID) <= 128 &&
		strings.TrimSpace(m.ReceiptHandle) != "" && len(m.ReceiptHandle) <= 2048
}

func (c *httpClient) do(request *http.Request, limit int64) (result []byte, resultErr error) {
	response, err := c.client.Do(request)
	if err != nil {
		return nil, errors.New("relay HTTP request failed")
	}
	defer func() {
		if err := response.Body.Close(); err != nil && resultErr == nil {
			resultErr = errors.New("relay HTTP response close failed")
		}
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("relay HTTP status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, errors.New("relay HTTP response read failed")
	}
	if int64(len(body)) > limit {
		return nil, ErrInvalidDelivery
	}
	return body, nil
}
