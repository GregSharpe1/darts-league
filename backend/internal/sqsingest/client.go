// Package sqsingest polls the results relay for match results reported by
// external scoring tools and hands each message to the league package for
// storage as a pending result awaiting admin confirmation.
package sqsingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Message is a single result payload received from the queue, along with the
// receipt handle needed to delete it once processed.
type Message struct {
	Body          string
	ReceiptHandle string
}

// Client wraps the AWS SQS API surface the poller needs, so it can be faked
// in tests without exercising real AWS calls.
type Client interface {
	ReceiveMessages(ctx context.Context) ([]Message, error)
	DeleteMessage(ctx context.Context, receiptHandle string) error
}

type httpClient struct {
	endpoint string
	client   *http.Client
}

type httpResponse struct {
	Messages []struct {
		Body          string `json:"body"`
		MessageID     string `json:"messageId"`
		ReceiptHandle string `json:"receiptHandle"`
	} `json:"messages"`
}

// NewHTTPClient reads and drains messages through the public relay endpoint.
// The relay deletes messages as part of serving the response, so DeleteMessage
// is intentionally a no-op for this client.
func NewHTTPClient(endpoint string) (Client, error) {
	if endpoint == "" {
		return nil, errors.New("sqs results endpoint is required")
	}
	return &httpClient{endpoint: endpoint, client: http.DefaultClient}, nil
}

func (c *httpClient) ReceiveMessages(ctx context.Context) ([]Message, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, err
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("results endpoint returned %s", response.Status)
	}

	var payload httpResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(payload.Messages))
	for _, message := range payload.Messages {
		messages = append(messages, Message{Body: message.Body, ReceiptHandle: message.ReceiptHandle})
	}
	return messages, nil
}

func (c *httpClient) DeleteMessage(context.Context, string) error {
	return nil
}
