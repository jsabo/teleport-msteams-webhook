package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTOML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoadConfig_valid(t *testing.T) {
	path := writeTOML(t, `
[teleport]
addr = "proxy.example.com:443"
identity = "/tmp/identity"

[role_to_recipients]
"*" = ["https://prod.westus2.logic.azure.com/test"]
`)
	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf.Teleport.Addr != "proxy.example.com:443" {
		t.Errorf("addr = %q, want %q", conf.Teleport.Addr, "proxy.example.com:443")
	}
	if conf.MSTeams.LogoURL != defaultLogoURL {
		t.Errorf("logo_url = %q, want default", conf.MSTeams.LogoURL)
	}
}

func TestLoadConfig_missingAddr(t *testing.T) {
	path := writeTOML(t, `
[teleport]
identity = "/tmp/identity"

[role_to_recipients]
"*" = ["https://example.com/hook"]
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing teleport.addr")
	}
}

func TestLoadConfig_missingWildcard(t *testing.T) {
	path := writeTOML(t, `
[teleport]
addr = "proxy.example.com:443"
identity = "/tmp/identity"

[role_to_recipients]
"dev" = ["https://example.com/dev"]
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal(`expected error for missing role_to_recipients["*"]`)
	}
}

func TestLoadConfig_customLogo(t *testing.T) {
	const custom = "https://example.com/logo.png"
	path := writeTOML(t, `
[teleport]
addr = "proxy.example.com:443"
identity = "/tmp/identity"

[msteams]
logo_url = "`+custom+`"

[role_to_recipients]
"*" = ["https://example.com/hook"]
`)
	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf.MSTeams.LogoURL != custom {
		t.Errorf("logo_url = %q, want %q", conf.MSTeams.LogoURL, custom)
	}
}

func TestResolveWebhookURL(t *testing.T) {
	const goodURL = "https://prod.westus2.logic.azure.com/invoke"

	t.Run("plain https passthrough", func(t *testing.T) {
		got, err := resolveWebhookURL(goodURL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != goodURL {
			t.Errorf("got %q, want %q", got, goodURL)
		}
	})

	t.Run("env: set", func(t *testing.T) {
		t.Setenv("TEAMS_WH_TEST", goodURL)
		got, err := resolveWebhookURL("env:TEAMS_WH_TEST")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != goodURL {
			t.Errorf("got %q, want %q", got, goodURL)
		}
	})

	t.Run("env: unset", func(t *testing.T) {
		_, err := resolveWebhookURL("env:TEAMS_WH_DEFINITELY_NOT_SET_XYZ")
		if err == nil {
			t.Fatal("expected error for unset env var")
		}
	})

	t.Run("file: readable", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "webhook.txt")
		if err := os.WriteFile(p, []byte(goodURL+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := resolveWebhookURL("file:" + p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != goodURL {
			t.Errorf("got %q, want %q", got, goodURL)
		}
	})

	t.Run("file: missing", func(t *testing.T) {
		_, err := resolveWebhookURL("file:/nonexistent/path/webhook.txt")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("file: empty", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "empty.txt")
		if err := os.WriteFile(p, []byte("   \n"), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := resolveWebhookURL("file:" + p)
		if err == nil {
			t.Fatal("expected error for empty file")
		}
	})
}

func TestLoadConfig_envRecipient(t *testing.T) {
	const webhookURL = "https://prod.westus2.logic.azure.com/env-test"
	t.Setenv("TEAMS_WH_ENV_TEST", webhookURL)

	path := writeTOML(t, `
[teleport]
addr = "proxy.example.com:443"
identity = "/tmp/identity"

[role_to_recipients]
"*" = ["env:TEAMS_WH_ENV_TEST"]
`)
	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := conf.Recipients["*"][0]
	if got != webhookURL {
		t.Errorf("resolved URL = %q, want %q", got, webhookURL)
	}
}

func TestLoadConfig_fileRecipient(t *testing.T) {
	const webhookURL = "https://prod.westus2.logic.azure.com/file-test"
	p := filepath.Join(t.TempDir(), "webhook.txt")
	if err := os.WriteFile(p, []byte(webhookURL), 0600); err != nil {
		t.Fatal(err)
	}

	path := writeTOML(t, `
[teleport]
addr = "proxy.example.com:443"
identity = "/tmp/identity"

[role_to_recipients]
"*" = ["file:`+p+`"]
`)
	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := conf.Recipients["*"][0]
	if got != webhookURL {
		t.Errorf("resolved URL = %q, want %q", got, webhookURL)
	}
}

func TestLoadConfig_nonHTTPSRejected(t *testing.T) {
	path := writeTOML(t, `
[teleport]
addr = "proxy.example.com:443"
identity = "/tmp/identity"

[role_to_recipients]
"*" = ["http://insecure.example.com/hook"]
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for non-https webhook URL")
	}
}

func TestLoadConfig_disableLogo(t *testing.T) {
	path := writeTOML(t, `
[teleport]
addr = "proxy.example.com:443"
identity = "/tmp/identity"

[msteams]
disable_logo = true

[role_to_recipients]
"*" = ["https://example.com/hook"]
`)
	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := conf.EffectiveLogoURL(); got != "" {
		t.Errorf("EffectiveLogoURL() = %q, want empty when disabled", got)
	}
}
