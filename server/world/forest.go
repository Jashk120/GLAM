package world

func GetForestLayout(cols, rows int) Layout {
	tm := createGrassMap(cols, rows)
	for x := 0; x < cols; x++ {
		tm[0][x] = Tree
		if rows > 1 {
			tm[rows-1][x] = Tree
		}
	}
	for y := 0; y < rows; y++ {
		tm[y][0] = Tree
		tm[y][cols-1] = Tree
	}
	var cl []Plot
	if cols == 15 && rows == 12 {
		cl = []Plot{
			{ID: "clearing_1", Name: "Northwest Clearing", X: 2, Y: 2, W: 4, H: 3, Type: "clearing"},
			{ID: "clearing_2", Name: "Northeast Clearing", X: 10, Y: 2, W: 4, H: 3, Type: "clearing"},
			{ID: "clearing_3", Name: "Southwest Clearing", X: 2, Y: 7, W: 5, H: 3, Type: "clearing"},
			{ID: "clearing_4", Name: "Southeast Clearing", X: 9, Y: 8, W: 4, H: 3, Type: "clearing"},
		}
	} else {
		wA, hA := cols*27/100, rows*25/100
		if wA < 2 {
			wA = 2
		}
		if hA < 2 {
			hA = 2
		}
		wB := cols * 33 / 100
		if wB < 2 {
			wB = 2
		}
		c2x, c3y, c4x, c4y := cols-wA-1, rows-hA-1, cols-wA-1, rows-hA
		if c2x < 1 {
			c2x = 1
		}
		if c3y < 1 {
			c3y = 1
		}
		if c4x < 1 {
			c4x = 1
		}
		if c4y < 1 {
			c4y = 1
		}
		cl = []Plot{
			{ID: "clearing_1", Name: "Northwest Clearing", X: 1, Y: 1, W: wA, H: hA, Type: "clearing"},
			{ID: "clearing_2", Name: "Northeast Clearing", X: c2x, Y: 1, W: wA, H: hA, Type: "clearing"},
			{ID: "clearing_3", Name: "Southwest Clearing", X: 1, Y: c3y, W: wB, H: hA, Type: "clearing"},
			{ID: "clearing_4", Name: "Southeast Clearing", X: c4x, Y: c4y, W: wA, H: hA, Type: "clearing"},
		}
		for i := range cl {
			c := &cl[i]
			if c.X+c.W >= cols {
				c.W = cols - c.X - 1
			}
			if c.Y+c.H >= rows {
				c.H = rows - c.Y - 1
			}
			if c.X < 1 {
				c.X = 1
			}
			if c.Y < 1 {
				c.Y = 1
			}
			if c.W < 1 {
				c.W = 1
			}
			if c.H < 1 {
				c.H = 1
			}
		}
	}
	for _, c := range cl {
		for dy := 0; dy < c.H; dy++ {
			for dx := 0; dx < c.W; dx++ {
				xx, yy := c.X+dx, c.Y+dy
				if yy >= 0 && yy < rows && xx >= 0 && xx < cols {
					tm[yy][xx] = Grass
				}
			}
		}
	}
	if cols == 15 && rows == 12 {
		fixed := [][2]int{{5, 5}, {6, 5}, {8, 3}, {5, 3}, {7, 5}, {12, 6}, {11, 6}, {4, 6}, {6, 9}, {13, 5}, {3, 5}, {7, 10}}
		for _, p := range fixed {
			x, y := p[0], p[1]
			if IsInPlot(x, y, cl) {
				continue
			}
			if y > 0 && y < rows-1 && x > 0 && x < cols-1 {
				tm[y][x] = Tree
			}
		}
	} else {
		for y := 1; y < rows-1; y++ {
			for x := 1; x < cols-1; x++ {
				if IsInPlot(x, y, cl) {
					continue
				}
				v := (x*37 + y*71 + (x*y)%19) % 100
				if v < 13 {
					tm[y][x] = Tree
				}
			}
		}
	}
	for _, c := range cl {
		for dy := 0; dy < c.H; dy++ {
			for dx := 0; dx < c.W; dx++ {
				xx, yy := c.X+dx, c.Y+dy
				if yy > 0 && yy < rows-1 && xx > 0 && xx < cols-1 {
					tm[yy][xx] = Grass
				}
			}
		}
	}
	return Layout{Tilemap: tm, Plots: cl}
}
