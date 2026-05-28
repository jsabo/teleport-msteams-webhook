package config

import (
	"os"
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
