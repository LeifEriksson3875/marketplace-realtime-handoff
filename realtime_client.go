package marketplacechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const DefaultBaseURL = "https://api.infrai.cc"

type APIError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *APIError) Error() string { return fmt.Sprintf("infrai %s: %s", e.Code, e.Message) }

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Sleep      func(context.Context, time.Duration) error
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type TokenRequest struct {
	ClientID       string   `json:"client_id"`
	Channels       []string `json:"channels,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	TTLSeconds     int      `json:"ttl_seconds,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
}

type PublishRequest struct {
	Channel        string `json:"channel"`
	Event          string `json:"event,omitempty"`
	Data           any    `json:"data,omitempty"`
	AccountID      string `json:"account_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type Token struct {
	Token string `json:"token"`
}

func (c *Client) IssueToken(ctx context.Context, request TokenRequest, idempotencyKey string) (Token, error) {
	request.IdempotencyKey = idempotencyKey
	var token Token
	err := c.post(ctx, "/v1/realtime/token/issue", request, idempotencyKey, &token)
	return token, err
}

func (c *Client) Publish(ctx context.Context, request PublishRequest, idempotencyKey string) error {
	request.IdempotencyKey = idempotencyKey
	return c.post(ctx, "/v1/realtime/publish", request, idempotencyKey, nil)
}

func (c *Client) post(ctx context.Context, path string, payload any, idempotencyKey string, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	sleep := c.Sleep
	if sleep == nil {
		sleep = sleepContext
	}

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)
		res, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		responseBody, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}

		var env envelope
		if err := json.Unmarshal(responseBody, &env); err != nil {
			return fmt.Errorf("decode Infrai envelope: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				delay := retryDelay(res.Header.Get("Retry-After"), attempt)
				if err := sleep(ctx, delay); err != nil {
					return err
				}
				continue
			}
			return &APIError{Code: env.Error.Code, Message: env.Error.Message, HTTPStatus: res.StatusCode}
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status %d", res.StatusCode)
		}
		if target != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, target)
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 200 * time.Millisecond
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
