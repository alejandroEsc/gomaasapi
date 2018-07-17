package maasapiv2

type Version struct {
	Capabilities []string `json:"capabilities,omitempty"`
	Version      string   `json:"version,omitempty"`
	SubVersion   string   `json:"subversion,omitempty"`
}
