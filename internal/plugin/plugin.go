package plugin

import (
	"context"
	"log/slog"
	"time"

	"github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/trace"

	"github.com/jsabo/teleport-msteams-webhook/internal/bot"
)

const (
	initialBackoff = time.Second
	maxBackoff     = 60 * time.Second
)

// Plugin watches Teleport access request events and posts Teams cards.
type Plugin struct {
	client     *client.Client
	bot        *bot.Bot
	recipients map[string][]string
}

// New creates a Plugin.
func New(clt *client.Client, b *bot.Bot, recipients map[string][]string) *Plugin {
	return &Plugin{client: clt, bot: b, recipients: recipients}
}

// Run watches access_request events, reconnecting with exponential backoff on error.
// Blocks until ctx is cancelled.
func (p *Plugin) Run(ctx context.Context) error {
	backoff := initialBackoff
	for {
		err := p.watch(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			slog.ErrorContext(ctx, "Watcher error, reconnecting", "error", err, "backoff", backoff)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
			backoff = min(backoff*2, maxBackoff)
		}
	}
}

func (p *Plugin) watch(ctx context.Context) error {
	watcher, err := p.client.NewWatcher(ctx, types.Watch{
		Kinds: []types.WatchKind{{Kind: types.KindAccessRequest}},
	})
	if err != nil {
		return trace.Wrap(err)
	}
	defer watcher.Close()

	// Wait for the OpInit event before processing requests.
	select {
	case <-ctx.Done():
		return nil
	case event := <-watcher.Events():
		if event.Type != types.OpInit {
			return trace.Errorf("expected OpInit, got %v", event.Type)
		}
		slog.InfoContext(ctx, "Watcher connected, watching access requests")
	case <-watcher.Done():
		return trace.Wrap(watcher.Error())
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-watcher.Events():
			if err := p.handleEvent(ctx, event); err != nil {
				slog.ErrorContext(ctx, "Error handling event", "error", err)
			}
		case <-watcher.Done():
			return trace.Wrap(watcher.Error())
		}
	}
}

func (p *Plugin) handleEvent(ctx context.Context, event types.Event) error {
	if event.Resource == nil || event.Resource.GetKind() != types.KindAccessRequest {
		return nil
	}

	req, ok := event.Resource.(types.AccessRequest)
	if !ok {
		return nil
	}

	reqID := req.GetName()
	data := bot.RequestData{
		User:          req.GetUser(),
		Roles:         req.GetRoles(),
		RequestReason: req.GetRequestReason(),
	}

	switch {
	case req.GetState().IsPending():
		return p.postToRecipients(ctx, reqID, data, req.GetRoles())

	case req.GetState() == types.RequestState_APPROVED:
		data.ResolutionTag = bot.Approved
		data.ResolutionReason = req.GetResolveReason()
		return p.postToRecipients(ctx, reqID, data, req.GetRoles())

	case req.GetState() == types.RequestState_DENIED:
		data.ResolutionTag = bot.Denied
		data.ResolutionReason = req.GetResolveReason()
		return p.postToRecipients(ctx, reqID, data, req.GetRoles())

	case req.GetState() == types.RequestState_PROMOTED:
		data.ResolutionTag = bot.Promoted
		data.ResolutionReason = req.GetResolveReason()
		return p.postToRecipients(ctx, reqID, data, req.GetRoles())
	}

	return nil
}

func (p *Plugin) postToRecipients(ctx context.Context, reqID string, data bot.RequestData, roles []string) error {
	webhooks := p.resolve(roles)
	if len(webhooks) == 0 {
		slog.WarnContext(ctx, "No webhook recipients configured for roles", "roles", roles)
		return nil
	}

	var errs []error
	for _, url := range webhooks {
		if err := p.bot.Post(ctx, url, reqID, data); err != nil {
			slog.ErrorContext(ctx, "Failed to post Teams card", "url", url, "error", err)
			errs = append(errs, err)
		} else {
			slog.InfoContext(ctx, "Teams card posted", "url", url, "request_id", reqID, "state", string(data.ResolutionTag))
		}
	}
	return trace.NewAggregate(errs...)
}

// resolve returns the webhook URLs for the given roles, using wildcard fallback per role.
func (p *Plugin) resolve(roles []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, role := range roles {
		targets := p.recipients[role]
		if len(targets) == 0 {
			targets = p.recipients["*"]
		}
		for _, t := range targets {
			if !seen[t] {
				seen[t] = true
				result = append(result, t)
			}
		}
	}
	return result
}
