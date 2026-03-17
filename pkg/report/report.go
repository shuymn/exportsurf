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

	ReasonPackageMain      = "package_main"
	ReasonPackageUnderCmd  = "package_under_cmd"
	ReasonGeneratedFile    = "generated_file"
	ReasonEmbeddedField    = "embedded_field"
	ReasonSerializationTag = "serialization_tag"
	ReasonReflectUsage     = "reflect_usage"
	ReasonPluginUsage      = "plugin_usage"
	ReasonCgoExport        = "cgo_export"
	ReasonLinkname         = "linkname"
)

func SatisfiesInterfaceReason(iface string) string {
	return "satisfies_interface:" + iface
}

type Candidate struct {
	Symbol           string   `json:"symbol"`
	Kind             string   `json:"kind"`
	DefinedIn        string   `json:"defined_in"`
	InternalRefCount int      `json:"internal_ref_count"`
	Confidence       string   `json:"confidence"`
	Reasons          []string `json:"reasons"`
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
