package report

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"
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
	InternalRefCount int      `json:"internal_ref_count"`
	Confidence       string   `json:"confidence"`
	Reasons          []string `json:"reasons"`
	kind             string
	src              string
}

type candidateJSON struct {
	Symbol           string   `json:"symbol"`
	Kind             string   `json:"kind"`
	Src              string   `json:"src"`
	InternalRefCount int      `json:"internal_ref_count"`
	Confidence       string   `json:"confidence"`
	Reasons          []string `json:"reasons"`
}

func NewCandidate(
	symbol string,
	kind string,
	src string,
	internalRefCount int,
	confidence string,
	reasons []string,
) Candidate {
	return Candidate{
		Symbol:           symbol,
		InternalRefCount: internalRefCount,
		Confidence:       confidence,
		Reasons:          slices.Clone(reasons),
		kind:             kind,
		src:              src,
	}
}

func (c Candidate) MarshalJSON() ([]byte, error) {
	return json.Marshal(candidateJSON{
		Symbol:           c.Symbol,
		Kind:             c.kind,
		Src:              c.src,
		InternalRefCount: c.InternalRefCount,
		Confidence:       c.Confidence,
		Reasons:          c.Reasons,
	})
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
	src := sanitizeTextField(c.src)
	name := sanitizeTextField(shortName(c.Symbol))
	kind := sanitizeTextField(c.kind)
	if c.Confidence == ConfidenceLow && len(c.Reasons) > 0 {
		reasons := make([]string, 0, len(c.Reasons))
		for _, reason := range c.Reasons {
			reasons = append(reasons, sanitizeTextField(reason))
		}
		return fmt.Sprintf(
			"%s: %s (%s) [low: %s]",
			src, name, kind,
			strings.Join(reasons, ", "),
		)
	}
	return fmt.Sprintf("%s: %s (%s)", src, name, kind)
}

func shortName(symbol string) string {
	if idx := strings.LastIndex(symbol, "."); idx >= 0 {
		return symbol[idx+1:]
	}
	return symbol
}

func sanitizeTextField(value string) string {
	if value == "" {
		return value
	}

	var b strings.Builder
	b.Grow(len(value))

	for _, r := range value {
		if !unicode.IsControl(r) {
			b.WriteRune(r)
			continue
		}

		switch {
		case r <= 0xFF:
			fmt.Fprintf(&b, "\\x%02X", r)
		case r <= 0xFFFF:
			fmt.Fprintf(&b, "\\u%04X", r)
		default:
			fmt.Fprintf(&b, "\\U%08X", r)
		}
	}

	return b.String()
}
