package main

import (
	"fmt"
	"log"
	"os"

	"labyrinth/internal/interpreter"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <labyrinth file> <script file>", os.Args[0])
	}

	lab, err := interpreter.LoadLabyrinth(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	// load the program script from disk
	scriptData, err := os.ReadFile(os.Args[2])
	script, err := os.ReadFile(os.Args[2])

	data, err := os.ReadFile(os.Args[2])

	if err != nil {
		log.Fatal(err)
	}

	prog, err := interpreter.Parse(string(scriptData))
	prog, err := interpreter.Parse(string(script))
	if err != nil {
		log.Fatal(err)
	}

	env := interpreter.NewEnvironment()
	robot := interpreter.NewRobot()
	robot.X, robot.Y, robot.Z = lab.Start[0], lab.Start[1], lab.Start[2]
	ctx := &interpreter.Context{Env: env, Robot: robot, Lab: lab}

	if err := prog.Exec(ctx); err != nil {
		log.Fatal(err)
	}
	x, y, z := robot.Position()
	fmt.Printf("Robot final position: (%d,%d,%d)\n", x, y, z)
}
