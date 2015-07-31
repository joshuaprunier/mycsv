// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"database/sql"
	"errors"
	"testing"
)

// The output must have start and end quotes added as the tests use a default writer
var writeTests = []struct {
	Input  [][]sql.RawBytes
	Output string
}{
	{Input: [][]sql.RawBytes{{[]byte("abc")}}, Output: "\"abc\"\n"},
	{Input: [][]sql.RawBytes{{[]byte(`"abc"`)}}, Output: `"\"abc\""` + "\n"},
	{Input: [][]sql.RawBytes{{[]byte(`a"b`)}}, Output: `"a\"b"` + "\n"},
	{Input: [][]sql.RawBytes{{[]byte(`"a"b"`)}}, Output: `"\"a\"b\""` + "\n"},
	{Input: [][]sql.RawBytes{{[]byte(" abc")}}, Output: `" abc"` + "\n"},
	{Input: [][]sql.RawBytes{{[]byte("abc,def")}}, Output: `"abc,def"` + "\n"},
	{Input: [][]sql.RawBytes{{[]byte("abc"), []byte("def")}}, Output: `"abc","def"` + "\n"},
	{Input: [][]sql.RawBytes{{[]byte("abc")}, {[]byte("def")}}, Output: `"abc"` + "\n" + `"def"` + "\n"},
	{Input: [][]sql.RawBytes{{[]byte("abc\ndef")}}, Output: "\"abc\\\ndef\"\n"},
	{Input: [][]sql.RawBytes{{[]byte("abc\rdef")}}, Output: "\"abc\rdef\"\n"},
	{Input: [][]sql.RawBytes{{[]byte("")}}, Output: `""` + "\n"},
	{Input: [][]sql.RawBytes{{[]byte(""), []byte("")}}, Output: "\"\",\"\"\n"},
	{Input: [][]sql.RawBytes{{[]byte(""), []byte(""), []byte("")}}, Output: "\"\",\"\",\"\"\n"},
	{Input: [][]sql.RawBytes{{[]byte(""), []byte(""), []byte("a")}}, Output: "\"\",\"\",\"a\"\n"},
	{Input: [][]sql.RawBytes{{[]byte(""), []byte("a"), []byte("")}}, Output: "\"\",\"a\",\"\"\n"},
	{Input: [][]sql.RawBytes{{[]byte(""), []byte("a"), []byte("a")}}, Output: "\"\",\"a\",\"a\"\n"},
	{Input: [][]sql.RawBytes{{[]byte("a"), []byte(""), []byte("")}}, Output: "\"a\",\"\",\"\"\n"},
	{Input: [][]sql.RawBytes{{[]byte("a"), []byte(""), []byte("a")}}, Output: "\"a\",\"\",\"a\"\n"},
	{Input: [][]sql.RawBytes{{[]byte("a"), []byte("a"), []byte("")}}, Output: "\"a\",\"a\",\"\"\n"},
	{Input: [][]sql.RawBytes{{[]byte("a"), []byte("a"), []byte("a")}}, Output: "\"a\",\"a\",\"a\"\n"},
	{Input: [][]sql.RawBytes{{[]byte(`\.`)}}, Output: `"\\."` + "\n"},
}

var empty string

func TestWrite(t *testing.T) {
	for n, tt := range writeTests {
		b := &bytes.Buffer{}
		f := NewWriter(b)
		err := f.WriteAll(tt.Input)
		if err != nil {
			t.Errorf("Unexpected error: %s\n", err)
		}
		got := b.String()
		if got != tt.Output {
			t.Errorf("#%d: got=%q want=%q", n, got, tt.Output)
		}
	}
}

type errorWriter struct{}

func (e errorWriter) Write(b []byte) (int, error) {
	return 0, errors.New("Test")
}

func TestError(t *testing.T) {
	b := &bytes.Buffer{}
	f := NewWriter(b)
	f.Write([]sql.RawBytes{[]byte("abc")})
	f.Flush()
	err := f.Error()

	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	f = NewWriter(errorWriter{})
	f.Write([]sql.RawBytes{[]byte("abc")})
	f.Flush()
	err = f.Error()

	if err == nil {
		t.Error("Error should not be nil")
	}
}

func BenchmarkWritePlainCharacters(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		f := NewWriter(buf)
		f.Write([]sql.RawBytes{[]byte(`abcdef`)})
		f.Flush()
		err := f.Error()

		if err != nil {
			b.Errorf("Unexpected error: %s\n", err)
		}
	}
}

func BenchmarkWriteEscapeCharacters(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		f := NewWriter(buf)
		f.Write([]sql.RawBytes{[]byte(`a,b,,c\n"def"`)})
		f.Flush()
		err := f.Error()

		if err != nil {
			b.Errorf("Unexpected error: %s\n", err)
		}
	}
}

func BenchmarkWriteBaconIpsum(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		f := NewWriter(buf)
		f.Write([]sql.RawBytes{[]byte(`
		Bacon ipsum dolor amet beef ribs fatback cupim, pig pancetta pork loin ribeye shankle t-bone beef strip steak capicola. Jerky alcatra rump, andouille doner turducken jowl. Turducken landjaeger beef, rump drumstick ham shoulder pork belly biltong boudin meatball jowl doner fatback. Flank ball tip pork belly brisket. Flank spare ribs tail alcatra, doner turducken sausage. Beef ribs drumstick spare ribs biltong ham hock rump jowl, ham brisket kevin prosciutto.

		Beef andouille spare ribs, jowl alcatra doner bresaola chuck landjaeger pork ball tip. Sausage sirloin ham chicken bacon. Rump pastrami tenderloin pancetta brisket andouille kielbasa fatback cow ribeye. Shankle chicken leberkas, pancetta shank drumstick doner filet mignon pastrami cupim. Drumstick filet mignon tail doner, tenderloin flank shank ground round pork loin landjaeger. Bresaola fatback filet mignon flank kielbasa shoulder. Chuck cupim bacon leberkas.

		Biltong brisket tail, swine chuck kevin picanha cow rump corned beef landjaeger cupim meatloaf porchetta ball tip. Kielbasa ham doner beef ribs t-bone tongue cow drumstick flank filet mignon fatback. Boudin salami ham hock, tail sausage spare ribs pancetta meatloaf flank filet mignon jowl meatball doner. Chicken salami shank, jerky meatloaf short ribs bacon cow.

		Chicken ham leberkas, short loin tri-tip capicola fatback tenderloin pig sausage meatloaf tongue beef sirloin shoulder. Short loin chuck beef jowl drumstick fatback pork loin ribeye tri-tip turkey picanha kevin short ribs rump. Meatloaf turkey frankfurter flank. Salami drumstick rump, tail bacon kevin meatball jowl ribeye swine ball tip bresaola. Doner corned beef sausage flank cupim fatback. Spare ribs pork loin meatloaf picanha turducken landjaeger pastrami salami. Fatback turkey drumstick ham landjaeger bresaola tri-tip short loin.
		`)})
		f.Flush()
		err := f.Error()

		if err != nil {
			b.Errorf("Unexpected error: %s\n", err)
		}
	}
}
