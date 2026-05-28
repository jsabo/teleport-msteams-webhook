package config

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gravitational/teleport/api/client"
	"github.com/gravitational/trace"
)

const (
	defaultLogoURL          = "https://raw.githubusercontent.com/jsabo/teleport-msteams-webhook/main/assets/teleport-logo.png"
	defaultRefreshInterval  = time.Minute
)

// TeleportConfig holds connection settings for the Teleport cluster.
type TeleportConfig struct {
	Addr                    string        `toml:"addr"`
	Identity                string        `toml:"identity"`
	RefreshIdentity         bool          `toml:"refresh_identity"`
	RefreshIdentityInterval time.Duration `toml:"refresh_identity_interval"`
}

// MSTeamsConfig holds optional branding settings.
type MSTeamsConfig struct {
	// LogoURL is the image URL displayed at the top of Teams cards.
	// Defaults to the Teleport brand logo hosted in this repo.
	LogoURL string `toml:"logo_url"`
	// DisableLogo omits the logo from cards entirely.
	DisableLogo bool `toml:"disable_logo"`
}

// LogConfig controls log output.
type LogConfig struct {
	Output   string `toml:"output"`
	Severity string `toml:"severity"`
}

// Config is the full plugin configuration parsed from TOML.
type Config struct {
	Teleport   TeleportConfig      `toml:"teleport"`
	MSTeams    MSTeamsConfig       `toml:"msteams"`
	Log        LogConfig           `toml:"log"`
	Recipients map[string][]string `toml:"role_to_recipients"`
}

// LoadConfig reads and validates the TOML config file at path.
func LoadConfig(path string) (*Config, error) {
	var conf Config
	if _, err := toml.DecodeFile(path, &conf); err != nil {
		return nil, trace.Wrap(err)
	}
	if err := conf.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	return &conf, nil
}

// CheckAndSetDefaults validates the config and fills in defaults.
func (c *Config) CheckAndSetDefaults() error {
	if c.Teleport.Addr == "" {
		return trace.BadParameter("missing required value teleport.addr")
	}
	if c.Teleport.Identity == "" {
		return trace.BadParameter("missing required value teleport.identity")
	}
	if c.Teleport.RefreshIdentityInterval == 0 {
		c.Teleport.RefreshIdentityInterval = defaultRefreshInterval
	}

	if c.Log.Output == "" {
		c.Log.Output = "stderr"
	}
	if c.Log.Severity == "" {
		c.Log.Severity = "info"
	}

	if len(c.Recipients) == 0 {
		return trace.BadParameter("missing required value role_to_recipients")
	}
	if len(c.Recipients["*"]) == 0 {
		return trace.BadParameter(`missing required value role_to_recipients["*"]`)
	}

	if err := resolveAndValidateRecipients(c.Recipients); err != nil {
		return trace.Wrap(err)
	}

	if !c.MSTeams.DisableLogo && c.MSTeams.LogoURL == "" {
		c.MSTeams.LogoURL = defaultLogoURL
	}

	return nil
}

// resolveAndValidateRecipients resolves env: and file: prefixes in webhook URLs
// in-place and validates that all resolved values use https.
func resolveAndValidateRecipients(recipients map[string][]string) error {
	for role, urls := range recipients {
		for i, raw := range urls {
			resolved, err := resolveWebhookURL(raw)
			if err != nil {
				return trace.BadParameter("role_to_recipients[%q][%d] %q: %v", role, i, raw, err)
			}
			if !strings.HasPrefix(resolved, "https://") {
				return trace.BadParameter("role_to_recipients[%q][%d] %q: webhook URLs must use https", role, i, raw)
			}
			recipients[role][i] = resolved
		}
	}
	return nil
}

// resolveWebhookURL resolves a single URL value, expanding env: and file: prefixes.
// Plain https:// URLs are returned as-is.
func resolveWebhookURL(s string) (string, error) {
	switch {
	case strings.HasPrefix(s, "env:"):
		name := strings.TrimPrefix(s, "env:")
		val := os.Getenv(name)
		if val == "" {
			return "", trace.Errorf("environment variable %q is not set or empty", name)
		}
		return val, nil
	case strings.HasPrefix(s, "file:"):
		path := strings.TrimPrefix(s, "file:")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", trace.Wrap(err)
		}
		val := strings.TrimSpace(string(data))
		if val == "" {
			return "", trace.Errorf("file %q is empty", path)
		}
		return val, nil
	default:
		return s, nil
	}
}

// EffectiveLogoURL returns the logo URL to use in cards, respecting DisableLogo.
func (c *Config) EffectiveLogoURL() string {
	if c.MSTeams.DisableLogo {
		return ""
	}
	return c.MSTeams.LogoURL
}

// NewClient creates a Teleport API client using identity file credentials.
// If refresh_identity is true, credentials are reloaded automatically when tbot renews them.
func (c *Config) NewClient(ctx context.Context) (*client.Client, error) {
	var creds client.Credentials

	if c.Teleport.RefreshIdentity {
		dynCreds, err := client.NewDynamicIdentityFileCreds(c.Teleport.Identity)
		if err != nil {
			return nil, trace.Wrap(err, "loading identity file")
		}
		go c.watchIdentityFile(ctx, dynCreds)
		creds = dynCreds
	} else {
		creds = client.LoadIdentityFile(c.Teleport.Identity)
	}

	clt, err := client.New(ctx, client.Config{
		Addrs:       []string{c.Teleport.Addr},
		Credentials: []client.Credentials{creds},
	})
	return clt, trace.Wrap(err)
}

// watchIdentityFile reloads dynamic credentials on an interval until ctx is cancelled.
func (c *Config) watchIdentityFile(ctx context.Context, creds *client.DynamicIdentityFileCreds) {
	ticker := time.NewTicker(c.Teleport.RefreshIdentityInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := creds.Reload(); err != nil {
				slog.ErrorContext(ctx, "Failed to reload identity file", "error", err)
			}
		}
	}
}
