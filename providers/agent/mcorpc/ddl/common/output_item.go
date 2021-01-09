package common

// OutputItem describes an individual output item
type OutputItem struct {
	Description string      `json:"description"`
	DisplayAs   string      `json:"display_as"`
	Default     interface{} `json:"default,omitempty"`
	Type        string      `json:"type,omitempty"`
}
