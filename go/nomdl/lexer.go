// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nomdl

import (
	"fmt"
	"text/scanner"
)

type token struct {
	Pos  scanner.Position
	Type rune
	Text string
}

func (t token) String() string {
	return t.Text
}

type lexer struct {
	scanner   *scanner.Scanner
	nextToken token
}

func new(s *scanner.Scanner) *lexer {
	rv := &lexer{scanner: s}
	rv.next()
	return rv
}

func (lex *lexer) next() token {
	nextToken := lex.nextToken
	r := lex.scanner.Scan()
	lex.nextToken = token{
		Pos:  lex.scanner.Pos(),
		Text: lex.scanner.TokenText(),
		Type: r,
	}
	return nextToken
}

func (lex *lexer) peek() token {
	return lex.nextToken
}

func (lex *lexer) eat(expected rune) token {
	tok := lex.next()
	lex.check(expected, tok)
	return tok
}

func (lex *lexer) eatIf(expected rune) bool {
	tok := lex.peek()
	if tok.Type == expected {
		lex.next()
		return true
	}
	return false
}

func (lex *lexer) check(expected rune, actual token) {
	if actual.Type != expected {
		lex.tokenMismatch(expected, actual)
	}
}

func (lex *lexer) tokenMismatch(expected rune, actual token) {
	raiseSyntaxError(fmt.Sprintf("Unexpected token %s, expected %s", scanner.TokenString(actual.Type), scanner.TokenString(expected)), actual.Pos)
}

func (lex *lexer) unexpectedToken(tok token) {
	raiseSyntaxError(fmt.Sprintf("Unexpected token %s", scanner.TokenString(tok.Type)), tok.Pos)
}

func raiseSyntaxError(msg string, pos scanner.Position) {
	panic(syntaxError{
		msg: msg,
		pos: pos,
	})
}

type syntaxError struct {
	msg string
	pos scanner.Position
}

func (e syntaxError) Error() string {
	return fmt.Sprintf("%s, %s", e.msg, e.pos)
}

func catchSyntaxError(f func()) (errRes error) {
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(syntaxError); ok {
				errRes = err
				return
			}
			panic(err)
		}
	}()

	f()
	return
}
