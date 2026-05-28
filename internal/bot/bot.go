package bot

import (
	"context"
	"net/http"
	"net/url"
)

// Bot posts Adaptive Cards to Teams webhook URLs.
type Bot struct {
	webProxyURL *url.URL
	logoURL     string
	httpClient  *http.Client
}

// New constructs a Bot. webProxyAddr may be empty (cards omit the action button).
func New(webProxyAddr, logoURL string) *Bot {
	var proxyURL *url.URL
	if webProxyAddr != "" {
		proxyURL = &url.URL{Scheme: "https", Host: webProxyAddr}
	}
	return &Bot{
		webProxyURL: proxyURL,
		logoURL:     logoURL,
		httpClient:  &http.Client{Timeout: requestTimeout},
	}
}

// Post sends an Adaptive Card for reqID/data to webhookURL.
func (b *Bot) Post(ctx context.Context, webhookURL string, reqID string, data RequestData) error {
	card := BuildCard(reqID, data, b.webProxyURL, b.logoURL)
	return postCard(ctx, b.httpClient, webhookURL, card)
}
