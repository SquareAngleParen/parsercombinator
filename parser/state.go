package parser

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"unicode/utf8"
)

type RuneError struct {
	Position
}

func (e RuneError) Error() string {
	return fmt.Sprintf("parser: invalid rune at position %v", e.Position)
}

type State struct {
	data  *data
	datap int

	offset int64
	line   int
	column int
}

func NewStateString(s string) State {
	return State{
		data: newDataString(s),
	}
}

func NewStateBytes(b []byte) State {
	return State{
		data: newDataBytes(b),
	}
}

func NewStateReader(r io.Reader) State {
	return NewStateReaderSize(r, 1024)
}

func NewStateReaderSize(r io.Reader, size int) State {
	size = max(size, minDataSize)
	return State{
		data: newDataReaderSize(r, size),
	}
}

func (s State) Position() Position {
	return Position{
		Offset: s.offset,
		Line:   s.Line(),
		Column: s.Column(),
	}
}

func (s State) Line() int {
	return s.line + 1
}

func (s State) Column() int {
	return s.column + 1
}

func (s State) Rune() (rune, State, error) {
	if s.datap < len(s.data.buf) {
		r, rs := utf8.DecodeRune(s.data.buf[s.datap:])
		// TODO what did it think the error was without the rs == 1 check.
		if r == utf8.RuneError {
			if s.data.r == nil {
				return 0, s, RuneError{s.Position()}
			}
			// There is more data to read.
			if s.data.next == nil {
				// There is more data to read and next to initialize.
				s.data.next = newDataReaderSize(s.data.r, cap(s.data.buf))
			}
			// Now have more data in next to use for decode rune.
			// minDataSize means run has to fit in s.data and s.data.next.
			nextDataPMax := min(len(s.data.next.buf), 6)
			runeBytes := make([]byte, 0, len(s.data.buf)-s.datap+nextDataPMax)
			runeBytes = append(runeBytes, s.data.buf[s.datap:]...)
			runeBytes = append(runeBytes, s.data.next.buf[:nextDataPMax]...)
			r, rs = utf8.DecodeRune(runeBytes)
			// TODO what did it think the error was without the rs == 1 check.
			if r == utf8.RuneError {
				return 0, s, RuneError{s.Position()}
			}
			nextState := s.nextDataState()
			nextState.datap = rs - (len(s.data.buf) - s.datap)
			return r, nextState, nil
		}
		next, err := s.consume(rs, r)
		return r, next, err
	}

	// following should be the same as the Byte method.

	// assuming we are at a clean rune boundary.
	// s is at the end of data's buffer.

	if s.data.r != nil && s.data.next == nil {
		// There is more data to read and next to initialize.
		s.data.next = newDataReaderSize(s.data.r, cap(s.data.buf))
		return s.nextDataState().Rune()
	}

	// r is nil OR next is not nil
	if s.data.next != nil {
		log.Println("Rune() ALREADY AT A NEXT")
		return s.nextDataState().Rune()
	}

	return 0, s, io.EOF
}

func (s State) Byte() (byte, State, error) {
	if s.datap < len(s.data.buf) {
		// Have a byte available in data's buffer.
		b := s.data.buf[s.datap]
		next, err := s.consume(1, rune(b))
		return b, next, err
	}

	// s is at the end of data's buffer.

	if s.data.r != nil && s.data.next == nil {
		// There is more data to read and next to initialize.
		s.data.next = newDataReaderSize(s.data.r, cap(s.data.buf))
		return s.nextDataState().Byte()
	}

	// r is nil OR next is not nil
	if s.data.next != nil {
		return s.nextDataState().Byte()
	}

	return 0, s, io.EOF
}

func (s State) nextDataState() State {
	if s.data.next == nil {
		panic("s.data.next is nil")
	}
	return State{
		data:   s.data.next,
		datap:  0,
		offset: s.offset,
		line:   s.line,
		column: s.column,
	}
}

func (s State) consume(count int, v rune) (State, error) {
	s.datap += count
	s.offset += int64(count)
	s.column++
	if v == '\n' {
		s.line++
		s.column = 0
	}
	return s, nil
}

func keepBytes(start, end State) []byte {
	if start.data == end.data {
		result := start.data.buf[start.datap:end.datap]
		clone := make([]byte, len(result))
		copy(clone, result)
		return clone
	}

	result := make([]byte, end.offset-start.offset)
	resultp := copy(result, start.data.buf[start.datap:])
	current := start.nextDataState()
	for current.data != end.data {
		n := copy(result[resultp:], current.data.buf)
		resultp += n
		current = current.nextDataState()
	}
	copy(result[resultp:], end.data.buf[:end.datap])
	return result
}

// data is a node in a linked list of []byte.
// INVARIANTS: TODO
type data struct {
	r    io.Reader
	err  error
	buf  []byte
	next *data
}

func newDataString(s string) *data {
	return &data{
		buf: []byte(s),
	}
}

func newDataBytes(b []byte) *data {
	d := &data{
		buf: make([]byte, len(b)),
	}
	copy(d.buf, b)
	return d
}

const minDataSize = 8

func newDataReaderSize(r io.Reader, size int) *data {
	// TODO both: only optimize when the size isn't specified.
	if _, ok := r.(*strings.Reader); ok {
	}
	if _, ok := r.(*bytes.Reader); ok {
	}

	d := &data{
		r:   r,
		buf: make([]byte, size),
	}
	// TODO make more robust, check for empty reads in loop.
	n, err := r.Read(d.buf)
	d.buf = d.buf[:n]
	if err == io.EOF {
		err = nil
		d.r = nil
	}
	d.err = err
	return d
}

// ---------------------------------------------

type Position struct {
	Offset int64
	Line   int
	Column int
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
