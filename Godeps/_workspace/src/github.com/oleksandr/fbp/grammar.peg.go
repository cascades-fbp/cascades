package fbp

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const end_symbol rune = 4

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	rulestart
	ruleline
	ruleLineTerminator
	rulecomment
	ruleconnection
	rulebridge
	ruleleftlet
	ruleiip
	rulerightlet
	rulenode
	rulecomponent
	rulecompMeta
	ruleport
	ruleportWithIndex
	ruleanychar
	ruleiipchar
	rule_
	rule__
	rulePegText
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14

	rulePre_
	rule_In_
	rule_Suf
)

var rul3s = [...]string{
	"Unknown",
	"start",
	"line",
	"LineTerminator",
	"comment",
	"connection",
	"bridge",
	"leftlet",
	"iip",
	"rightlet",
	"node",
	"component",
	"compMeta",
	"port",
	"portWithIndex",
	"anychar",
	"iipchar",
	"_",
	"__",
	"PegText",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",

	"Pre_",
	"_In_",
	"_Suf",
}

type tokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule pegRule, begin, end, next, depth int)
	Expand(index int) tokenTree
	Tokens() <-chan token32
	AST() *node32
	Error() []token32
	trim(length int)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(depth int, buffer string) {
	for node != nil {
		for c := 0; c < depth; c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(buffer[node.begin:node.end]))
		if node.up != nil {
			node.up.print(depth+1, buffer)
		}
		node = node.next
	}
}

func (ast *node32) Print(buffer string) {
	ast.print(0, buffer)
}

type element struct {
	node *node32
	down *element
}

/* ${@} bit structure for abstract syntax tree */
type token16 struct {
	pegRule
	begin, end, next int16
}

func (t *token16) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token16) isParentOf(u token16) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token16) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token16) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens16 struct {
	tree    []token16
	ordered [][]token16
}

func (t *tokens16) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens16) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens16) Order() [][]token16 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int16, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token16, len(depths)), make([]token16, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int16(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state16 struct {
	token16
	depths []int16
	leaf   bool
}

func (t *tokens16) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens16) PreOrder() (<-chan state16, [][]token16) {
	s, ordered := make(chan state16, 6), t.Order()
	go func() {
		var states [8]state16
		for i, _ := range states {
			states[i].depths = make([]int16, len(ordered))
		}
		depths, state, depth := make([]int16, len(ordered)), 0, 1
		write := func(t token16, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, int16(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token16 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token16{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token16{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token16{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens16) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens16) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(buffer[token.begin:token.end]))
	}
}

func (t *tokens16) Add(rule pegRule, begin, end, depth, index int) {
	t.tree[index] = token16{pegRule: rule, begin: int16(begin), end: int16(end), next: int16(depth)}
}

func (t *tokens16) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens16) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	pegRule
	begin, end, next int32
}

func (t *token32) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens32) PreOrder() (<-chan state32, [][]token32) {
	s, ordered := make(chan state32, 6), t.Order()
	go func() {
		var states [8]state32
		for i, _ := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, int32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(buffer[token.begin:token.end]))
	}
}

func (t *tokens32) Add(rule pegRule, begin, end, depth, index int) {
	t.tree[index] = token32{pegRule: rule, begin: int32(begin), end: int32(end), next: int32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

func (t *tokens16) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		for i, v := range tree {
			expanded[i] = v.getToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}

func (t *tokens32) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type Fbp struct {
	BaseFbp

	Buffer string
	buffer []rune
	rules  [35]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	tokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer string, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer[0:] {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p *Fbp
}

func (e *parseError) Error() string {
	tokens, error := e.p.tokenTree.Error(), "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.Buffer, positions)
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf("parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n",
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			/*strconv.Quote(*/ e.p.Buffer[begin:end] /*)*/)
	}

	return error
}

func (p *Fbp) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *Fbp) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *Fbp) Execute() {
	buffer, begin, end := p.Buffer, 0, 0
	for token := range p.tokenTree.Tokens() {
		switch token.pegRule {
		case rulePegText:
			begin, end = int(token.begin), int(token.end)
		case ruleAction0:
			p.createInport(buffer[begin:end])
		case ruleAction1:
			p.createOutport(buffer[begin:end])
		case ruleAction2:
			p.inPort = p.port
			p.inPortIndex = p.index
		case ruleAction3:
			p.outPort = p.port
			p.outPortIndex = p.index
		case ruleAction4:
			p.createMiddlet()
		case ruleAction5:
			p.createLeftlet()
		case ruleAction6:
			p.createRightlet()
		case ruleAction7:
			p.iip = buffer[begin:end]
		case ruleAction8:
			p.nodeProcessName = buffer[begin:end]
		case ruleAction9:
			p.createNode()
		case ruleAction10:
			p.nodeComponentName = buffer[begin:end]
		case ruleAction11:
			p.nodeMeta = buffer[begin:end]
		case ruleAction12:
			p.port = buffer[begin:end]
		case ruleAction13:
			p.port = buffer[begin:end]
		case ruleAction14:
			p.index = buffer[begin:end]

		}
	}
}

func (p *Fbp) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != end_symbol {
		p.buffer = append(p.buffer, end_symbol)
	}

	var tree tokenTree = &tokens16{tree: make([]token16, math.MaxInt16)}
	position, depth, tokenIndex, buffer, rules := 0, 0, 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokenTree = tree
		if matches {
			p.tokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule pegRule, begin int) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
	}

	matchDot := func() bool {
		if buffer[position] != end_symbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	rules = [...]func() bool{
		nil,
		/* 0 start <- <(line* _ !.)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
			l2:
				{
					position3, tokenIndex3, depth3 := position, tokenIndex, depth
					{
						position4 := position
						depth++
						{
							position5, tokenIndex5, depth5 := position, tokenIndex, depth
							if !rules[rule_]() {
								goto l6
							}
							{
								position7, tokenIndex7, depth7 := position, tokenIndex, depth
								if buffer[position] != rune('e') {
									goto l8
								}
								position++
								goto l7
							l8:
								position, tokenIndex, depth = position7, tokenIndex7, depth7
								if buffer[position] != rune('E') {
									goto l6
								}
								position++
							}
						l7:
							{
								position9, tokenIndex9, depth9 := position, tokenIndex, depth
								if buffer[position] != rune('x') {
									goto l10
								}
								position++
								goto l9
							l10:
								position, tokenIndex, depth = position9, tokenIndex9, depth9
								if buffer[position] != rune('X') {
									goto l6
								}
								position++
							}
						l9:
							{
								position11, tokenIndex11, depth11 := position, tokenIndex, depth
								if buffer[position] != rune('p') {
									goto l12
								}
								position++
								goto l11
							l12:
								position, tokenIndex, depth = position11, tokenIndex11, depth11
								if buffer[position] != rune('P') {
									goto l6
								}
								position++
							}
						l11:
							{
								position13, tokenIndex13, depth13 := position, tokenIndex, depth
								if buffer[position] != rune('o') {
									goto l14
								}
								position++
								goto l13
							l14:
								position, tokenIndex, depth = position13, tokenIndex13, depth13
								if buffer[position] != rune('O') {
									goto l6
								}
								position++
							}
						l13:
							{
								position15, tokenIndex15, depth15 := position, tokenIndex, depth
								if buffer[position] != rune('r') {
									goto l16
								}
								position++
								goto l15
							l16:
								position, tokenIndex, depth = position15, tokenIndex15, depth15
								if buffer[position] != rune('R') {
									goto l6
								}
								position++
							}
						l15:
							{
								position17, tokenIndex17, depth17 := position, tokenIndex, depth
								if buffer[position] != rune('t') {
									goto l18
								}
								position++
								goto l17
							l18:
								position, tokenIndex, depth = position17, tokenIndex17, depth17
								if buffer[position] != rune('T') {
									goto l6
								}
								position++
							}
						l17:
							if buffer[position] != rune('=') {
								goto l6
							}
							position++
							{
								switch buffer[position] {
								case '_':
									if buffer[position] != rune('_') {
										goto l6
									}
									position++
									break
								case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
									if c := buffer[position]; c < rune('0') || c > rune('9') {
										goto l6
									}
									position++
									break
								case '.':
									if buffer[position] != rune('.') {
										goto l6
									}
									position++
									break
								case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
									if c := buffer[position]; c < rune('a') || c > rune('z') {
										goto l6
									}
									position++
									break
								default:
									if c := buffer[position]; c < rune('A') || c > rune('Z') {
										goto l6
									}
									position++
									break
								}
							}

						l19:
							{
								position20, tokenIndex20, depth20 := position, tokenIndex, depth
								{
									switch buffer[position] {
									case '_':
										if buffer[position] != rune('_') {
											goto l20
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l20
										}
										position++
										break
									case '.':
										if buffer[position] != rune('.') {
											goto l20
										}
										position++
										break
									case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
										if c := buffer[position]; c < rune('a') || c > rune('z') {
											goto l20
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l20
										}
										position++
										break
									}
								}

								goto l19
							l20:
								position, tokenIndex, depth = position20, tokenIndex20, depth20
							}
							if buffer[position] != rune(':') {
								goto l6
							}
							position++
							{
								switch buffer[position] {
								case '_':
									if buffer[position] != rune('_') {
										goto l6
									}
									position++
									break
								case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
									if c := buffer[position]; c < rune('0') || c > rune('9') {
										goto l6
									}
									position++
									break
								default:
									if c := buffer[position]; c < rune('A') || c > rune('Z') {
										goto l6
									}
									position++
									break
								}
							}

						l23:
							{
								position24, tokenIndex24, depth24 := position, tokenIndex, depth
								{
									switch buffer[position] {
									case '_':
										if buffer[position] != rune('_') {
											goto l24
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l24
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l24
										}
										position++
										break
									}
								}

								goto l23
							l24:
								position, tokenIndex, depth = position24, tokenIndex24, depth24
							}
							if !rules[rule_]() {
								goto l6
							}
							{
								position27, tokenIndex27, depth27 := position, tokenIndex, depth
								if !rules[ruleLineTerminator]() {
									goto l27
								}
								goto l28
							l27:
								position, tokenIndex, depth = position27, tokenIndex27, depth27
							}
						l28:
							goto l5
						l6:
							position, tokenIndex, depth = position5, tokenIndex5, depth5
							if !rules[rule_]() {
								goto l29
							}
							{
								position30, tokenIndex30, depth30 := position, tokenIndex, depth
								if buffer[position] != rune('i') {
									goto l31
								}
								position++
								goto l30
							l31:
								position, tokenIndex, depth = position30, tokenIndex30, depth30
								if buffer[position] != rune('I') {
									goto l29
								}
								position++
							}
						l30:
							{
								position32, tokenIndex32, depth32 := position, tokenIndex, depth
								if buffer[position] != rune('n') {
									goto l33
								}
								position++
								goto l32
							l33:
								position, tokenIndex, depth = position32, tokenIndex32, depth32
								if buffer[position] != rune('N') {
									goto l29
								}
								position++
							}
						l32:
							{
								position34, tokenIndex34, depth34 := position, tokenIndex, depth
								if buffer[position] != rune('p') {
									goto l35
								}
								position++
								goto l34
							l35:
								position, tokenIndex, depth = position34, tokenIndex34, depth34
								if buffer[position] != rune('P') {
									goto l29
								}
								position++
							}
						l34:
							{
								position36, tokenIndex36, depth36 := position, tokenIndex, depth
								if buffer[position] != rune('o') {
									goto l37
								}
								position++
								goto l36
							l37:
								position, tokenIndex, depth = position36, tokenIndex36, depth36
								if buffer[position] != rune('O') {
									goto l29
								}
								position++
							}
						l36:
							{
								position38, tokenIndex38, depth38 := position, tokenIndex, depth
								if buffer[position] != rune('r') {
									goto l39
								}
								position++
								goto l38
							l39:
								position, tokenIndex, depth = position38, tokenIndex38, depth38
								if buffer[position] != rune('R') {
									goto l29
								}
								position++
							}
						l38:
							{
								position40, tokenIndex40, depth40 := position, tokenIndex, depth
								if buffer[position] != rune('t') {
									goto l41
								}
								position++
								goto l40
							l41:
								position, tokenIndex, depth = position40, tokenIndex40, depth40
								if buffer[position] != rune('T') {
									goto l29
								}
								position++
							}
						l40:
							if buffer[position] != rune('=') {
								goto l29
							}
							position++
							{
								position42 := position
								depth++
								{
									switch buffer[position] {
									case '_':
										if buffer[position] != rune('_') {
											goto l29
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l29
										}
										position++
										break
									case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
										if c := buffer[position]; c < rune('a') || c > rune('z') {
											goto l29
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l29
										}
										position++
										break
									}
								}

							l43:
								{
									position44, tokenIndex44, depth44 := position, tokenIndex, depth
									{
										switch buffer[position] {
										case '_':
											if buffer[position] != rune('_') {
												goto l44
											}
											position++
											break
										case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l44
											}
											position++
											break
										case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
											if c := buffer[position]; c < rune('a') || c > rune('z') {
												goto l44
											}
											position++
											break
										default:
											if c := buffer[position]; c < rune('A') || c > rune('Z') {
												goto l44
											}
											position++
											break
										}
									}

									goto l43
								l44:
									position, tokenIndex, depth = position44, tokenIndex44, depth44
								}
								if buffer[position] != rune('.') {
									goto l29
								}
								position++
								{
									switch buffer[position] {
									case ']':
										if buffer[position] != rune(']') {
											goto l29
										}
										position++
										break
									case '[':
										if buffer[position] != rune('[') {
											goto l29
										}
										position++
										break
									case '_':
										if buffer[position] != rune('_') {
											goto l29
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l29
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l29
										}
										position++
										break
									}
								}

							l47:
								{
									position48, tokenIndex48, depth48 := position, tokenIndex, depth
									{
										switch buffer[position] {
										case ']':
											if buffer[position] != rune(']') {
												goto l48
											}
											position++
											break
										case '[':
											if buffer[position] != rune('[') {
												goto l48
											}
											position++
											break
										case '_':
											if buffer[position] != rune('_') {
												goto l48
											}
											position++
											break
										case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l48
											}
											position++
											break
										default:
											if c := buffer[position]; c < rune('A') || c > rune('Z') {
												goto l48
											}
											position++
											break
										}
									}

									goto l47
								l48:
									position, tokenIndex, depth = position48, tokenIndex48, depth48
								}
								if buffer[position] != rune(':') {
									goto l29
								}
								position++
								{
									switch buffer[position] {
									case '_':
										if buffer[position] != rune('_') {
											goto l29
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l29
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l29
										}
										position++
										break
									}
								}

							l51:
								{
									position52, tokenIndex52, depth52 := position, tokenIndex, depth
									{
										switch buffer[position] {
										case '_':
											if buffer[position] != rune('_') {
												goto l52
											}
											position++
											break
										case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l52
											}
											position++
											break
										default:
											if c := buffer[position]; c < rune('A') || c > rune('Z') {
												goto l52
											}
											position++
											break
										}
									}

									goto l51
								l52:
									position, tokenIndex, depth = position52, tokenIndex52, depth52
								}
								depth--
								add(rulePegText, position42)
							}
							if !rules[rule_]() {
								goto l29
							}
							{
								position55, tokenIndex55, depth55 := position, tokenIndex, depth
								if !rules[ruleLineTerminator]() {
									goto l55
								}
								goto l56
							l55:
								position, tokenIndex, depth = position55, tokenIndex55, depth55
							}
						l56:
							{
								add(ruleAction0, position)
							}
							goto l5
						l29:
							position, tokenIndex, depth = position5, tokenIndex5, depth5
							if !rules[rule_]() {
								goto l58
							}
							{
								position59, tokenIndex59, depth59 := position, tokenIndex, depth
								if buffer[position] != rune('o') {
									goto l60
								}
								position++
								goto l59
							l60:
								position, tokenIndex, depth = position59, tokenIndex59, depth59
								if buffer[position] != rune('O') {
									goto l58
								}
								position++
							}
						l59:
							{
								position61, tokenIndex61, depth61 := position, tokenIndex, depth
								if buffer[position] != rune('u') {
									goto l62
								}
								position++
								goto l61
							l62:
								position, tokenIndex, depth = position61, tokenIndex61, depth61
								if buffer[position] != rune('U') {
									goto l58
								}
								position++
							}
						l61:
							{
								position63, tokenIndex63, depth63 := position, tokenIndex, depth
								if buffer[position] != rune('t') {
									goto l64
								}
								position++
								goto l63
							l64:
								position, tokenIndex, depth = position63, tokenIndex63, depth63
								if buffer[position] != rune('T') {
									goto l58
								}
								position++
							}
						l63:
							{
								position65, tokenIndex65, depth65 := position, tokenIndex, depth
								if buffer[position] != rune('p') {
									goto l66
								}
								position++
								goto l65
							l66:
								position, tokenIndex, depth = position65, tokenIndex65, depth65
								if buffer[position] != rune('P') {
									goto l58
								}
								position++
							}
						l65:
							{
								position67, tokenIndex67, depth67 := position, tokenIndex, depth
								if buffer[position] != rune('o') {
									goto l68
								}
								position++
								goto l67
							l68:
								position, tokenIndex, depth = position67, tokenIndex67, depth67
								if buffer[position] != rune('O') {
									goto l58
								}
								position++
							}
						l67:
							{
								position69, tokenIndex69, depth69 := position, tokenIndex, depth
								if buffer[position] != rune('r') {
									goto l70
								}
								position++
								goto l69
							l70:
								position, tokenIndex, depth = position69, tokenIndex69, depth69
								if buffer[position] != rune('R') {
									goto l58
								}
								position++
							}
						l69:
							{
								position71, tokenIndex71, depth71 := position, tokenIndex, depth
								if buffer[position] != rune('t') {
									goto l72
								}
								position++
								goto l71
							l72:
								position, tokenIndex, depth = position71, tokenIndex71, depth71
								if buffer[position] != rune('T') {
									goto l58
								}
								position++
							}
						l71:
							if buffer[position] != rune('=') {
								goto l58
							}
							position++
							{
								position73 := position
								depth++
								{
									switch buffer[position] {
									case '_':
										if buffer[position] != rune('_') {
											goto l58
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l58
										}
										position++
										break
									case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
										if c := buffer[position]; c < rune('a') || c > rune('z') {
											goto l58
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l58
										}
										position++
										break
									}
								}

							l74:
								{
									position75, tokenIndex75, depth75 := position, tokenIndex, depth
									{
										switch buffer[position] {
										case '_':
											if buffer[position] != rune('_') {
												goto l75
											}
											position++
											break
										case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l75
											}
											position++
											break
										case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
											if c := buffer[position]; c < rune('a') || c > rune('z') {
												goto l75
											}
											position++
											break
										default:
											if c := buffer[position]; c < rune('A') || c > rune('Z') {
												goto l75
											}
											position++
											break
										}
									}

									goto l74
								l75:
									position, tokenIndex, depth = position75, tokenIndex75, depth75
								}
								if buffer[position] != rune('.') {
									goto l58
								}
								position++
								{
									switch buffer[position] {
									case ']':
										if buffer[position] != rune(']') {
											goto l58
										}
										position++
										break
									case '[':
										if buffer[position] != rune('[') {
											goto l58
										}
										position++
										break
									case '_':
										if buffer[position] != rune('_') {
											goto l58
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l58
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l58
										}
										position++
										break
									}
								}

							l78:
								{
									position79, tokenIndex79, depth79 := position, tokenIndex, depth
									{
										switch buffer[position] {
										case ']':
											if buffer[position] != rune(']') {
												goto l79
											}
											position++
											break
										case '[':
											if buffer[position] != rune('[') {
												goto l79
											}
											position++
											break
										case '_':
											if buffer[position] != rune('_') {
												goto l79
											}
											position++
											break
										case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l79
											}
											position++
											break
										default:
											if c := buffer[position]; c < rune('A') || c > rune('Z') {
												goto l79
											}
											position++
											break
										}
									}

									goto l78
								l79:
									position, tokenIndex, depth = position79, tokenIndex79, depth79
								}
								if buffer[position] != rune(':') {
									goto l58
								}
								position++
								{
									switch buffer[position] {
									case '_':
										if buffer[position] != rune('_') {
											goto l58
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l58
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l58
										}
										position++
										break
									}
								}

							l82:
								{
									position83, tokenIndex83, depth83 := position, tokenIndex, depth
									{
										switch buffer[position] {
										case '_':
											if buffer[position] != rune('_') {
												goto l83
											}
											position++
											break
										case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l83
											}
											position++
											break
										default:
											if c := buffer[position]; c < rune('A') || c > rune('Z') {
												goto l83
											}
											position++
											break
										}
									}

									goto l82
								l83:
									position, tokenIndex, depth = position83, tokenIndex83, depth83
								}
								depth--
								add(rulePegText, position73)
							}
							if !rules[rule_]() {
								goto l58
							}
							{
								position86, tokenIndex86, depth86 := position, tokenIndex, depth
								if !rules[ruleLineTerminator]() {
									goto l86
								}
								goto l87
							l86:
								position, tokenIndex, depth = position86, tokenIndex86, depth86
							}
						l87:
							{
								add(ruleAction1, position)
							}
							goto l5
						l58:
							position, tokenIndex, depth = position5, tokenIndex5, depth5
							if !rules[rulecomment]() {
								goto l89
							}
							{
								position90, tokenIndex90, depth90 := position, tokenIndex, depth
								{
									position92, tokenIndex92, depth92 := position, tokenIndex, depth
									if buffer[position] != rune('\n') {
										goto l93
									}
									position++
									goto l92
								l93:
									position, tokenIndex, depth = position92, tokenIndex92, depth92
									if buffer[position] != rune('\r') {
										goto l90
									}
									position++
								}
							l92:
								goto l91
							l90:
								position, tokenIndex, depth = position90, tokenIndex90, depth90
							}
						l91:
							goto l5
						l89:
							position, tokenIndex, depth = position5, tokenIndex5, depth5
							if !rules[rule_]() {
								goto l94
							}
							{
								position95, tokenIndex95, depth95 := position, tokenIndex, depth
								if buffer[position] != rune('\n') {
									goto l96
								}
								position++
								goto l95
							l96:
								position, tokenIndex, depth = position95, tokenIndex95, depth95
								if buffer[position] != rune('\r') {
									goto l94
								}
								position++
							}
						l95:
							goto l5
						l94:
							position, tokenIndex, depth = position5, tokenIndex5, depth5
							if !rules[rule_]() {
								goto l3
							}
							if !rules[ruleconnection]() {
								goto l3
							}
							if !rules[rule_]() {
								goto l3
							}
							{
								position97, tokenIndex97, depth97 := position, tokenIndex, depth
								if !rules[ruleLineTerminator]() {
									goto l97
								}
								goto l98
							l97:
								position, tokenIndex, depth = position97, tokenIndex97, depth97
							}
						l98:
						}
					l5:
						depth--
						add(ruleline, position4)
					}
					goto l2
				l3:
					position, tokenIndex, depth = position3, tokenIndex3, depth3
				}
				if !rules[rule_]() {
					goto l0
				}
				{
					position99, tokenIndex99, depth99 := position, tokenIndex, depth
					if !matchDot() {
						goto l99
					}
					goto l0
				l99:
					position, tokenIndex, depth = position99, tokenIndex99, depth99
				}
				depth--
				add(rulestart, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 line <- <((_ (('e' / 'E') ('x' / 'X') ('p' / 'P') ('o' / 'O') ('r' / 'R') ('t' / 'T') '=') ((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('.') '.') | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+ ':' ((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+ _ LineTerminator?) / (_ (('i' / 'I') ('n' / 'N') ('p' / 'P') ('o' / 'O') ('r' / 'R') ('t' / 'T') '=') <(((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+ '.' ((&(']') ']') | (&('[') '[') | (&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+ ':' ((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+)> _ LineTerminator? Action0) / (_ (('o' / 'O') ('u' / 'U') ('t' / 'T') ('p' / 'P') ('o' / 'O') ('r' / 'R') ('t' / 'T') '=') <(((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+ '.' ((&(']') ']') | (&('[') '[') | (&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+ ':' ((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+)> _ LineTerminator? Action1) / (comment ('\n' / '\r')?) / (_ ('\n' / '\r')) / (_ connection _ LineTerminator?))> */
		nil,
		/* 2 LineTerminator <- <(_ ','? comment? ('\n' / '\r')?)> */
		func() bool {
			position101, tokenIndex101, depth101 := position, tokenIndex, depth
			{
				position102 := position
				depth++
				if !rules[rule_]() {
					goto l101
				}
				{
					position103, tokenIndex103, depth103 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l103
					}
					position++
					goto l104
				l103:
					position, tokenIndex, depth = position103, tokenIndex103, depth103
				}
			l104:
				{
					position105, tokenIndex105, depth105 := position, tokenIndex, depth
					if !rules[rulecomment]() {
						goto l105
					}
					goto l106
				l105:
					position, tokenIndex, depth = position105, tokenIndex105, depth105
				}
			l106:
				{
					position107, tokenIndex107, depth107 := position, tokenIndex, depth
					{
						position109, tokenIndex109, depth109 := position, tokenIndex, depth
						if buffer[position] != rune('\n') {
							goto l110
						}
						position++
						goto l109
					l110:
						position, tokenIndex, depth = position109, tokenIndex109, depth109
						if buffer[position] != rune('\r') {
							goto l107
						}
						position++
					}
				l109:
					goto l108
				l107:
					position, tokenIndex, depth = position107, tokenIndex107, depth107
				}
			l108:
				depth--
				add(ruleLineTerminator, position102)
			}
			return true
		l101:
			position, tokenIndex, depth = position101, tokenIndex101, depth101
			return false
		},
		/* 3 comment <- <(_ '#' anychar*)> */
		func() bool {
			position111, tokenIndex111, depth111 := position, tokenIndex, depth
			{
				position112 := position
				depth++
				if !rules[rule_]() {
					goto l111
				}
				if buffer[position] != rune('#') {
					goto l111
				}
				position++
			l113:
				{
					position114, tokenIndex114, depth114 := position, tokenIndex, depth
					{
						position115 := position
						depth++
						{
							position116, tokenIndex116, depth116 := position, tokenIndex, depth
							{
								position117, tokenIndex117, depth117 := position, tokenIndex, depth
								if buffer[position] != rune('\n') {
									goto l118
								}
								position++
								goto l117
							l118:
								position, tokenIndex, depth = position117, tokenIndex117, depth117
								if buffer[position] != rune('\r') {
									goto l116
								}
								position++
							}
						l117:
							goto l114
						l116:
							position, tokenIndex, depth = position116, tokenIndex116, depth116
						}
						if !matchDot() {
							goto l114
						}
						depth--
						add(ruleanychar, position115)
					}
					goto l113
				l114:
					position, tokenIndex, depth = position114, tokenIndex114, depth114
				}
				depth--
				add(rulecomment, position112)
			}
			return true
		l111:
			position, tokenIndex, depth = position111, tokenIndex111, depth111
			return false
		},
		/* 4 connection <- <((bridge _ ('-' '>') _ connection) / bridge)> */
		func() bool {
			position119, tokenIndex119, depth119 := position, tokenIndex, depth
			{
				position120 := position
				depth++
				{
					position121, tokenIndex121, depth121 := position, tokenIndex, depth
					if !rules[rulebridge]() {
						goto l122
					}
					if !rules[rule_]() {
						goto l122
					}
					if buffer[position] != rune('-') {
						goto l122
					}
					position++
					if buffer[position] != rune('>') {
						goto l122
					}
					position++
					if !rules[rule_]() {
						goto l122
					}
					if !rules[ruleconnection]() {
						goto l122
					}
					goto l121
				l122:
					position, tokenIndex, depth = position121, tokenIndex121, depth121
					if !rules[rulebridge]() {
						goto l119
					}
				}
			l121:
				depth--
				add(ruleconnection, position120)
			}
			return true
		l119:
			position, tokenIndex, depth = position119, tokenIndex119, depth119
			return false
		},
		/* 5 bridge <- <((port _ Action2 node _ port Action3 Action4) / iip / (leftlet Action5) / (rightlet Action6))> */
		func() bool {
			position123, tokenIndex123, depth123 := position, tokenIndex, depth
			{
				position124 := position
				depth++
				{
					position125, tokenIndex125, depth125 := position, tokenIndex, depth
					if !rules[ruleport]() {
						goto l126
					}
					if !rules[rule_]() {
						goto l126
					}
					{
						add(ruleAction2, position)
					}
					if !rules[rulenode]() {
						goto l126
					}
					if !rules[rule_]() {
						goto l126
					}
					if !rules[ruleport]() {
						goto l126
					}
					{
						add(ruleAction3, position)
					}
					{
						add(ruleAction4, position)
					}
					goto l125
				l126:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					{
						position131 := position
						depth++
						if buffer[position] != rune('\'') {
							goto l130
						}
						position++
						{
							position132 := position
							depth++
						l133:
							{
								position134, tokenIndex134, depth134 := position, tokenIndex, depth
								{
									position135 := position
									depth++
									{
										position136, tokenIndex136, depth136 := position, tokenIndex, depth
										if buffer[position] != rune('\\') {
											goto l137
										}
										position++
										if buffer[position] != rune('\'') {
											goto l137
										}
										position++
										goto l136
									l137:
										position, tokenIndex, depth = position136, tokenIndex136, depth136
										{
											position138, tokenIndex138, depth138 := position, tokenIndex, depth
											if buffer[position] != rune('\'') {
												goto l138
											}
											position++
											goto l134
										l138:
											position, tokenIndex, depth = position138, tokenIndex138, depth138
										}
										if !matchDot() {
											goto l134
										}
									}
								l136:
									depth--
									add(ruleiipchar, position135)
								}
								goto l133
							l134:
								position, tokenIndex, depth = position134, tokenIndex134, depth134
							}
							depth--
							add(rulePegText, position132)
						}
						if buffer[position] != rune('\'') {
							goto l130
						}
						position++
						{
							add(ruleAction7, position)
						}
						depth--
						add(ruleiip, position131)
					}
					goto l125
				l130:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					{
						position141 := position
						depth++
						{
							position142, tokenIndex142, depth142 := position, tokenIndex, depth
							if !rules[rulenode]() {
								goto l143
							}
							if !rules[rule_]() {
								goto l143
							}
							if !rules[ruleportWithIndex]() {
								goto l143
							}
							goto l142
						l143:
							position, tokenIndex, depth = position142, tokenIndex142, depth142
							if !rules[rulenode]() {
								goto l140
							}
							if !rules[rule_]() {
								goto l140
							}
							if !rules[ruleport]() {
								goto l140
							}
						}
					l142:
						depth--
						add(ruleleftlet, position141)
					}
					{
						add(ruleAction5, position)
					}
					goto l125
				l140:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					{
						position145 := position
						depth++
						{
							position146, tokenIndex146, depth146 := position, tokenIndex, depth
							if !rules[ruleportWithIndex]() {
								goto l147
							}
							if !rules[rule_]() {
								goto l147
							}
							if !rules[rulenode]() {
								goto l147
							}
							goto l146
						l147:
							position, tokenIndex, depth = position146, tokenIndex146, depth146
							if !rules[ruleport]() {
								goto l123
							}
							if !rules[rule_]() {
								goto l123
							}
							if !rules[rulenode]() {
								goto l123
							}
						}
					l146:
						depth--
						add(rulerightlet, position145)
					}
					{
						add(ruleAction6, position)
					}
				}
			l125:
				depth--
				add(rulebridge, position124)
			}
			return true
		l123:
			position, tokenIndex, depth = position123, tokenIndex123, depth123
			return false
		},
		/* 6 leftlet <- <((node _ portWithIndex) / (node _ port))> */
		nil,
		/* 7 iip <- <('\'' <iipchar*> '\'' Action7)> */
		nil,
		/* 8 rightlet <- <((portWithIndex _ node) / (port _ node))> */
		nil,
		/* 9 node <- <(<((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> Action8 component? Action9)> */
		func() bool {
			position152, tokenIndex152, depth152 := position, tokenIndex, depth
			{
				position153 := position
				depth++
				{
					position154 := position
					depth++
					{
						switch buffer[position] {
						case '_':
							if buffer[position] != rune('_') {
								goto l152
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l152
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l152
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l152
							}
							position++
							break
						}
					}

				l155:
					{
						position156, tokenIndex156, depth156 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '_':
								if buffer[position] != rune('_') {
									goto l156
								}
								position++
								break
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l156
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l156
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l156
								}
								position++
								break
							}
						}

						goto l155
					l156:
						position, tokenIndex, depth = position156, tokenIndex156, depth156
					}
					depth--
					add(rulePegText, position154)
				}
				{
					add(ruleAction8, position)
				}
				{
					position160, tokenIndex160, depth160 := position, tokenIndex, depth
					{
						position162 := position
						depth++
						if buffer[position] != rune('(') {
							goto l160
						}
						position++
						{
							position163 := position
							depth++
						l164:
							{
								position165, tokenIndex165, depth165 := position, tokenIndex, depth
								{
									switch buffer[position] {
									case '_':
										if buffer[position] != rune('_') {
											goto l165
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l165
										}
										position++
										break
									case '-':
										if buffer[position] != rune('-') {
											goto l165
										}
										position++
										break
									case '/':
										if buffer[position] != rune('/') {
											goto l165
										}
										position++
										break
									case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l165
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('a') || c > rune('z') {
											goto l165
										}
										position++
										break
									}
								}

								goto l164
							l165:
								position, tokenIndex, depth = position165, tokenIndex165, depth165
							}
							depth--
							add(rulePegText, position163)
						}
						{
							add(ruleAction10, position)
						}
						{
							position168, tokenIndex168, depth168 := position, tokenIndex, depth
							{
								position170 := position
								depth++
								if buffer[position] != rune(':') {
									goto l168
								}
								position++
								{
									position171 := position
									depth++
									{
										switch buffer[position] {
										case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
											if c := buffer[position]; c < rune('0') || c > rune('9') {
												goto l168
											}
											position++
											break
										case ',':
											if buffer[position] != rune(',') {
												goto l168
											}
											position++
											break
										case '_':
											if buffer[position] != rune('_') {
												goto l168
											}
											position++
											break
										case '=':
											if buffer[position] != rune('=') {
												goto l168
											}
											position++
											break
										case '/':
											if buffer[position] != rune('/') {
												goto l168
											}
											position++
											break
										case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
											if c := buffer[position]; c < rune('A') || c > rune('Z') {
												goto l168
											}
											position++
											break
										default:
											if c := buffer[position]; c < rune('a') || c > rune('z') {
												goto l168
											}
											position++
											break
										}
									}

								l172:
									{
										position173, tokenIndex173, depth173 := position, tokenIndex, depth
										{
											switch buffer[position] {
											case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
												if c := buffer[position]; c < rune('0') || c > rune('9') {
													goto l173
												}
												position++
												break
											case ',':
												if buffer[position] != rune(',') {
													goto l173
												}
												position++
												break
											case '_':
												if buffer[position] != rune('_') {
													goto l173
												}
												position++
												break
											case '=':
												if buffer[position] != rune('=') {
													goto l173
												}
												position++
												break
											case '/':
												if buffer[position] != rune('/') {
													goto l173
												}
												position++
												break
											case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
												if c := buffer[position]; c < rune('A') || c > rune('Z') {
													goto l173
												}
												position++
												break
											default:
												if c := buffer[position]; c < rune('a') || c > rune('z') {
													goto l173
												}
												position++
												break
											}
										}

										goto l172
									l173:
										position, tokenIndex, depth = position173, tokenIndex173, depth173
									}
									depth--
									add(rulePegText, position171)
								}
								{
									add(ruleAction11, position)
								}
								depth--
								add(rulecompMeta, position170)
							}
							goto l169
						l168:
							position, tokenIndex, depth = position168, tokenIndex168, depth168
						}
					l169:
						if buffer[position] != rune(')') {
							goto l160
						}
						position++
						depth--
						add(rulecomponent, position162)
					}
					goto l161
				l160:
					position, tokenIndex, depth = position160, tokenIndex160, depth160
				}
			l161:
				{
					add(ruleAction9, position)
				}
				depth--
				add(rulenode, position153)
			}
			return true
		l152:
			position, tokenIndex, depth = position152, tokenIndex152, depth152
			return false
		},
		/* 10 component <- <('(' <((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('-') '-') | (&('/') '/') | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))*> Action10 compMeta? ')')> */
		nil,
		/* 11 compMeta <- <(':' <((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&(',') ',') | (&('_') '_') | (&('=') '=') | (&('/') '/') | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+> Action11)> */
		nil,
		/* 12 port <- <(<((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('.') '.') | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+> __ Action12)> */
		func() bool {
			position180, tokenIndex180, depth180 := position, tokenIndex, depth
			{
				position181 := position
				depth++
				{
					position182 := position
					depth++
					{
						switch buffer[position] {
						case '_':
							if buffer[position] != rune('_') {
								goto l180
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l180
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l180
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l180
							}
							position++
							break
						}
					}

				l183:
					{
						position184, tokenIndex184, depth184 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '_':
								if buffer[position] != rune('_') {
									goto l184
								}
								position++
								break
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l184
								}
								position++
								break
							case '.':
								if buffer[position] != rune('.') {
									goto l184
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l184
								}
								position++
								break
							}
						}

						goto l183
					l184:
						position, tokenIndex, depth = position184, tokenIndex184, depth184
					}
					depth--
					add(rulePegText, position182)
				}
				if !rules[rule__]() {
					goto l180
				}
				{
					add(ruleAction12, position)
				}
				depth--
				add(ruleport, position181)
			}
			return true
		l180:
			position, tokenIndex, depth = position180, tokenIndex180, depth180
			return false
		},
		/* 13 portWithIndex <- <(<((&('_') '_') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('.') '.') | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+> Action13 '[' <[0-9]+> Action14 ']' __)> */
		func() bool {
			position188, tokenIndex188, depth188 := position, tokenIndex, depth
			{
				position189 := position
				depth++
				{
					position190 := position
					depth++
					{
						switch buffer[position] {
						case '_':
							if buffer[position] != rune('_') {
								goto l188
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l188
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l188
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l188
							}
							position++
							break
						}
					}

				l191:
					{
						position192, tokenIndex192, depth192 := position, tokenIndex, depth
						{
							switch buffer[position] {
							case '_':
								if buffer[position] != rune('_') {
									goto l192
								}
								position++
								break
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l192
								}
								position++
								break
							case '.':
								if buffer[position] != rune('.') {
									goto l192
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l192
								}
								position++
								break
							}
						}

						goto l191
					l192:
						position, tokenIndex, depth = position192, tokenIndex192, depth192
					}
					depth--
					add(rulePegText, position190)
				}
				{
					add(ruleAction13, position)
				}
				if buffer[position] != rune('[') {
					goto l188
				}
				position++
				{
					position196 := position
					depth++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l188
					}
					position++
				l197:
					{
						position198, tokenIndex198, depth198 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l198
						}
						position++
						goto l197
					l198:
						position, tokenIndex, depth = position198, tokenIndex198, depth198
					}
					depth--
					add(rulePegText, position196)
				}
				{
					add(ruleAction14, position)
				}
				if buffer[position] != rune(']') {
					goto l188
				}
				position++
				if !rules[rule__]() {
					goto l188
				}
				depth--
				add(ruleportWithIndex, position189)
			}
			return true
		l188:
			position, tokenIndex, depth = position188, tokenIndex188, depth188
			return false
		},
		/* 14 anychar <- <(!('\n' / '\r') .)> */
		nil,
		/* 15 iipchar <- <(('\\' '\'') / (!'\'' .))> */
		nil,
		/* 16 _ <- <(' ' / '\t')*> */
		func() bool {
			{
				position203 := position
				depth++
			l204:
				{
					position205, tokenIndex205, depth205 := position, tokenIndex, depth
					{
						position206, tokenIndex206, depth206 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l207
						}
						position++
						goto l206
					l207:
						position, tokenIndex, depth = position206, tokenIndex206, depth206
						if buffer[position] != rune('\t') {
							goto l205
						}
						position++
					}
				l206:
					goto l204
				l205:
					position, tokenIndex, depth = position205, tokenIndex205, depth205
				}
				depth--
				add(rule_, position203)
			}
			return true
		},
		/* 17 __ <- <(' ' / '\t')+> */
		func() bool {
			position208, tokenIndex208, depth208 := position, tokenIndex, depth
			{
				position209 := position
				depth++
				{
					position212, tokenIndex212, depth212 := position, tokenIndex, depth
					if buffer[position] != rune(' ') {
						goto l213
					}
					position++
					goto l212
				l213:
					position, tokenIndex, depth = position212, tokenIndex212, depth212
					if buffer[position] != rune('\t') {
						goto l208
					}
					position++
				}
			l212:
			l210:
				{
					position211, tokenIndex211, depth211 := position, tokenIndex, depth
					{
						position214, tokenIndex214, depth214 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l215
						}
						position++
						goto l214
					l215:
						position, tokenIndex, depth = position214, tokenIndex214, depth214
						if buffer[position] != rune('\t') {
							goto l211
						}
						position++
					}
				l214:
					goto l210
				l211:
					position, tokenIndex, depth = position211, tokenIndex211, depth211
				}
				depth--
				add(rule__, position209)
			}
			return true
		l208:
			position, tokenIndex, depth = position208, tokenIndex208, depth208
			return false
		},
		nil,
		/* 20 Action0 <- <{ p.createInport(buffer[begin:end]) }> */
		nil,
		/* 21 Action1 <- <{ p.createOutport(buffer[begin:end]) }> */
		nil,
		/* 22 Action2 <- <{ p.inPort = p.port; p.inPortIndex = p.index }> */
		nil,
		/* 23 Action3 <- <{ p.outPort = p.port; p.outPortIndex = p.index }> */
		nil,
		/* 24 Action4 <- <{ p.createMiddlet() }> */
		nil,
		/* 25 Action5 <- <{ p.createLeftlet() }> */
		nil,
		/* 26 Action6 <- <{ p.createRightlet() }> */
		nil,
		/* 27 Action7 <- <{ p.iip = buffer[begin:end] }> */
		nil,
		/* 28 Action8 <- <{ p.nodeProcessName = buffer[begin:end] }> */
		nil,
		/* 29 Action9 <- <{ p.createNode() }> */
		nil,
		/* 30 Action10 <- <{ p.nodeComponentName = buffer[begin:end] }> */
		nil,
		/* 31 Action11 <- <{ p.nodeMeta = buffer[begin:end] }> */
		nil,
		/* 32 Action12 <- <{ p.port = buffer[begin:end] }> */
		nil,
		/* 33 Action13 <- <{ p.port = buffer[begin:end] }> */
		nil,
		/* 34 Action14 <- <{ p.index = buffer[begin:end] }> */
		nil,
	}
	p.rules = rules
}
