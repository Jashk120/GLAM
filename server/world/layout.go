package world

import "fmt"

type TileKind string

const Grass TileKind = "grass"
const Path TileKind = "path"
const Tree TileKind = "tree"
const Water TileKind = "water"

type Plot struct {
	ID   string
	Name string
	X    int
	Y    int
	W    int
	H    int
	Type string
}
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}
type Layout struct {
	Tilemap [][]TileKind
	Plots   []Plot
	Spawn   Position
}

func IsSolidTile(k TileKind) bool { return k == Tree || k == Water }
func IsWalkable(k TileKind) bool  { return !IsSolidTile(k) }
func IsSolid(k TileKind) bool     { return IsSolidTile(k) }
func IsInPlot(x, y int, plots []Plot) bool {
	for _, p := range plots {
		if x >= p.X && x < p.X+p.W && y >= p.Y && y < p.Y+p.H {
			return true
		}
	}
	return false
}
func createGrassMap(cols, rows int) [][]TileKind {
	m := make([][]TileKind, rows)
	for y := 0; y < rows; y++ {
		r := make([]TileKind, cols)
		for x := 0; x < cols; x++ {
			r[x] = Grass
		}
		m[y] = r
	}
	return m
}
func cloneMap(m [][]TileKind) [][]TileKind {
	o := make([][]TileKind, len(m))
	for i, r := range m {
		cp := make([]TileKind, len(r))
		copy(cp, r)
		o[i] = cp
	}
	return o
}
func ensureWalkableSpawn(tilemap [][]TileKind, s Position) Position {
	rows := len(tilemap)
	if rows == 0 {
		return s
	}
	cols := len(tilemap[0])
	if cols == 0 {
		return s
	}
	if s.Y >= 0 && s.Y < rows && s.X >= 0 && s.X < cols && IsWalkable(tilemap[s.Y][s.X]) {
		return s
	}
	mx := cols
	if rows > mx {
		mx = rows
	}
	for d := 1; d <= mx; d++ {
		for dy := -d; dy <= d; dy++ {
			for dx := -d; dx <= d; dx++ {
				adx := dx
				if adx < 0 {
					adx = -adx
				}
				ady := dy
				if ady < 0 {
					ady = -ady
				}
				if adx != d && ady != d {
					continue
				}
				nx, ny := s.X+dx, s.Y+dy
				if nx < 0 || ny < 0 || nx >= cols || ny >= rows {
					continue
				}
				if IsWalkable(tilemap[ny][nx]) {
					return Position{X: nx, Y: ny}
				}
			}
		}
	}
	return s
}
func GetTownLayout(cols, rows int) Layout {
	tm := createGrassMap(cols, rows)
	rx1, rx2 := cols/3, (2*cols)/3
	ry1, ry2 := rows/3, (2*rows)/3
	for x := 0; x < cols; x++ {
		if ry1 >= 0 && ry1 < rows {
			tm[ry1][x] = Path
		}
		if ry2 >= 0 && ry2 < rows {
			tm[ry2][x] = Path
		}
	}
	for y := 0; y < rows; y++ {
		if rx1 >= 0 && rx1 < cols {
			tm[y][rx1] = Path
		}
		if rx2 >= 0 && rx2 < cols {
			tm[y][rx2] = Path
		}
	}
	type xs struct{ a, b int }
	type ys struct{ a, b int; l string }
	xSegs := []xs{{0, rx1 - 1}, {rx1 + 1, rx2 - 1}, {rx2 + 1, cols - 1}}
	ySegs := []ys{{0, ry1 - 1, "North"}, {ry2 + 1, rows - 1, "South"}}
	var plots []Plot
	id := 1
	for _, ysg := range ySegs {
		if ysg.a > ysg.b {
			continue
		}
		for xi, xsg := range xSegs {
			if xsg.a > xsg.b {
				continue
			}
			w, h := xsg.b-xsg.a+1, ysg.b-ysg.a+1
			if w <= 0 || h <= 0 {
				continue
			}
			xL := "West"
			if xi == 1 {
				xL = "Central"
			} else if xi == 2 {
				xL = "East"
			}
			plots = append(plots, Plot{ID: fmt.Sprintf("plot_%d", id), Name: ysg.l + " " + xL + " Plot", X: xsg.a, Y: ysg.a, W: w, H: h, Type: "plot"})
			id++
		}
	}
	return Layout{Tilemap: tm, Plots: plots}
}
func GetLayout(template string, cols, rows int) *Layout {
	if cols < 8 || cols > 30 || rows < 8 || rows > 20 {
		return nil
	}
	var tm [][]TileKind
	var plots []Plot
	switch template {
	case "town":
		r := GetTownLayout(cols, rows)
		tm, plots = r.Tilemap, r.Plots
	case "forest":
		r := GetForestLayout(cols, rows)
		tm, plots = r.Tilemap, r.Plots
	case "desert", "school":
		tm = createGrassMap(cols, rows)
		plots = []Plot{}
	default:
		return nil
	}
	spawn := ensureWalkableSpawn(tm, Position{X: cols / 2, Y: rows / 2})
	return &Layout{Tilemap: cloneMap(tm), Plots: plots, Spawn: spawn}
}
func GetPlotsForTemplate(template string, cols, rows int) []Plot {
	l := GetLayout(template, cols, rows)
	if l == nil {
		return nil
	}
	return l.Plots
}
