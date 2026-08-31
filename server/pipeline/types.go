package pipeline

import (
	"encoding/json"

	scenario_pkg "glam/server/scenario"
)

type StructureDraft struct {
	Template string            `json:"template"`
	Size     scenario_pkg.Size `json:"size"`
	Entities []EntityDraft     `json:"entities"`
}

type EntityDraft struct {
	ID       string                `json:"id"`
	Kind     string                `json:"kind"`
	Position scenario_pkg.Position `json:"position"`
	Plot     *string               `json:"plot,omitempty"`
	Role     string                `json:"role"`
}

type FlavorDraft struct {
	Title string                  `json:"title"`
	Names map[string]EntityFlavor `json:"names"`
}

type EntityFlavor struct {
	Name        string  `json:"name"`
	Profession  *string `json:"profession,omitempty"`
	TypeAssetID *string `json:"typeAssetId,omitempty"`
}

type InteractionDraft map[string]json.RawMessage

type Deps struct {
	SchemaJSON   []byte
	RegistryJSON []byte
	SchemaPath   string
	RegistryPath string
}
