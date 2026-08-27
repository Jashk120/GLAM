package scenario

import (
	"encoding/json"
	"fmt"
	"os"
)

// Asset represents a single entry in asset-registry.json
type Asset struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Bundle   string          `json:"bundle"`
	Sprite   string          `json:"sprite"`
	Icon     string          `json:"icon"`
	Solid    bool            `json:"solid"`
	Metadata json.RawMessage `json:"metadata"`
}

// LoadRegistry loads registry file and returns map of id -> Asset.
func LoadRegistry(path string) (map[string]Asset, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read registry: %w", err)
	}
	var assets []Asset
	if err := json.Unmarshal(data, &assets); err != nil {
		return nil, nil, fmt.Errorf("parse registry: %w", err)
	}
	m := make(map[string]Asset, len(assets))
	for _, a := range assets {
		m[a.ID] = a
	}
	return m, data, nil
}
