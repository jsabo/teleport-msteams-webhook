package bot

// ResolutionTag represents the current state of an access request.
type ResolutionTag string

const (
	Unresolved ResolutionTag = ""
	Approved   ResolutionTag = "APPROVED"
	Denied     ResolutionTag = "DENIED"
	Expired    ResolutionTag = "EXPIRED"
	Promoted   ResolutionTag = "PROMOTED"
)

// RequestData holds the fields from a Teleport access request used to build a card.
type RequestData struct {
	User             string
	Roles            []string
	LoginsByRole     map[string][]string
	Resources        []string
	RequestReason    string
	ResolutionTag    ResolutionTag
	ResolutionReason string
}
