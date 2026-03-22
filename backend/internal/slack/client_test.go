package slack

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPostMessageSendsSlackAPIRequest(t *testing.T) {
	t.Parallel()

	httpClient := &stubHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}},
	}
	client := NewClientForTesting("xoxb-test", defaultAPIURL, httpClient, 0)

	if err := client.PostMessage(context.Background(), "C123", "Weekly fixtures"); err != nil {
		t.Fatalf("expected post to succeed, got %v", err)
	}

	if len(httpClient.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(httpClient.requests))
	}

	req := httpClient.requests[0]
	if got := req.header.Get("Authorization"); got != "Bearer xoxb-test" {
		t.Fatalf("expected bearer auth header, got %q", got)
	}

	if !strings.Contains(req.body, `"channel":"C123"`) {
		t.Fatalf("expected request body to include channel, got %s", req.body)
	}
	if !strings.Contains(req.body, `"text":"Weekly fixtures"`) {
		t.Fatalf("expected request body to include text, got %s", req.body)
	}
}

func TestPostMessageRetriesRetryableFailures(t *testing.T) {
	t.Parallel()

	httpClient := &stubHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(strings.NewReader("busy")),
		}, {
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}},
	}
	client := NewClientForTesting("xoxb-test", defaultAPIURL, httpClient, 1)

	if err := client.PostMessage(context.Background(), "C123", "Fixtures"); err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}

	if len(httpClient.requests) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(httpClient.requests))
	}
}

func TestPostMessageReturnsSlackAPIErrors(t *testing.T) {
	t.Parallel()

	httpClient := &stubHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"error":"channel_not_found"}`)),
		}},
	}
	client := NewClientForTesting("xoxb-test", defaultAPIURL, httpClient, 0)

	err := client.PostMessage(context.Background(), "C123", "Fixtures")
	if err == nil {
		t.Fatal("expected slack api error")
	}
	if !strings.Contains(err.Error(), "channel_not_found") {
		t.Fatalf("expected slack error in message, got %v", err)
	}
}

func TestPostMessageReturnsDisabledWhenTokenMissing(t *testing.T) {
	t.Parallel()

	client := NewClientForTesting("", defaultAPIURL, &stubHTTPClient{}, 0)
	err := client.PostMessage(context.Background(), "C123", "Fixtures")
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected disabled error, got %v", err)
	}
}

type recordedRequest struct {
	method string
	url    string
	body   string
	header http.Header
}

type stubHTTPClient struct {
	responses []*http.Response
	errors    []error
	requests  []recordedRequest
	index     int
}

func (c *stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	c.requests = append(c.requests, recordedRequest{
		method: req.Method,
		url:    req.URL.String(),
		body:   string(body),
		header: req.Header.Clone(),
	})

	if c.index < len(c.errors) && c.errors[c.index] != nil {
		err := c.errors[c.index]
		c.index++
		return nil, err
	}
	if c.index < len(c.responses) {
		resp := c.responses[c.index]
		c.index++
		return resp, nil
	}

	return nil, fmt.Errorf("unexpected request %d", c.index+1)
}
