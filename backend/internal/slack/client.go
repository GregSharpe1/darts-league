package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultAPIURL = "https://slack.com/api/chat.postMessage"

var ErrDisabled = errors.New("slack client is disabled")

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	token      string
	apiURL     string
	httpClient HTTPClient
	maxRetries int
}

type postMessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

type postMessageResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func NewClient(token string) *Client {
	return &Client{
		token:      strings.TrimSpace(token),
		apiURL:     defaultAPIURL,
		httpClient: http.DefaultClient,
		maxRetries: 2,
	}
}

func NewClientForTesting(token, apiURL string, httpClient HTTPClient, maxRetries int) *Client {
	if maxRetries < 0 {
		maxRetries = 0
	}

	return &Client{
		token:      strings.TrimSpace(token),
		apiURL:     apiURL,
		httpClient: httpClient,
		maxRetries: maxRetries,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.token != ""
}

func (c *Client) PostMessage(ctx context.Context, channelID, text string) error {
	if !c.Enabled() {
		return ErrDisabled
	}
	if strings.TrimSpace(channelID) == "" {
		return errors.New("slack channel id is required")
	}

	payload, err := json.Marshal(postMessageRequest{
		Channel: channelID,
		Text:    text,
	})
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		lastErr = c.postOnce(ctx, payload)
		if lastErr == nil {
			return nil
		}
		if !isRetryable(lastErr) {
			return lastErr
		}
	}

	return lastErr
}

func (c *Client) postOnce(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return retryableError{err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return retryableError{err: err}
	}

	if resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests {
		return retryableError{err: fmt.Errorf("slack returned status %d", resp.StatusCode)}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded postMessageResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return fmt.Errorf("decode slack response: %w", err)
	}
	if !decoded.OK {
		return fmt.Errorf("slack api error: %s", decoded.Error)
	}

	return nil
}

type retryableError struct {
	err error
}

func (e retryableError) Error() string {
	return e.err.Error()
}

func (e retryableError) Unwrap() error {
	return e.err
}

func isRetryable(err error) bool {
	var retryErr retryableError
	return errors.As(err, &retryErr)
}
