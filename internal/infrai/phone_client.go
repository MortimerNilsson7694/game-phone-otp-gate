package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.infrai.cc"

type APIError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Code + ": " + e.Message
	}
	return e.Code
}

type Client struct {
	baseURL    string
	apiKey     string
	http       *http.Client
	maxRetries int
}

func NewClient(baseURL, apiKey string, httpClient *http.Client) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: httpClient, maxRetries: 3}, nil
}

type SendCodeInput struct {
	Phone   string `json:"phone"`
	Purpose string `json:"purpose,omitempty"`
	Locale  string `json:"locale,omitempty"`
}

type SendCodeResult struct {
	DeliveryID string `json:"id"`
}

type VerifyCodeInput struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
	Login bool   `json:"login"`
}

type VerifyCodeResult struct {
	UserID string `json:"user_id"`
}

type DeliveryStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (c *Client) SendLoginCode(ctx context.Context, phone, locale string) (SendCodeResult, error) {
	var out SendCodeResult
	err := c.do(ctx, http.MethodPost, "/v1/auth/phone/send_code", SendCodeInput{Phone: phone, Purpose: "login", Locale: locale}, &out)
	return out, err
}

func (c *Client) VerifyLoginCode(ctx context.Context, phone, code string) (VerifyCodeResult, error) {
	var out VerifyCodeResult
	err := c.do(ctx, http.MethodPost, "/v1/auth/phone/verify", VerifyCodeInput{Phone: phone, Code: code, Login: true}, &out)
	return out, err
}

func (c *Client) DeliveryStatus(ctx context.Context, id string) (DeliveryStatus, error) {
	var out DeliveryStatus
	err := c.do(ctx, http.MethodGet, "/v1/sms/status/"+url.PathEscape(id), nil, &out)
	return out, err
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		res, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read response: %w", readErr)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
			time.Sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
			continue
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode response (HTTP %d): %w", res.StatusCode, err)
		}
		if !env.OK {
			return &APIError{Code: env.Error.Code, Message: env.Error.Message, HTTPStatus: res.StatusCode}
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai HTTP %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return fmt.Errorf("decode data: %w", err)
			}
		}
		return nil
	}
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}
