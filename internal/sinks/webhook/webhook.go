package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type WebhookSink struct {
	endpoint  string
	headers   map[string]string
	eventChan chan v1.Event
	client    http.Client
}

func New(endpoint string, headers map[string]string) (*WebhookSink, error) {
	return &WebhookSink{
		endpoint:  endpoint,
		headers:   headers,
		eventChan: make(chan v1.Event, 50),
		client:    http.Client{Timeout: time.Duration(5) * time.Second},
	}, nil
}

func (w *WebhookSink) Start(ctx context.Context) error {
	go w.worker(ctx)
	return nil
}

func (w *WebhookSink) Stop() error {
	w.client.CloseIdleConnections()
	return nil
}

func (w *WebhookSink) Write(ctx context.Context, event v1.Event) error {
	w.eventChan <- event
	return nil
}

func (w *WebhookSink) worker(ctx context.Context) {
	for {
		select {
		case event := <-w.eventChan:
			if err := w.send(ctx, event); err != nil {
				log.Log.Error(err, "failed to write event")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *WebhookSink) send(ctx context.Context, event v1.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range w.headers {
		req.Header.Set(key, value)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to response body: %w", err)
		}
		return fmt.Errorf("request failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
