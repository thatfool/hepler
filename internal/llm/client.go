// Package llm is a minimal streaming client for OpenAI-compatible chat endpoints.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client talks to an OpenAI-compatible /chat/completions endpoint.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// Message is a single chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Stream sends messages and returns the full assistant reply. If onToken is
// non-nil it is called with each content delta as it arrives.
func (c *Client) Stream(ctx context.Context, messages []Message, onToken func(string)) (string, error) {
	body, err := json.Marshal(chatRequest{Model: c.Model, Messages: messages, Stream: true})
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return "", fmt.Errorf("endpoint returned %s: %s", resp.Status, msg)
	}

	var out strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // tolerate keep-alives and partial frames
		}
		if chunk.Error != nil {
			return out.String(), fmt.Errorf("endpoint error: %s", chunk.Error.Message)
		}
		for _, ch := range chunk.Choices {
			if ch.Delta.Content == "" {
				continue
			}
			out.WriteString(ch.Delta.Content)
			if onToken != nil {
				onToken(ch.Delta.Content)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}
