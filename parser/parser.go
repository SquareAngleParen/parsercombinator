package parser

import (
	"fmt"
)

var ErrNoMatch = fmt.Errorf("parser: no match")

type Parser[T any] func(State) (T, State, error)

type Empty struct{}

func DoParse[T any](parser Parser[T], input string) (T, error) {
	state := State{
		buffer: []byte(input),
		offset: 0,
	}
	result, _, err := parser(state)
	return result, err
}

func AndThen[T any, U any](parser Parser[T], handler func(T) Parser[U]) Parser[U] {
	return func(s State) (U, State, error) {
		t, next, err := parser(s)
		if err != nil {
			var zero U
			return zero, s, err
		}
		return handler(t)(next)
	}
}

func ExactString(token string) Parser[Empty] {
	return func(s State) (Empty, State, error) {
		next := s
		for _, tokenRune := range token {
			var r rune
			var err error
			r, next, err = next.Rune()
			if err != nil {
				return Empty{}, s, err
			}
			if tokenRune != r {
				return Empty{}, s, ErrNoMatch
			}
		}
		return Empty{}, next, nil
	}
}

func ExactBytes(token []byte) Parser[Empty] {
	return func(s State) (Empty, State, error) {
		next := s
		for _, tokenByte := range token {
			var b byte
			var err error
			b, next, err = next.Byte()
			if err != nil {
				return Empty{}, s, err
			}
			if tokenByte != b {
				return Empty{}, s, ErrNoMatch
			}
		}
		return Empty{}, next, nil
	}
}

func Sequence2[T, U, R any](tParser Parser[T], uParser Parser[U], mapper func(T, U) (R, error)) Parser[R] {
	return func(s State) (R, State, error) {
		t, next, err := tParser(s)
		if err != nil {
			var zero R
			return zero, s, err
		}
		var u U
		u, next, err = uParser(s)
		if err != nil {
			var zero R
			return zero, s, err
		}
		r, err := mapper(t, u)
		if err != nil {
			var zero R
			return zero, s, err
		}
		return r, next, nil
	}
}

func ConditionRune(cond func(rune) bool) Parser[rune] {
	return func(s State) (rune, State, error) {
		r, next, err := s.Rune()
		if err != nil {
			return 0, s, err
		}
		if !cond(r) {
			return 0, s, ErrNoMatch
		}
		return r, next, nil
	}
}

func GetString[T any](parser Parser[T]) Parser[string] {
	return func(s State) (string, State, error) {
		start := s.offset
		_, next, err := parser(s)
		if err != nil {
			return "", s, err
		}
		end := s.offset
		return string(s.buffer[start:end]), next, nil
	}
}

func GetBytes[T any](parser Parser[T]) Parser[[]byte] {
	return func(s State) ([]byte, State, error) {
		start := s.offset
		_, next, err := parser(s)
		if err != nil {
			return nil, s, err
		}
		end := s.offset
		result := make([]byte, end-start)
		copy(result, s.buffer[start:end])
		return result, next, nil
	}
}

func GetPositions[T any](parser Parser[T]) Parser[[2]Position] {
	return func(s State) ([2]Position, State, error) {
		start := s.Position()
		_, next, err := parser(s)
		if err != nil {
			return [2]Position{}, s, err
		}
		end := s.Position()
		return [2]Position{start, end}, next, nil
	}
}
