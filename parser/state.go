package parser

import (
	"fmt"
	"unicode/utf8"
)

type State struct {
	buffer []byte
	offset int64

	lineCount  int64
	lineOffset int64
}

func (s State) Rune() (rune, State, error) {
	r, rs := utf8.DecodeRune(s.buffer[s.offset:])
	next, err := s.consume(rs)
	return r, next, err
}

func (s State) Byte() (byte, State, error) {
	b := s.buffer[s.offset]
	next, err := s.consume(1)
	return b, next, err
}

func (s State) consume(size int) (State, error) {
	s.offset += int64(size)
	return s, nil
}

func (s State) Position() Position {
	return Position{
		Offset: s.offset,
		Line:   s.Line(),
		Column: s.Column(),
	}
}

func (s State) Line() int64 {
	return s.lineCount + 1
}

func (s State) Column() int64 {
	return s.offset - s.lineOffset
}

type Position struct {
	Offset int64
	Line   int64
	Column int64
}

type PositionError struct {
	Err      error
	Position Position
}

func (e PositionError) Error() string {
	return fmt.Sprintf("parser: error at: %+v, %v", e.Position, e.Err)
}

func (e PositionError) Unwrap() error {
	return e.Err
}
