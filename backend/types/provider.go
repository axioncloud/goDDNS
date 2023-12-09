package types

// provider represents data about a DDNS Provider.
type Provider struct {
	UUID     string
	NAME     string
	URL      string
	SELECTED uint8
}
