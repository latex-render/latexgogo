// Copyright ©2020 The go-latex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package latex // import "github.com/go-latex/latex"

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-latex/latex/ast"
	"github.com/go-latex/latex/token"
)

// ParseExpr parses a simple LaTeX expression.
func ParseExpr(x string) (ast.Node, error) {
	p := newParser(x)
	return p.parse()
}

type state int

const (
	normalState state = iota
	mathState
)

type parser struct {
	s     *texScanner
	state state

	macros map[string]macroParser
}

func newParser(x string) *parser {
	p := &parser{
		s:     newScanner(strings.NewReader(x)),
		state: normalState,
	}
	p.addBuiltinMacros()
	return p
}

func (p *parser) parse() (ast.Node, error) {
	var nodes ast.List
	for p.s.Next() {
		tok := p.s.Token()
		node, err := p.parseNode(tok)
		if err != nil {
			return nil, err
		}
		if node == nil {
			continue
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (p *parser) next() token.Token {
	if !p.s.Next() {
		return token.Token{Kind: token.EOF}
	}
	return p.s.tok
}

func (p *parser) expect(v rune) error {
	p.next()
	if p.s.tok.Text != string(v) {
		return fmt.Errorf("expected %q, got %q", v, p.s.tok.Text)
	}
	return nil
}

func (p *parser) parseNode(tok token.Token) (ast.Node, error) {
	switch tok.Kind {
	case token.Comment:
		return nil, nil
	case token.Macro:
		return p.parseMacro(tok)
	case token.Word:
		return p.parseWord(tok), nil
	case token.Number:
		return p.parseNumber(tok), nil
	case token.Symbol:
		switch tok.Text {
		case "$":
			return p.parseMathExpr(tok)
		case "^":
			return p.parseSup(tok)
		case "_":
			return p.parseSub(tok)
		default:
			return p.parseSymbol(tok), nil
		}
	case token.Lbrace:
		switch p.state {
		case mathState:
			return p.parseMathLbrace(tok)
		default:
			return nil, errors.New("lbrace unMathState not implemented")
		}
	case token.Other:
		switch tok.Text {
		default:
			return nil, errors.New("not implemented: " + tok.String())
		}
	case token.Space:
		switch p.state {
		case mathState:
			return nil, nil
		default:
			return p.parseSymbol(tok), nil
		}

	case token.Lparen, token.Rparen,
		token.Lbrack, token.Rbrack:
		return p.parseSymbol(tok), nil

	default:
		return nil, fmt.Errorf("impossible: %v (%v)", tok, tok.Kind)
	}
}

func (p *parser) parseMathExpr(tok token.Token) (ast.Node, error) {
	state := p.state
	p.state = mathState
	defer func() {
		p.state = state
	}()

	math := &ast.MathExpr{
		Delim: tok.Text,
		Left:  tok.Pos,
	}
	var end string
	switch tok.Text {
	case "$":
		end = "$"
	case `\(`:
		end = `\)`
	case `\[`:
		end = `\]`
	case `\begin`:
		return nil, errors.New("begin not implemeneted")
	default:
		return nil, fmt.Errorf("opening math-expression delimiter %q not supported", tok.Text)
	}

loop:
	for p.s.Next() {
		switch p.s.tok.Text {
		case end:
			math.Right = p.s.tok.Pos
			break loop
		default:
			node, err := p.parseNode(p.s.tok)
			if err != nil {
				return nil, err
			}
			if node == nil {
				continue
			}
			math.List = append(math.List, node)
		}
	}

	return math, nil
}

func (p *parser) parseMacro(tok token.Token) (ast.Node, error) {
	name := tok.Text
	macro, ok := p.macros[name]
	if !ok {
		return nil, errors.New("unknown macro " + name)
		//return nil
	}
	return macro.parseMacro(p), nil
}

func (p *parser) parseWord(tok token.Token) ast.Node {
	return &ast.Word{
		WordPos: tok.Pos,
		Text:    tok.Text,
	}
}

func (p *parser) parseNumber(tok token.Token) ast.Node {
	return &ast.Literal{
		LitPos: tok.Pos,
		Text:   tok.Text,
	}
}

func (p *parser) parseMacroArg(macro *ast.Macro) error {
	var arg ast.Arg
	p.expect('{')
	arg.Lbrace = p.s.tok.Pos

loop:
	for p.s.Next() {
		switch p.s.tok.Kind {
		case token.Rbrace:
			arg.Rbrace = p.s.tok.Pos
			break loop
		default:
			node, err := p.parseNode(p.s.tok)
			if err != nil {
				return err
			}
			if node == nil {
				continue
			}
			arg.List = append(arg.List, node)
		}
	}
	macro.Args = append(macro.Args, &arg)
	return nil
}

func (p *parser) parseOptMacroArg(macro *ast.Macro) error{
	nxt := p.s.sc.Peek()
	if nxt != '[' {
		return nil
	}

	var opt ast.OptArg

	p.expect('[')
	opt.Lbrack = p.s.tok.Pos

loop:
	for p.s.Next() {
		switch p.s.tok.Kind {
		case token.Rbrack:
			opt.Rbrack = p.s.tok.Pos
			break loop
		default:
			node, err := p.parseNode(p.s.tok)
			if err != nil {
				return err
			}
			if node == nil {
				continue
			}
			opt.List = append(opt.List, node)
		}
	}
	macro.Args = append(macro.Args, &opt)
	return nil
}

func (p *parser) parseVerbatimMacroArg(macro *ast.Macro) {
}

func (p *parser) parseSup(tok token.Token) (ast.Node, error) {
	hat := &ast.Sup{
		HatPos: tok.Pos,
	}

	switch next := p.s.sc.Peek(); next {
	case '{':
		p.expect('{')
		var list ast.List
	loop:
		for p.s.Next() {
			switch p.s.tok.Kind {
			case token.Rbrace:
				break loop
			default:
				node, err := p.parseNode(p.s.tok)
				if err != nil {
					return nil, err
				}
				if node == nil {
					continue
				}
				list = append(list, node)
			}
		}
		hat.Node = list
	default:
		var err error
		hat.Node, err = p.parseNode(p.next())
		if err != nil {
			return nil, err
		}
	}

	return hat, nil
}

func (p *parser) parseSub(tok token.Token) (ast.Node, error) {
	sub := &ast.Sub{
		UnderPos: tok.Pos,
	}

	switch next := p.s.sc.Peek(); next {
	case '{':
		p.expect('{')
		var list ast.List
	loop:
		for p.s.Next() {
			switch p.s.tok.Kind {
			case token.Rbrace:
				break loop
			default:
				node, err := p.parseNode(p.s.tok)
				if err != nil {
					return nil, err
				}
				if node == nil {
					continue
				}
				list = append(list, node)
			}
		}
		sub.Node = list
	default:
		var err error
		sub.Node, err = p.parseNode(p.next())
		return nil, err
	}

	return sub, nil
}

func (p *parser) parseSymbol(tok token.Token) ast.Node {
	return &ast.Symbol{
		SymPos: tok.Pos,
		Text:   tok.Text,
	}
}

func (p *parser) parseMathLbrace(tok token.Token) (ast.Node, error) {
	var (
		lst    ast.List
		ldelim = tok.Kind
		rdelim = map[token.Kind]token.Kind{
			token.Lbrace: token.Rbrace,
			token.Lparen: token.Rparen,
		}[ldelim]
	)

	if rdelim == token.Invalid {
		return nil, errors.New("impossible: no matching right-delim for: " + tok.String())
	}

loop:
	for p.s.Next() {
		switch p.s.tok.Kind {
		case rdelim:
			break loop
		default:
			node, err := p.parseNode(p.s.tok)
			if err != nil {
				return nil, err
			}
			if node == nil {
				continue
			}
			lst = append(lst, node)
		}
	}
	return lst, nil
}