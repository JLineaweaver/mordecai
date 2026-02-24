package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Delivery sends digests to a Discord channel via webhook.
type Delivery struct {
	webhookURL string
}

func New(webhookURL string) *Delivery {
	return &Delivery{webhookURL: webhookURL}
}

func (d *Delivery) Name() string {
	return "discord"
}

// Discord webhook message payload.
type webhookPayload struct {
	Content string `json:"content"`
	Flags   int    `json:"flags,omitempty"`
}

// flagSuppressEmbeds tells Discord not to generate link preview embeds.
const flagSuppressEmbeds = 4

// maxMessageLen is Discord's max message length.
const maxMessageLen = 2000

func (d *Delivery) Send(ctx context.Context, digest string) error {
	// Discord has a 2000 char limit — split into chunks if needed.
	chunks := splitMessage(digest, maxMessageLen)

	for _, chunk := range chunks {
		if err := d.sendChunk(ctx, chunk); err != nil {
			return err
		}
	}

	return nil
}

func (d *Delivery) sendChunk(ctx context.Context, content string) error {
	payload := webhookPayload{Content: content, Flags: flagSuppressEmbeds}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned HTTP %d", resp.StatusCode)
	}

	return nil
}

// splitMessage breaks a message into chunks that fit within Discord's limit,
// splitting at newline boundaries where possible.
func splitMessage(s string, maxLen int) []string {
	if len(s) <= maxLen {
		return []string{s}
	}

	var chunks []string
	for len(s) > 0 {
		if len(s) <= maxLen {
			chunks = append(chunks, s)
			break
		}

		// Find last newline within the limit.
		cutPoint := maxLen
		for i := maxLen - 1; i > 0; i-- {
			if s[i] == '\n' {
				cutPoint = i + 1
				break
			}
		}

		chunks = append(chunks, s[:cutPoint])
		s = s[cutPoint:]
	}

	return chunks
}
