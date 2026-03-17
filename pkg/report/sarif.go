package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const sarifSchema = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json"

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name  string      `json:"name"`
	Rules []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	ShortDescription sarifMessage `json:"shortDescription"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

const ruleID = "exportsurf/unused-export"

func WriteSARIF(w io.Writer, candidates []Candidate) error {
	results := make([]sarifResult, 0, len(candidates))
	for _, c := range candidates {
		results = append(results, candidateToSARIFResult(c))
	}

	log := sarifLog{
		Schema:  sarifSchema,
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name: "exportsurf",
						Rules: []sarifRule{
							{
								ID:               ruleID,
								ShortDescription: sarifMessage{Text: "Exported symbol with no external references"},
							},
						},
					},
				},
				Results: results,
			},
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(log)
}

func candidateToSARIFResult(c Candidate) sarifResult {
	level := "warning"
	if c.Confidence == ConfidenceLow {
		level = "note"
	}

	file, line := parseSrc(c.Src)

	return sarifResult{
		RuleID:  ruleID,
		Level:   level,
		Message: sarifMessage{Text: fmt.Sprintf("%s (%s) has no external references", c.Symbol, c.Kind)},
		Locations: []sarifLocation{
			{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: file},
					Region:           sarifRegion{StartLine: line},
				},
			},
		},
	}
}

func parseSrc(src string) (string, int) {
	idx := strings.LastIndex(src, ":")
	if idx == -1 {
		return src, 0
	}
	line, err := strconv.Atoi(src[idx+1:])
	if err != nil {
		return src, 0
	}
	return src[:idx], line
}
