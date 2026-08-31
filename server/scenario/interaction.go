package scenario

// Canonical interaction types — mirrors schema/scenario.schema.json $defs/interaction oneOf
// and client/src/types/interactionTypes.ts. Keep in sync; schema is the canonical source.

// InteractionTypes is the ordered list of valid interaction.type values.
var InteractionTypes = []string{"dialogue", "mcq", "math", "shop", "information"}

// ValidActivityTypes is the allowed set for validator checks. Comment references canonical.
var ValidActivityTypes = map[string]bool{
	"dialogue":    true,
	"mcq":         true,
	"math":        true,
	"shop":        true,
	"information": true,
}

// IsValidInteractionType reports whether t is a known interaction type.
func IsValidInteractionType(t string) bool {
	return ValidActivityTypes[t]
}
