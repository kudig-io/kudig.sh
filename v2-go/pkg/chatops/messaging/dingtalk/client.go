// Package dingtalk provides DingTalk webhook integration for ChatOps.
package dingtalk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client sends messages to DingTalk via webhook.
type Client struct {
	appKey    string
	appSecret string
	webhook   string
	httpClient *http.Client
}

// webhookRequest is the DingTalk webhook message payload.
type webhookRequest struct {
	MsgType  string       `json:"msgtype"`
	Markdown webhookMarkdown `json:"markdown"`
	Sign     string       `json:"sign,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}

type webhookMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// webhookResponse is the DingTalk webhook API response.
type webhookResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// NewClient creates a new DingTalk webhook client.
func NewClient(appKey, appSecret, webhook string) *Client {
	return &Client{
		appKey:    appKey,
		appSecret: appSecret,
		webhook:   webhook,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendMessage sends a markdown message to DingTalk webhook with HMAC-SHA256 signing.
func (c *Client) SendMessage(title, markdown string) error {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sign, err := c.computeSign(ts)
	if err != nil {
		return fmt.Errorf("failed to compute sign: %w", err)
	}

	payload := webhookRequest{
		MsgType: "markdown",
		Markdown: webhookMarkdown{
			Title: title,
			Text:  markdown,
		},
		Sign:      sign,
		Timestamp: ts,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	webhookURL := c.webhook
	if sign != "" {
		webhookURL = fmt.Sprintf("%s&timestamp=%s&sign=%s",
			c.webhook,
			url.QueryEscape(ts),
			url.QueryEscape(sign),
		)
	}

	resp, err := c.httpClient.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result webhookResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("dingtalk error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// computeSign generates HMAC-SHA256 signature for DingTalk webhook authentication.
func (c *Client) computeSign(timestamp string) (string, error) {
	if c.appSecret == "" {
		return "", nil
	}

	stringToSign := timestamp + "\n" + c.appSecret
	mac := hmac.New(sha256.New, []byte(c.appSecret))
	if _, err := mac.Write([]byte(stringToSign)); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}
