package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gravitational/trace"
)

const (
	maxRetries      = 3
	retryBaseDelay  = time.Second
	requestTimeout  = 15 * time.Second
)

// postCard marshals msg and POSTs it to webhookURL with retry on 5xx.
func postCard(ctx context.Context, client *http.Client, webhookURL string, msg teamsMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return trace.Wrap(err)
	}

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return trace.Wrap(ctx.Err())
			case <-time.After(retryBaseDelay * time.Duration(attempt)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
		if err != nil {
			return trace.Wrap(err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = trace.Wrap(err)
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return trace.Errorf("Teams webhook returned %d for %s — check the webhook URL", resp.StatusCode, webhookURL)
		}
		// 5xx — retry
		lastErr = trace.Errorf("Teams webhook returned %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxRetries)
	}

	return fmt.Errorf("Teams webhook failed after %d attempts: %w", maxRetries, lastErr)
}
