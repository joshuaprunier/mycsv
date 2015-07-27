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
