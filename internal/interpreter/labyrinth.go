package interpreter

import (
	"fmt"
	"os"
)

type Labyrinth struct {
	Grid  [][][]int
	Start [3]int
	Exit  [3]int
}

// Load labyrinth from text file
// Format:
// first line: X Y Z sizes
// next line: startX startY startZ
// next line: exitX exitY exitZ
// then Z layers separated by blank line. Each layer is Y lines of X digits (0 or 1)

func LoadLabyrinth(path string) (*Labyrinth, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var sx, sy, sz int
	var ex, ey, ez int
	var xsize, ysize, zsize int
	fmt.Fscan(f, &xsize, &ysize, &zsize)
	fmt.Fscan(f, &sx, &sy, &sz)
	fmt.Fscan(f, &ex, &ey, &ez)
	grid := make([][][]int, zsize)
	for z := 0; z < zsize; z++ {
		grid[z] = make([][]int, ysize)
		for y := 0; y < ysize; y++ {
			grid[z][y] = make([]int, xsize)
			for x := 0; x < xsize; x++ {
				var c int
				fmt.Fscan(f, &c)
				grid[z][y][x] = c
			}
		}
	}
	return &Labyrinth{Grid: grid, Start: [3]int{sx, sy, sz}, Exit: [3]int{ex, ey, ez}}, nil
}

func (l *Labyrinth) InBounds(x, y, z int) bool {
	return z >= 0 && z < len(l.Grid) && y >= 0 && y < len(l.Grid[0]) && x >= 0 && x < len(l.Grid[0][0])
}

func (l *Labyrinth) IsFree(x, y, z int) bool {
	return l.InBounds(x, y, z) && l.Grid[z][y][x] == 0
}

// BFS search path length
func (l *Labyrinth) FindPath() ([][3]int, bool) {
	type node struct {
		x, y, z int
		path    [][3]int
	}
	moves := [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
	visited := make(map[[3]int]bool)
	start := node{x: l.Start[0], y: l.Start[1], z: l.Start[2], path: [][3]int{{l.Start[0], l.Start[1], l.Start[2]}}}
	q := []node{start}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		if cur.x == l.Exit[0] && cur.y == l.Exit[1] && cur.z == l.Exit[2] {
			return cur.path, true
		}
		key := [3]int{cur.x, cur.y, cur.z}
		if visited[key] {
			continue
		}
		visited[key] = true
		for _, mv := range moves {
			nx := cur.x + mv[0]
			ny := cur.y + mv[1]
			nz := cur.z + mv[2]
			if l.IsFree(nx, ny, nz) && !visited[[3]int{nx, ny, nz}] {
				np := append(append([][3]int{}, cur.path...), [3]int{nx, ny, nz})
				q = append(q, node{nx, ny, nz, np})
			}
		}
	}
	return nil, false
}
