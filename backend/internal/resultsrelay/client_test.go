package resultsrelay

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientReceivesWithoutAckAndExplicitlyAcknowledges(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"messages":[{"messageId":"id","receiptHandle":"receipt","body":"{}"}]}`))
			return
		}
		if r.URL.Path != "/results/ack" {
			t.Errorf("ack path: %s", r.URL.Path)
		}
		var payload struct {
			Messages []Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if len(payload.Messages) != 1 || payload.Messages[0].ReceiptHandle != "receipt" || payload.Messages[0].Body != "" {
			t.Errorf("ack must contain only receipt identity: %+v", payload)
		}
		_, _ = w.Write([]byte(`{"acknowledged":["id"],"failed":[]}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL + "/results")
	if err != nil {
		t.Fatal(err)
	}
	messages, err := client.ReceiveMessages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(methods) != 1 || messages[0].MessageID != "id" {
		t.Fatalf("receive: %+v, %v", messages, methods)
	}
	if err := client.AcknowledgeMessages(context.Background(), messages); err != nil {
		t.Fatal(err)
	}
}

func TestClientRejectsUnboundedOrInvalidResponses(t *testing.T) {
	for _, body := range []string{`{"messages":[],"padding":"` + strings.Repeat("x", 2*1024*1024) + `"}`, `{"messages":[]} trailing`, `{}`, `{"messages":null}`,
		`{"messages":[` + strings.TrimSuffix(strings.Repeat(`{"body":"{}"},`, 11), ",") + `]}`} {
		t.Run("invalid response", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			client, err := NewClient(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.ReceiveMessages(context.Background()); err == nil {
				t.Fatal("expected receive error")
			}
		})
	}
}

func TestClientRequiresCompleteAcknowledgement(t *testing.T) {
	for _, response := range []struct {
		status int
		body   string
	}{
		{503, `{"acknowledged":["id"],"failed":["other"]}`},
		{200, `{"acknowledged":[],"failed":[]}`}, {200, `{"acknowledged":["wrong"],"failed":[]}`},
		{200, `{"acknowledged":["id"],"failed":["id"]}`}, {200, `{}`},
	} {
		t.Run("invalid ack", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(response.status)
				_, _ = w.Write([]byte(response.body))
			}))
			defer server.Close()
			client, err := NewClient(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			if err := client.AcknowledgeMessages(context.Background(), []Message{{MessageID: "id", ReceiptHandle: "receipt"}}); err == nil {
				t.Fatal("expected ack error")
			}
		})
	}
}

func TestClientRejectsInvalidEndpointsAndBoundsTimeout(t *testing.T) {
	for _, endpoint := range []string{"", "relative", "ftp://example.com", "http://user:secret@example.com", "https://example.com/results?token=secret", "https://example.com/#fragment"} {
		if _, err := NewClient(endpoint); err == nil {
			t.Errorf("accepted invalid endpoint")
		}
	}
	client, err := NewClient("https://example.com/results")
	if err != nil {
		t.Fatal(err)
	}
	if client.(*httpClient).client.Timeout <= 0 {
		t.Fatal("HTTP requests must have a deadline")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failingResponseBody struct{ io.Reader }

func (f failingResponseBody) Close() error { return errors.New("close failed") }

func TestClientSignalsResponseCloseFailure(t *testing.T) {
	client, err := NewClient("https://example.test/results")
	if err != nil {
		t.Fatal(err)
	}
	client.(*httpClient).client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: failingResponseBody{strings.NewReader(`{"messages":[]}`)}}, nil
	})
	if _, err := client.ReceiveMessages(context.Background()); err == nil {
		t.Fatal("response close failure must surface")
	}
}

func TestClientHonorsCancellationAndRejectsRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/results" {
			t.Error("followed redirect")
		}
		http.Redirect(w, r, "/other", http.StatusFound)
	}))
	defer server.Close()
	client, err := NewClient(server.URL + "/results")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ReceiveMessages(context.Background()); err == nil {
		t.Fatal("accepted redirect")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.ReceiveMessages(ctx); err == nil {
		t.Fatal("ignored cancellation")
	}
}
