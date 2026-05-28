package bot

import (
	"net/url"
	"strings"
)

type teamsMessage struct {
	Type        string       `json:"type"`
	Attachments []attachment `json:"attachments"`
}

type attachment struct {
	ContentType string      `json:"contentType"`
	ContentURL  interface{} `json:"contentUrl"`
	Content     adaptiveCard `json:"content"`
}

type adaptiveCard struct {
	Schema  string        `json:"$schema"`
	Type    string        `json:"type"`
	Version string        `json:"version"`
	Body    []interface{} `json:"body"`
	Actions []interface{} `json:"actions,omitempty"`
}

type imageElement struct {
	Type                string `json:"type"`
	URL                 string `json:"url"`
	Size                string `json:"size,omitempty"`
	HorizontalAlignment string `json:"horizontalAlignment,omitempty"`
}

type columnSet struct {
	Type    string   `json:"type"`
	Columns []column `json:"columns"`
}

type column struct {
	Type  string        `json:"type"`
	Width string        `json:"width"`
	Items []interface{} `json:"items"`
}

type textBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Weight string `json:"weight,omitempty"`
	Size   string `json:"size,omitempty"`
	Color  string `json:"color,omitempty"`
	Wrap   bool   `json:"wrap,omitempty"`
}

type factSet struct {
	Type  string `json:"type"`
	Facts []fact `json:"facts"`
}

type fact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type openURLAction struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// BuildCard constructs the Teams Adaptive Card payload for an access request event.
// logoURL may be empty to omit the logo; webProxyURL may be nil to omit the action button.
func BuildCard(reqID string, data RequestData, webProxyURL *url.URL, logoURL string) teamsMessage {
	title, statusText, statusColor := resolveState(data.ResolutionTag)

	var body []interface{}

	if logoURL != "" {
		body = append(body, imageElement{
			Type:                "Image",
			URL:                 logoURL,
			Size:                "Stretch",
			HorizontalAlignment: "Center",
		})
	}

	body = append(body, columnSet{
		Type: "ColumnSet",
		Columns: []column{
			{
				Type:  "Column",
				Width: "stretch",
				Items: []interface{}{
					textBlock{Type: "TextBlock", Text: title, Weight: "Bolder", Size: "Medium", Wrap: true},
				},
			},
			{
				Type:  "Column",
				Width: "auto",
				Items: []interface{}{
					textBlock{Type: "TextBlock", Text: statusText, Color: statusColor, Weight: "Bolder"},
				},
			},
		},
	})

	facts := []fact{
		{Title: "Requester", Value: data.User},
	}
	if len(data.Roles) > 0 {
		facts = append(facts, fact{Title: "Roles", Value: strings.Join(data.Roles, ", ")})
	}
	if data.RequestReason != "" {
		facts = append(facts, fact{Title: "Reason", Value: data.RequestReason})
	}
	if data.ResolutionReason != "" {
		facts = append(facts, fact{Title: "Resolution", Value: data.ResolutionReason})
	}

	body = append(body, factSet{Type: "FactSet", Facts: facts})

	var actions []interface{}
	if webProxyURL != nil {
		u := *webProxyURL
		u.Path = "/web/requests/" + reqID
		actions = append(actions, openURLAction{
			Type:  "Action.OpenUrl",
			Title: "View Request →",
			URL:   u.String(),
		})
	}

	return teamsMessage{
		Type: "message",
		Attachments: []attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				ContentURL:  nil,
				Content: adaptiveCard{
					Schema:  "https://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.4",
					Body:    body,
					Actions: actions,
				},
			},
		},
	}
}

func resolveState(tag ResolutionTag) (title, status, color string) {
	switch tag {
	case Approved:
		return "Access Request Approved", "APPROVED", "Good"
	case Denied:
		return "Access Request Denied", "DENIED", "Warning"
	case Expired:
		return "Access Request Expired", "EXPIRED", "Default"
	case Promoted:
		return "Access Request Promoted", "PROMOTED", "Good"
	default:
		return "New Access Request", "PENDING", "Attention"
	}
}
