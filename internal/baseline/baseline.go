package baseline

import (
	"encoding/json"
	"fmt"
	"os"
)

type acceptedCandidate struct {
	Symbol string `json:"symbol"`
}

func Load(path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open baseline: %w", err)
	}
	defer file.Close()

	var accepted []acceptedCandidate
	if err := json.NewDecoder(file).Decode(&accepted); err != nil {
		return nil, fmt.Errorf("decode baseline: %w", err)
	}

	acceptedBySymbol := make(map[string]struct{}, len(accepted))
	for _, candidate := range accepted {
		if candidate.Symbol == "" {
			return nil, fmt.Errorf("decode baseline: symbol is required")
		}
		acceptedBySymbol[candidate.Symbol] = struct{}{}
	}

	return acceptedBySymbol, nil
}
