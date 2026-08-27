package scenario

import (
	"encoding/json"

	"glam/server/world"
)

// NormalizePlotRefs attempts to auto-fix cross-template plot IDs and
// mismatched plot/position combos. It is a best-effort salvage for
// LLM hallucinations like "clearing_2" inside a town.
//
// Strategy:
//   - desert/school have no plots: any plot field is stripped.
//   - town valid IDs: plot_1..plot_6 (dynamic based on cols/rows, but
//     forest IDs are clearing_* and vice-versa).
//   - If plot ID is invalid for the template, try to remap to the plot
//     that actually contains the entity's position. If position is
//     outside all plots, snap position to the nearest plot center and
//     set plot accordingly.
//   - If plot ID is valid but position is not inside the claimed plot,
//     remap plot to the containing plot if found, otherwise snap
//     position to the center of the claimed plot.
//
// Returns normalized JSON (or original if no fix), whether any fix was
// applied, and any marshal error.
func NormalizePlotRefs(data []byte) ([]byte, bool, error) {
	var sc Scenario
	if err := json.Unmarshal(data, &sc); err != nil {
		return data, false, nil // not our problem — let validator report JSON error
	}

	layout := world.GetLayout(sc.World.Template, sc.World.Size.Cols, sc.World.Size.Rows)
	if layout == nil {
		return data, false, nil
	}

	// Build lookup of valid plot IDs
	validIDs := map[string]world.Plot{}
	for _, p := range layout.Plots {
		validIDs[p.ID] = p
	}

	fixed := false

	// Helper: find plot containing position
	findContaining := func(pos Position) *world.Plot {
		for _, p := range layout.Plots {
			if pos.X >= p.X && pos.X < p.X+p.W && pos.Y >= p.Y && pos.Y < p.Y+p.H {
				return &p
			}
		}
		return nil
	}

	// Helper: center of plot (floor)
	centerOf := func(p world.Plot) Position {
		return Position{X: p.X + p.W/2, Y: p.Y + p.H/2}
	}

	// Helper: nearest plot by Manhattan distance to center
	nearestPlot := func(pos Position) *world.Plot {
		if len(layout.Plots) == 0 {
			return nil
		}
		bestIdx := 0
		bestDist := int(^uint(0) >> 1)
		for i, p := range layout.Plots {
			c := centerOf(p)
			d := abs(pos.X-c.X) + abs(pos.Y-c.Y)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		return &layout.Plots[bestIdx]
	}

	// desert/school have no plots — strip any plot field
	if len(layout.Plots) == 0 {
		for i := range sc.Characters {
			if sc.Characters[i].Plot != nil {
				sc.Characters[i].Plot = nil
				fixed = true
			}
		}
		for i := range sc.Buildings {
			if sc.Buildings[i].Plot != nil {
				sc.Buildings[i].Plot = nil
				fixed = true
			}
		}
		for i := range sc.Objects {
			if sc.Objects[i].Plot != nil {
				sc.Objects[i].Plot = nil
				fixed = true
			}
		}
		if fixed {
			out, err := json.Marshal(sc)
			if err != nil {
				return data, false, err
			}
			return out, true, nil
		}
		return data, false, nil
	}

	// Generic fixer for one entity
	fixOne := func(pos *Position, plot **string) {
		if *plot == nil {
			// No plot claim — if position outside all plots, snap to nearest
			if findContaining(*pos) == nil {
				if np := nearestPlot(*pos); np != nil {
					*pos = centerOf(*np)
					// Keep plot nil (position now valid), or set to np.ID?
					// Prefer setting to the plot we snapped to for clarity.
					id := np.ID
					*plot = &id
					fixed = true
				}
			}
			return
		}
		plID := **plot
		claimed, isValid := validIDs[plID]
		if !isValid {
			// Cross-template or typo — remap to containing plot
			if containing := findContaining(*pos); containing != nil {
				**plot = containing.ID
				fixed = true
			} else {
				// Position not in any valid plot — snap to nearest valid plot
				if np := nearestPlot(*pos); np != nil {
					nid := np.ID
					**plot = nid
					*pos = centerOf(*np)
					fixed = true
				}
			}
			return
		}
		// Valid ID but check if position inside claimed
		if !(pos.X >= claimed.X && pos.X < claimed.X+claimed.W && pos.Y >= claimed.Y && pos.Y < claimed.Y+claimed.H) {
			if containing := findContaining(*pos); containing != nil {
				// Prefer the actual containing plot over claimed
				**plot = containing.ID
				fixed = true
			} else {
				// Snap position into claimed plot
				*pos = centerOf(claimed)
				fixed = true
			}
		}
	}

	for i := range sc.Characters {
		fixOne(&sc.Characters[i].Position, &sc.Characters[i].Plot)
	}
	for i := range sc.Buildings {
		// For buildings with footprint, use same logic on origin position
		fixOne(&sc.Buildings[i].Position, &sc.Buildings[i].Plot)
		// Ensure footprint doesn't exceed world bounds — if it does, clamp width/height
		if sc.Buildings[i].Width != nil || sc.Buildings[i].Height != nil {
			w, h := 1, 1
			if sc.Buildings[i].Width != nil {
				w = *sc.Buildings[i].Width
			}
			if sc.Buildings[i].Height != nil {
				h = *sc.Buildings[i].Height
			}
			cols := sc.World.Size.Cols
			rows := sc.World.Size.Rows
			if sc.Buildings[i].Position.X+w > cols {
				nw := cols - sc.Buildings[i].Position.X
				if nw < 1 {
					nw = 1
				}
				sc.Buildings[i].Width = &nw
				fixed = true
			}
			if sc.Buildings[i].Position.Y+h > rows {
				nh := rows - sc.Buildings[i].Position.Y
				if nh < 1 {
					nh = 1
				}
				sc.Buildings[i].Height = &nh
				fixed = true
			}
		}
	}
	for i := range sc.Objects {
		fixOne(&sc.Objects[i].Position, &sc.Objects[i].Plot)
	}

	if !fixed {
		return data, false, nil
	}
	out, err := json.Marshal(sc)
	if err != nil {
		return data, false, err
	}
	return out, true, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
