// Package resultsrelay polls the results relay for match results reported by
// external scoring tools and hands each message to the league package for
// storage as a pending result awaiting admin confirmation.
package resultsrelay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Message is a single result payload received from the results relay.
type Message struct {
	Body string
}

// Client wraps the results relay HTTP surface so it can be faked in tests.
type Client interface {
	ReceiveMessages(ctx context.Context) ([]Message, error)
}

type httpClient struct {
	endpoint string
	client   *http.Client
}

type httpResponse struct {
	Messages []struct {
		Body string `json:"body"`
	} `json:"messages"`
}

// NewClient reads and drains messages through the public relay endpoint.
func NewClient(endpoint string) (Client, error) {
	if endpoint == "" {
		return nil, errors.New("results endpoint is required")
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
		messages = append(messages, Message{Body: message.Body})
	}
	return messages, nil
}
