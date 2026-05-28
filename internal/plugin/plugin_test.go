package plugin

import (
	"testing"
)

func TestResolve_explicitRole(t *testing.T) {
	p := &Plugin{recipients: map[string][]string{
		"dev":     {"https://example.com/dev"},
		"*":       {"https://example.com/general"},
	}}
	got := p.resolve([]string{"dev"})
	if len(got) != 1 || got[0] != "https://example.com/dev" {
		t.Errorf("resolve(dev) = %v, want [https://example.com/dev]", got)
	}
}

func TestResolve_wildcardFallback(t *testing.T) {
	p := &Plugin{recipients: map[string][]string{
		"*": {"https://example.com/general"},
	}}
	got := p.resolve([]string{"unknown-role"})
	if len(got) != 1 || got[0] != "https://example.com/general" {
		t.Errorf("resolve(unknown) = %v, want wildcard URL", got)
	}
}

func TestResolve_deduplication(t *testing.T) {
	p := &Plugin{recipients: map[string][]string{
		"dev": {"https://example.com/shared"},
		"ops": {"https://example.com/shared"},
	}}
	got := p.resolve([]string{"dev", "ops"})
	if len(got) != 1 {
		t.Errorf("resolve with duplicate URLs returned %d results, want 1: %v", len(got), got)
	}
}

func TestResolve_multipleRolesMultipleURLs(t *testing.T) {
	p := &Plugin{recipients: map[string][]string{
		"dev": {"https://example.com/dev"},
		"ops": {"https://example.com/ops"},
	}}
	got := p.resolve([]string{"dev", "ops"})
	if len(got) != 2 {
		t.Errorf("resolve returned %d results, want 2: %v", len(got), got)
	}
}

func TestResolve_emptyRoles(t *testing.T) {
	p := &Plugin{recipients: map[string][]string{
		"*": {"https://example.com/general"},
	}}
	got := p.resolve([]string{})
	if len(got) != 0 {
		t.Errorf("resolve([]) = %v, want empty", got)
	}
}
