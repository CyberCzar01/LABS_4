package interpreter

import (
	"fmt"
	"time"
)

// Display shows the current Z layer of the labyrinth with the robot position.
func (l *Labyrinth) Display(r *Robot) {
	if r.Z < 0 || r.Z >= len(l.Grid) {
		return
	}
	fmt.Print("\033[H\033[2J")
	fmt.Printf("Layer Z=%d\n", r.Z)
	for y := 0; y < len(l.Grid[0]); y++ {
		for x := 0; x < len(l.Grid[0][0]); x++ {
			if r.X == x && r.Y == y {
				fmt.Print("R ")
			} else if l.Grid[r.Z][y][x] == 1 {
				fmt.Print("# ")
			} else {
				fmt.Print(". ")
			}
		}
		fmt.Println()
	}
	time.Sleep(200 * time.Millisecond)
}
