package scenario

import "glam/server/world"

// PositionInPlot reports whether pos lies inside plot (inclusive left/top,
// exclusive right/bottom).
func PositionInPlot(pos Position, plot world.Plot) bool {
	return pos.X >= plot.X && pos.X < plot.X+plot.W && pos.Y >= plot.Y && pos.Y < plot.Y+plot.H
}

// PositionInAnyPlot reports whether pos lies inside any of plots.
// Reuses world.IsInPlot to avoid duplicating bounds logic.
func PositionInAnyPlot(pos Position, plots []world.Plot) bool {
	return world.IsInPlot(pos.X, pos.Y, plots)
}

// IsInWorldBounds reports whether pos is within world size (0 <= x < cols, 0 <= y < rows).
func IsInWorldBounds(pos Position, size Size) bool {
	return pos.X >= 0 && pos.X < size.Cols && pos.Y >= 0 && pos.Y < size.Rows
}

// IsPositionInBounds is a convenience wrapper for raw cols/rows.
func IsPositionInBounds(pos Position, cols, rows int) bool {
	return pos.X >= 0 && pos.X < cols && pos.Y >= 0 && pos.Y < rows
}

// PlotCenter returns the floor center position of plot.
func PlotCenter(plot world.Plot) Position {
	return Position{X: plot.X + plot.W/2, Y: plot.Y + plot.H/2}
}

// FindContainingPlot returns the plot containing pos, or nil if none.
func FindContainingPlot(pos Position, plots []world.Plot) *world.Plot {
	for i := range plots {
		if PositionInPlot(pos, plots[i]) {
			p := plots[i]
			return &p
		}
	}
	return nil
}

// NearestPlot returns the plot whose center is closest to pos by Manhattan
// distance. Returns nil if plots is empty.
func NearestPlot(pos Position, plots []world.Plot) *world.Plot {
	if len(plots) == 0 {
		return nil
	}
	bestIdx := 0
	bestDist := int(^uint(0) >> 1)
	for i, p := range plots {
		c := PlotCenter(p)
		d := abs(pos.X-c.X) + abs(pos.Y-c.Y)
		if d < bestDist {
			bestDist = d
			bestIdx = i
		}
	}
	p := plots[bestIdx]
	return &p
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
