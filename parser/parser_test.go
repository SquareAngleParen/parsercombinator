package parser

import (
	"strings"
	"testing"
)

func TestExactString(t *testing.T) {
	p := ExactString("foo")

	_, err := DoParse(p, "foo")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAndThen_OK(t *testing.T) {
	p := AndThen(
		ExactString("foo"),
		func(_ Empty) Parser[Empty] {
			return ExactBytes([]byte("bar"))
		},
	)

	_, err := DoParse(p, "foobar")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCollectBytes(t *testing.T) {
	const data = "1234abcd5678efgh90"
	p := CollectBytes(ExactString(data))
	state := NewStateReaderSize(strings.NewReader(data), 8)

	result, err := DoParseState(p, state)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != data {
		t.Fatal()
	}
}

// TODO more collect tests
