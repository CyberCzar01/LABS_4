package main

import (
	"bufio"
	"fmt"
	"os"

	"lab2ta/regexlib"
)

func main() {
	re := regexlib.MustCompile("a(b|c)*d")
	txt := "abcbcd aaaaaacd abbcd"
	fmt.Println("matches:", re.FindAll(txt))

	// визуализируем минимальный DFA
	f, _ := os.Create("dfa.dot")
	defer f.Close()
	regexlib.ExportDOT(f, re.DFA())
	fmt.Println("dfa.dot written (run: dot -Tpng dfa.dot -o dfa.png)")

	// интерактив
	rdr := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("pattern> ")
		pat, _ := rdr.ReadString('\n')
		if len(pat) == 1 {
			break
		}
		pat = pat[:len(pat)-1]
		r, err := regexlib.Compile(pat)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Print("text> ")
		t, _ := rdr.ReadString('\n')
		t = t[:len(t)-1]
		fmt.Println(r.FindAll(t))
	}
}
