package bot

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func cardJSON(t *testing.T, msg teamsMessage) string {
	t.Helper()
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBuildCard_pending(t *testing.T) {
	data := RequestData{
		User:          "alice",
		Roles:         []string{"database-admin", "dev"},
		RequestReason: "P1 incident",
	}
	msg := BuildCard("req-123", data, mustURL(t, "https://proxy.example.com"), "")
	j := cardJSON(t, msg)

	for _, want := range []string{"alice", "database-admin, dev", "P1 incident", "PENDING", "New Access Request", "/web/requests/req-123"} {
		if !strings.Contains(j, want) {
			t.Errorf("card JSON missing %q\n%s", want, j)
		}
	}
}

func TestBuildCard_approved(t *testing.T) {
	data := RequestData{
		User:             "alice",
		Roles:            []string{"dev"},
		ResolutionTag:    Approved,
		ResolutionReason: "looks good",
	}
	msg := BuildCard("req-456", data, mustURL(t, "https://proxy.example.com"), "")
	j := cardJSON(t, msg)

	for _, want := range []string{"APPROVED", "Access Request Approved", "looks good"} {
		if !strings.Contains(j, want) {
			t.Errorf("card JSON missing %q\n%s", want, j)
		}
	}
}

func TestBuildCard_denied(t *testing.T) {
	data := RequestData{
		User:          "bob",
		Roles:         []string{"admin"},
		ResolutionTag: Denied,
	}
	msg := BuildCard("req-789", data, nil, "")
	j := cardJSON(t, msg)

	if !strings.Contains(j, "DENIED") {
		t.Errorf("card JSON missing DENIED\n%s", j)
	}
	if !strings.Contains(j, "Access Request Denied") {
		t.Errorf("card JSON missing title\n%s", j)
	}
	if strings.Contains(j, "View Request") {
		t.Errorf("expected no action button when webProxyURL is nil\n%s", j)
	}
}

func TestBuildCard_emptyReason_omitted(t *testing.T) {
	data := RequestData{User: "charlie", Roles: []string{"viewer"}}
	msg := BuildCard("req-000", data, nil, "")
	j := cardJSON(t, msg)

	if strings.Contains(j, "Reason") {
		t.Errorf("expected Reason fact to be omitted when empty\n%s", j)
	}
}

func TestBuildCard_multipleRoles(t *testing.T) {
	data := RequestData{User: "dave", Roles: []string{"a", "b", "c"}}
	msg := BuildCard("req-001", data, nil, "")
	j := cardJSON(t, msg)

	if !strings.Contains(j, "a, b, c") {
		t.Errorf("expected comma-separated roles\n%s", j)
	}
}

func TestBuildCard_withLogo(t *testing.T) {
	data := RequestData{User: "eve", Roles: []string{"dev"}}
	logoURL := "https://example.com/logo.png"
	msg := BuildCard("req-002", data, nil, logoURL)
	j := cardJSON(t, msg)

	if !strings.Contains(j, logoURL) {
		t.Errorf("expected logo URL in card JSON\n%s", j)
	}
}

func TestBuildCard_withoutLogo(t *testing.T) {
	data := RequestData{User: "eve", Roles: []string{"dev"}}
	msg := BuildCard("req-003", data, nil, "")
	j := cardJSON(t, msg)

	if strings.Contains(j, "Image") {
		t.Errorf("expected no Image element when logoURL is empty\n%s", j)
	}
}
