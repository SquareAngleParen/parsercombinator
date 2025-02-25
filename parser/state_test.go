package parser

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"log"
	mathrand "math/rand/v2"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestState_Rune_PanicsOnZeroState(t *testing.T) {
	s := State{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("recover() = %v WANT %v", r, "non-nil")
		}
	}()

	s.Rune()
}

func TestState_Rune_ErrorsEOFWithEmptyString(t *testing.T) {
	s := NewStateString("")

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatal()
	}
}

func TestState_Rune_ConsumesAllFromStringData(t *testing.T) {
	s := NewStateString("∞")

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != "∞" {
		t.Fatal(result)
	}
}

func TestState_Rune_ErrorsEOFWithEmptyBytes(t *testing.T) {
	s := NewStateBytes([]byte{})

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatal()
	}
}

func TestState_Rune_ConsumesAllFromByteData(t *testing.T) {
	b := make([]byte, 3) // Infinity is three bytes.
	utf8.EncodeRune(b, '∞')
	s := NewStateBytes(b)

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != "∞" {
		t.Fatal(result)
	}
}

func TestState_Rune_ConsumesStateEndingAtDataNode8(t *testing.T) {
	const data = "aaaaBBBB"
	s := NewStateReaderSize(strings.NewReader(data), 8)

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != data {
		t.Fatal(result)
	}
}

func TestState_Rune_ConsumesStateEndingAtDataNode32(t *testing.T) {
	const data = "aaaaBBBBccccDDDDeeeeFFFFggggHHHH"
	s := NewStateReaderSize(strings.NewReader(data), 8)

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != data {
		t.Fatal(result)
	}
}

func TestState_Rune_ConsumesStateAcrossDataNodes(t *testing.T) {
	const data = "aaaaBBBBccccDDDDee"
	s := NewStateReaderSize(strings.NewReader(data), 8)

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != data {
		t.Fatal(result)
	}
}

func TestState_Rune_ConsumesStateAcrossDataNodesRunesMatchingDataBoundary(t *testing.T) {
	const data = "\u0081\u0082\u0083\u0084\u0085\u0086\u0087\u0088\u0089\u008a"
	s := NewStateReaderSize(strings.NewReader(data), 8)

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != data {
		t.Fatal(result)
	}
}

func TestState_Rune_ConsumesRuneSplitAcrossDataBoundary(t *testing.T) {
	const data = "1234567∞"
	s := NewStateReaderSize(strings.NewReader(data), 8)

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != data {
		t.Fatal(result)
	}
}

func TestState_Rune_ErrorsRuneErrorWithInvalidRuneEncodingInDataBoundary(t *testing.T) {
	data := "1∞"
	data = data[:3]

	s := NewStateString(data)
	_, s, _ = s.Rune()
	_, s, err := s.Rune()

	var want RuneError
	if !errors.As(err, &want) {
		t.Fatal()
	}
	if want.Position.Offset != 1 {
		t.Fatal()
	}
}

func TestState_Rune_ErrorsRuneErrorWithInvalidRuneEncodingAcrossDataBoundary(t *testing.T) {
	data := "1234567∞"
	data = data[:9] + "acbd"

	s := NewStateReaderSize(strings.NewReader(data), 8)
	_, err := allStateRunes(s)

	var want RuneError
	if !errors.As(err, &want) {
		t.Fatal()
	}
	if want.Position.Offset != 7 {
		t.Fatal()
	}
}

func TestState_Rune_EventuallyConsumesEOFWithEmptyLastDataNode(t *testing.T) {
	s := NewStateReader(&fullThenZeroReader{5})
	_, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
}

func TestState_Rune_DuplicateFullReadsReturnTheSameResult(t *testing.T) {
	const count = 250
	data := make([]byte, count*utf8.UTFMax)
	p := 0
	for i := 0; i < count; i++ {
		// var r rune
		// for {
		// 	r = rune(mathrand.IntN(utf8.MaxRune))
		// 	if utf8.ValidRune(r) {
		// 		break
		// 	}
		// }
		r := rune(mathrand.IntN(utf8.MaxRune))
		ip := utf8.EncodeRune(data[p:], r)
		p += ip
	}
	data = data[:p] // TODO does removing this do anything?
	log.Println("len(data)", len(data))

	s := NewStateReaderSize(bytes.NewReader(data), 13)

	result1, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}

	result2, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}

	if !slices.Equal(result1, result2) {
		t.Fatal()
	}
}

func TestState_Rune_CanProperlyDecodeAValidRuneErrorValue(t *testing.T) {
	data := make([]byte, utf8.RuneLen(utf8.RuneError))
	utf8.EncodeRune(data, utf8.RuneError)

	s := NewStateBytes(data)

	result, err := allStateRunes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if len(result) != 1 && result[0] != utf8.RuneError {
		t.Fatal()
	}
}

// --

func TestState_Byte_PanicsOnZeroState(t *testing.T) {
	s := State{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("recover() = %v WANT %v", r, "non-nil")
		}
	}()

	s.Byte()
}

func TestState_Byte_ErrorsEOFWithEmptyString(t *testing.T) {
	s := NewStateString("")

	result, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatal()
	}
}

func TestState_Byte_ConsumesAllFromStringData(t *testing.T) {
	s := NewStateString("foobar")

	result, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != "foobar" {
		t.Fatal(result)
	}
}

func TestState_Byte_ErrorsEOFWithEmptyBytes(t *testing.T) {
	s := NewStateBytes([]byte{})

	result, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatal()
	}
}

func TestState_Byte_ConsumesAllFromByteData(t *testing.T) {
	s := NewStateBytes([]byte{1, 2, 3, 4})

	result, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if !bytes.Equal(result, []byte{1, 2, 3, 4}) {
		t.Fatal(result)
	}
}

func TestState_Byte_ConsumesStateEndingAtDataNode8(t *testing.T) {
	const data = "aaaaBBBB"
	s := NewStateReaderSize(strings.NewReader(data), 8)

	result, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != data {
		t.Fatal(result)
	}
}

func TestState_Byte_ConsumesStateEndingAtDataNode32(t *testing.T) {
	const data = "aaaaBBBBccccDDDDeeeeFFFFggggHHHH"
	s := NewStateReaderSize(strings.NewReader(data), 8)

	result, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != data {
		t.Fatal(result)
	}
}

func TestState_Byte_ConsumesStateAcrossDataNodes(t *testing.T) {
	const data = "aaaaBBBBccccDDDDee"
	s := NewStateReaderSize(strings.NewReader(data), 8)

	result, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	if string(result) != data {
		t.Fatal(result)
	}
}

func TestState_Byte_EventuallyConsumesEOFWithEmptyLastDataNode(t *testing.T) {
	s := NewStateReader(&fullThenZeroReader{5})
	_, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}
}

func TestState_Byte_DuplicateFullReadsReturnTheSameResult(t *testing.T) {
	r := io.LimitReader(rand.Reader, 250)
	s := NewStateReaderSize(r, 17)

	result1, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}

	result2, err := allStateBytes(s)
	if err != io.EOF {
		t.Fatal(err)
	}

	if !bytes.Equal(result1, result2) {
		t.Fatal()
	}
}

type fullThenZeroReader struct {
	count int
}

func (r *fullThenZeroReader) Read(p []byte) (n int, err error) {
	if r.count > 0 {
		n = len(p)
	} else {
		err = io.EOF
	}
	r.count--
	return
}

// TODO re-reading state gets us to already next and is the same as original read.
// TODO additionally, we should get the same errors with rune reading after multiple reads.

func allStateBytes(s State) ([]byte, error) {
	result := []byte{}
	var b byte
	var err error
	b, s, err = s.Byte()
	for err == nil {
		result = append(result, b)
		b, s, err = s.Byte()
	}
	return result, err
}

func allStateRunes(s State) ([]rune, error) {
	result := []rune{}
	var r rune
	var err error
	r, s, err = s.Rune()
	for err == nil {
		result = append(result, r)
		r, s, err = s.Rune()
	}
	return result, err
}
