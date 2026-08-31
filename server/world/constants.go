package world

// canonical: schema/scenario.schema.json — world.size and position bounds
// Mirrored in client/src/world/worldConstants.ts — keep in sync.

const (
	WorldColsMin = 8
	WorldColsMax = 30
	WorldRowsMin = 8
	WorldRowsMax = 20
	WorldPosMin  = 0
	WorldPosMax  = 30
)

// Forest layout constants — mirrored in client/src/world/worldConstants.ts
const (
	ForestTreeDensityThreshold = 13
	ForestClearingWFactorA     = 27 // numerator over 100 → 0.27
	ForestClearingHFactor      = 25 // numerator over 100 → 0.25
	ForestClearingWFactorB     = 33 // numerator over 100 → 0.33
	ForestClearingMinSize      = 2
	ForestHashA                = 37
	ForestHashB                = 71
	ForestHashMod              = 19
	ForestHashRange            = 100
)
