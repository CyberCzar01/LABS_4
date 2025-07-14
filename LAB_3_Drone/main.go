package main

import (
	"drone-maze/evaluator"
	"drone-maze/lexer"
	"drone-maze/maze"
	"drone-maze/object"
	"drone-maze/parser"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func main() {
	var input []byte
	var err error
	var sourceFile, mazeFile string
	var autoSolve bool
	var delayMs int
	isRepl := false

	flag.BoolVar(&autoSolve, "solve", false, "запустить встроенный BFS и выйти")
	flag.IntVar(&delayMs, "delay", 500, "задержка отрисовки шага, мс")
	flag.Parse()

	args := flag.Args()

	switch len(args) {
	case 0:
		isRepl = true
	case 1:
		if strings.HasSuffix(args[0], ".drone") {
			sourceFile = args[0]
		} else {
			mazeFile = args[0]
			isRepl = true
		}
	default:
		sourceFile = args[0]
		mazeFile = args[1]
	}

	if delayMs < 0 {
		fmt.Println("delay must be non-negative")
		return
	}
	maze.RenderDelay = time.Duration(delayMs) * time.Millisecond

	var mazeData *maze.Maze
	if mazeFile != "" {
		data, err := ioutil.ReadFile(mazeFile)
		if err != nil {
			fmt.Printf("Could not read maze file %s: %s\n", mazeFile, err)
			return
		}
		mazeData, err = maze.LoadMaze(data)
		if err != nil {
			fmt.Printf("Error loading maze: %s\n", err)
			return
		}
		evaluator.InitState(mazeData)
	} else {
		defaultMaze := maze.NewMaze()
		evaluator.InitState(defaultMaze)
	}

	if autoSolve {
		mz := evaluator.GetMaze()
		if mz == nil {
			fmt.Println("maze not loaded; use a maze file argument")
			return
		}
		mz.SolveBFS(10000)
		return
	}

	if isRepl {
		fmt.Println("Drone language interpreter. Enter code. Press Ctrl+D to evaluate.")
		if mazeFile != "" {
			fmt.Printf("Maze '%s' loaded.\n", mazeFile)
		}
		input, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println("Error reading from stdin:", err)
			return
		}
	} else {
		input, err = ioutil.ReadFile(sourceFile)
		if err != nil {
			fmt.Printf("Could not read source file %s: %s\n", sourceFile, err)
			return
		}
	}

	l, err := lexer.New(input)
	if err != nil {
		fmt.Printf("Lexer error: %s\n", err)
		return
	}

	program, errors := parser.Parse(l)
	if len(errors) != 0 {
		for _, msg := range errors {
			fmt.Println(msg.String())
		}
		return
	}

	env := object.NewEnvironment()
	evaluated := evaluator.Eval(program, env)

	// выводим финалку лабирнта
	mz := evaluator.GetMaze()
	if mz != nil {
		mz.Draw()
	}

	if evaluated != nil {
		fmt.Println(evaluated.Inspect())
	}
}
