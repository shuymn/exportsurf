package report

import (
	"encoding/json"
	"io"
)

const ConfidenceHigh = "high"

type Candidate struct {
	Symbol              string   `json:"symbol"`
	Kind                string   `json:"kind"`
	DefinedIn           string   `json:"defined_in"`
	InternalRefCount    int      `json:"internal_ref_count"`
	ExternalRefPkgCount int      `json:"external_ref_pkg_count"`
	Confidence          string   `json:"confidence"`
	Notes               []string `json:"notes"`
}

func WriteJSON(w io.Writer, candidates []Candidate) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(candidates)
}
