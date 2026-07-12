package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const cfAPIBase = "https://api.cloudflare.com/client/v4"

// queueClient talks to the Cloudflare Queues HTTP pull-consumer API.
type queueClient struct {
	http      *http.Client
	apiToken  string
	accountID string
	queueID   string
}

type pullResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Result struct {
		Messages []queueMessage `json:"messages"`
	} `json:"result"`
}

type queueMessage struct {
	Body     string `json:"body"` // base64-encoded for json/bytes content types
	ID       string `json:"id"`
	LeaseID  string `json:"lease_id"`
	Attempts int    `json:"attempts"`
}

func (c *queueClient) pull(ctx context.Context, batchSize int, visibilityTimeoutMS int) ([]queueMessage, error) {
	body, _ := json.Marshal(map[string]int{
		"batch_size":            batchSize,
		"visibility_timeout_ms": visibilityTimeoutMS,
	})

	url := fmt.Sprintf("%s/accounts/%s/queues/%s/messages/pull", cfAPIBase, c.accountID, c.queueID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pull request: %w", err)
	}
	defer resp.Body.Close()

	var out pullResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode pull response: %w", err)
	}
	if !out.Success {
		return nil, fmt.Errorf("queue pull failed: %+v", out.Errors)
	}
	return out.Result.Messages, nil
}

func (c *queueClient) ack(ctx context.Context, leaseIDs []string) error {
	if len(leaseIDs) == 0 {
		return nil
	}

	type ackEntry struct {
		LeaseID string `json:"lease_id"`
	}
	acks := make([]ackEntry, len(leaseIDs))
	for i, id := range leaseIDs {
		acks[i] = ackEntry{LeaseID: id}
	}

	payload, _ := json.Marshal(map[string]any{
		"acks":    acks,
		"retries": []ackEntry{},
	})

	url := fmt.Sprintf("%s/accounts/%s/queues/%s/messages/ack", cfAPIBase, c.accountID, c.queueID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ack request: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode ack response: %w", err)
	}
	if !out.Success {
		return fmt.Errorf("queue ack failed: %+v", out.Errors)
	}
	return nil
}
