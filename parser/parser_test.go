package parser

import "testing"

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
