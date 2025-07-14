package maze

import (
	"bufio"
	"bytes"
	"drone-maze/types"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type coord struct{ X, Y, Z int }

type Maze struct {
	Cells map[coord]types.Cell
	Robot types.Robot
	Exits []coord
	minX  int
	maxX  int
	minY  int
	maxY  int

	// Явно заданные границы лабиринта (если BoundariesSet == true)
	BoundariesSet bool
	minBoundX     int
	maxBoundX     int
	minBoundY     int
	maxBoundY     int
	minBoundZ     int
	maxBoundZ     int
}

var RenderDelay = 500 * time.Millisecond

func NewMaze() *Maze {
	return &Maze{
		Cells: make(map[coord]types.Cell),
		Exits: []coord{},
		minX:  1<<31 - 1,
		maxX:  -1 << 31,
		minY:  1<<31 - 1,
		maxY:  -1 << 31,

		BoundariesSet: false,
	}
}

func LoadMaze(data []byte) (*Maze, error) {
	maze := NewMaze()
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		command := parts[0]

		switch command {
		case "BOUNDARY":
			// Формат: BOUNDARY minX minY minZ maxX maxY maxZ
			if len(parts) != 7 {
				return nil, fmt.Errorf("BOUNDARY expects 6 integers, got: %s", line)
			}
			minX, err1 := strconv.Atoi(parts[1])
			minY, err2 := strconv.Atoi(parts[2])
			minZ, err3 := strconv.Atoi(parts[3])
			maxX, err4 := strconv.Atoi(parts[4])
			maxY, err5 := strconv.Atoi(parts[5])
			maxZ, err6 := strconv.Atoi(parts[6])
			if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
				return nil, fmt.Errorf("invalid BOUNDARY coordinates in line: %s", line)
			}
			maze.BoundariesSet = true
			maze.minBoundX, maze.minBoundY, maze.minBoundZ = minX, minY, minZ
			maze.maxBoundX, maze.maxBoundY, maze.maxBoundZ = maxX, maxY, maxZ

		default:
			// Для других команд ожидаем 3 координаты
			if len(parts) != 4 {
				return nil, fmt.Errorf("invalid line format: %s", line)
			}

			x, errX := strconv.Atoi(parts[1])
			y, errY := strconv.Atoi(parts[2])
			z, errZ := strconv.Atoi(parts[3])

			if errX != nil || errY != nil || errZ != nil {
				return nil, fmt.Errorf("invalid coordinates in line: %s", line)
			}

			c := coord{X: x, Y: y, Z: z}

			switch command {
			case "ROBOT":
				maze.Robot.Position = types.Cell{X: x, Y: y, Z: z, IsObstacle: false}
				maze.updateBounds(x, y)
			case "OBSTACLE":
				maze.SetCell(types.Cell{X: x, Y: y, Z: z, IsObstacle: true})
				maze.updateBounds(x, y)
			case "EXIT":
				maze.Exits = append(maze.Exits, c)
				maze.updateBounds(x, y)
			default:
				return nil, fmt.Errorf("unknown command in maze file: %s", command)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return maze, nil
}

func (m *Maze) SetCell(c types.Cell) {
	m.Cells[coord{c.X, c.Y, c.Z}] = c
	m.updateBounds(c.X, c.Y)
}

func (m *Maze) IsObstacle(x, y, z int) bool {
	// Сначала проверяем выход за явные границы
	if m.BoundariesSet {
		if x < m.minBoundX || x > m.maxBoundX || y < m.minBoundY || y > m.maxBoundY || z < m.minBoundZ || z > m.maxBoundZ {
			return true
		}
	}

	if cell, ok := m.Cells[coord{x, y, z}]; ok {
		return cell.IsObstacle
	}
	return false // неизвестная клетка свободна
}

func (m *Maze) Move(dx, dy, dz int) bool {
	if m.Robot.IsBroken {
		return false
	}

	newX := m.Robot.Position.X + dx
	newY := m.Robot.Position.Y + dy
	newZ := m.Robot.Position.Z + dz

	if m.IsObstacle(newX, newY, newZ) {
		m.Robot.IsBroken = true
		return false
	}

	m.Robot.Position.X = newX
	m.Robot.Position.Y = newY
	m.Robot.Position.Z = newZ

	m.updateBounds(newX, newY)

	// после каждого успешного шага рисуем
	m.Draw()
	time.Sleep(RenderDelay)

	return true
}

func (m *Maze) Measure(dx, dy, dz int) int {
	if m.Robot.IsBroken {
		return -1
	}

	x, y, z := m.Robot.Position.X, m.Robot.Position.Y, m.Robot.Position.Z
	distance := 0
	for {
		x += dx
		y += dy
		z += dz
		distance++
		if m.IsObstacle(x, y, z) {
			return distance
		}
		if distance > 1000 {
			return -1 // Indicate "infinity" or no obstacle found
		}
	}
}

func (m *Maze) GetRobot() *types.Robot {
	return &m.Robot
}

// IsExit проверяет, находится ли координата в списке выходов.
func (m *Maze) IsExit(x, y, z int) bool {
	for _, e := range m.Exits {
		if e.X == x && e.Y == y && e.Z == z {
			return true
		}
	}
	return false
}

// направление => смещение по координатам
var dirVectors = map[types.Direction]struct{ dx, dy int }{
	types.Up:    {0, 1},  // север (Y+)
	types.Right: {1, 0},  // восток (X+)
	types.Down:  {0, -1}, // юг (Y-)
	types.Left:  {-1, 0}, // запад (X-)
}

// leftOf возвращает направление, расположенное слева от текущего (для 2-D).
func leftOf(d types.Direction) types.Direction {
	switch d {
	case types.Up:
		return types.Left
	case types.Left:
		return types.Down
	case types.Down:
		return types.Right
	case types.Right:
		return types.Up
	default:
		return types.Up
	}
}

// rightOf возвращает направление справа от текущего.
func rightOf(d types.Direction) types.Direction {
	switch d {
	case types.Up:
		return types.Right
	case types.Right:
		return types.Down
	case types.Down:
		return types.Left
	case types.Left:
		return types.Up
	default:
		return types.Right
	}
}

// tryMove проверяет клетку в указанном направлении и делает шаг, если нет препятствия.
func (m *Maze) tryMove(dir types.Direction) bool {
	vec := dirVectors[dir]
	if m.IsObstacle(m.Robot.Position.X+vec.dx, m.Robot.Position.Y+vec.dy, m.Robot.Position.Z) {
		return false
	}
	m.Move(vec.dx, vec.dy, 0)
	return true
}

func (m *Maze) updateBounds(x, y int) {
	if x < m.minX {
		m.minX = x
	}
	if x > m.maxX {
		m.maxX = x
	}
	if y < m.minY {
		m.minY = y
	}
	if y > m.maxY {
		m.maxY = y
	}
}
func (m *Maze) SolveBFS(maxSteps int) bool {
	start := coord{m.Robot.Position.X, m.Robot.Position.Y, m.Robot.Position.Z}
	queue := []coord{start}
	visited := map[coord]bool{start: true}
	prev := map[coord]coord{}

	steps := 0

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		steps++
		if maxSteps > 0 && steps > maxSteps {
			break
		}

		if m.IsExit(current.X, current.Y, current.Z) {
			// восстановить путь (current -> start)
			path := []coord{current}
			for p, ok := prev[current]; ok; p, ok = prev[p] {
				path = append(path, p)
			}
			// пройти путь в прямом порядке (start -> exit)
			for i := len(path) - 1; i >= 0; i-- {
				t := path[i]
				dx := t.X - m.Robot.Position.X
				dy := t.Y - m.Robot.Position.Y
				dz := t.Z - m.Robot.Position.Z
				// корректируем направление по XY, если двигаемся по плоскости
				if dz == 0 {
					for dir, vec := range dirVectors {
						if vec.dx == dx && vec.dy == dy {
							m.Robot.Direction = dir
							break
						}
					}
				}
				m.Move(dx, dy, dz)
			}
			return true
		}

		// 6 соседей
		for _, vec := range []struct{ dx, dy, dz int }{
			{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1},
		} {
			nx, ny, nz := current.X+vec.dx, current.Y+vec.dy, current.Z+vec.dz
			nc := coord{nx, ny, nz}
			if visited[nc] {
				continue
			}
			if m.IsObstacle(nx, ny, nz) {
				continue
			}
			visited[nc] = true
			prev[nc] = current
			queue = append(queue, nc)
		}
	}
	return false
}

// выводит 15×15 клеток вокруг робота
func (m *Maze) Draw() {
	w := bufio.NewWriter(os.Stdout)
	fmt.Fprint(w, "\033[2J\033[H") // ANSI clear

	z := m.Robot.Position.Z
	minX, maxX := m.Robot.Position.X-7, m.Robot.Position.X+7
	minY, maxY := m.Robot.Position.Y-7, m.Robot.Position.Y+7

	for y := maxY; y >= minY; y-- {
		for x := minX; x <= maxX; x++ {
			switch {
			case m.Robot.Position.X == x && m.Robot.Position.Y == y && m.Robot.Position.Z == z:
				fmt.Fprint(w, "R")
			case m.IsExit(x, y, z):
				fmt.Fprint(w, "E")
			case func() bool { c, ok := m.Cells[coord{x, y, z}]; return ok && c.IsObstacle }():
				fmt.Fprint(w, "#")
			default:
				fmt.Fprint(w, ".")
			}
		}
		fmt.Fprint(w, "\n")
	}
	fmt.Fprintf(w, "Z=%d\n", z)
	w.Flush()
}
