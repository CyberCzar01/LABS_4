package regexlib

type nodeType int

const (
	nEmpty nodeType = iota // ε
	nChar
	nConcat
	nUnion
	nStar
	nPlus
	nQMark
	nRepeat  // {m,n}
	nSet     // character class
	nGroup   // ( ... )
	nBackRef // \1 etc
)

type astNode struct {
	typ   nodeType
	left  *astNode
	right *astNode

	ch       rune   // for nChar
	charset  []rune // for nSet
	min, max int    // for nRepeat
	grpNum   int    // group number or backref target
}

func charNode(r rune) *astNode { return &astNode{typ: nChar, ch: r} }
