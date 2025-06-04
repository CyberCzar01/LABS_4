// ===== cmd/regexviz/main.go =====
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"lab2ta/regexlib"
)

func main() {
	pattern := flag.String("re", "", "pattern (обязателен)")
	dfaFlag := flag.Bool("dfa", true, "export minimal DFA (default)")
	nfaFlag := flag.Bool("nfa", false, "export Thompson NFA")
	rawFlag := flag.Bool("rawdfa", false, "export raw (non-minimized) DFA")
	outFile := flag.String("o", "graph.dot", "output file")
	pngFlag := flag.Bool("png", false, "render PNG via dot -Tpng")
	flag.Parse()

	if *nfaFlag {
		*dfaFlag = false
		*rawFlag = false
	}
	if *rawFlag {
		*dfaFlag = false
	}

	if *pattern == "" {
		fmt.Fprintln(os.Stderr, "usage: regexviz -re <pattern> [-dfa|-nfa|-rawdfa] [-o file] [-png]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	re := regexlib.MustCompile(*pattern)

	var buf bytes.Buffer
	switch {
	case *nfaFlag:
		regexlib.ExportDOT(&buf, re.NFA())
	case *rawFlag:
		regexlib.ExportDOT(&buf, re.RawDFA())
	default:
		regexlib.ExportDOT(&buf, re.DFA())
	}

	if *pngFlag {
		cmd := exec.Command("dot", "-Tpng", "-o", *outFile)
		cmd.Stdin = bytes.NewReader(buf.Bytes())
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "dot failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("PNG written to %s\n", *outFile)
		return
	}

	var w io.Writer
	if *outFile == "-" {
		w = os.Stdout
	} else {
		f, err := os.Create(*outFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot create %s: %v\n", *outFile, err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	_, _ = io.Copy(w, &buf)
	if *outFile != "-" {
		fmt.Printf("DOT written to %s\n", *outFile)
	}
}
