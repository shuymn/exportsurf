package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	ConfidenceHigh = "high"
	ConfidenceLow  = "low"
)

type Candidate struct {
	Symbol              string   `json:"symbol"`
	Kind                string   `json:"kind"`
	DefinedIn           string   `json:"defined_in"`
	InternalRefCount    int      `json:"internal_ref_count"`
	ExternalRefPkgCount int      `json:"external_ref_pkg_count"`
	ExternalRefExamples []string `json:"external_ref_examples"`
	Confidence          string   `json:"confidence"`
	Reasons             []string `json:"reasons"`
}

func WriteJSON(w io.Writer, candidates []Candidate) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(candidates)
}

func WriteText(w io.Writer, candidates []Candidate) error {
	for _, c := range candidates {
		line := formatTextLine(c)
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func formatTextLine(c Candidate) string {
	name := shortName(c.Symbol)
	if c.Confidence == ConfidenceLow && len(c.Reasons) > 0 {
		return fmt.Sprintf(
			"%s: %s (%s) [low: %s]",
			c.DefinedIn, name, c.Kind,
			strings.Join(c.Reasons, ", "),
		)
	}
	return fmt.Sprintf("%s: %s (%s)", c.DefinedIn, name, c.Kind)
}

func shortName(symbol string) string {
	if idx := strings.LastIndex(symbol, "."); idx >= 0 {
		return symbol[idx+1:]
	}
	return symbol
}
