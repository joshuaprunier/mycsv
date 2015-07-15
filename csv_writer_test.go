// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"testing"
)

// The output must have start and end quotes added as the tests use a default writer
var writeTests = []struct {
	Input  [][]NullRawBytes
	Output string
}{
	{Input: [][]NullRawBytes{{{[]byte("abc"), true}}}, Output: "\"abc\"\n"},
	{Input: [][]NullRawBytes{{{[]byte(`"abc"`), true}}}, Output: `"""abc"""` + "\n"},
	//{Input: [][]NullRawBytes{{`a"b`}}, Output: `"a""b"` + "\n"},
	//{Input: [][]NullRawBytes{{`"a"b"`}}, Output: `"""a""b"""` + "\n"},
	//{Input: [][]NullRawBytes{{" abc"}}, Output: `" abc"` + "\n"},
	//{Input: [][]NullRawBytes{{"abc,def"}}, Output: `"abc,def"` + "\n"},
	//{Input: [][]NullRawBytes{{"abc", "def"}}, Output: "abc,def\n"},
	//{Input: [][]NullRawBytes{{"abc"}, {"def"}}, Output: "abc\ndef\n"},
	//{Input: [][]NullRawBytes{{"abc\ndef"}}, Output: "\"abc\ndef\"\n"},
	//{Input: [][]NullRawBytes{{"abc\ndef"}}, Output: "\"abc\r\ndef\"\r\n"},
	//{Input: [][]NullRawBytes{{"abc\rdef"}}, Output: "\"abcdef\"\r\n"},
	//{Input: [][]NullRawBytes{{"abc\rdef"}}, Output: "\"abc\rdef\"\n"},
	//{Input: [][]NullRawBytes{{""}}, Output: "\n"},
	//{Input: [][]NullRawBytes{{"", ""}}, Output: ",\n"},
	//{Input: [][]NullRawBytes{{"", "", ""}}, Output: ",,\n"},
	//{Input: [][]NullRawBytes{{"", "", "a"}}, Output: ",,a\n"},
	//{Input: [][]NullRawBytes{{"", "a", ""}}, Output: ",a,\n"},
	//{Input: [][]NullRawBytes{{"", "a", "a"}}, Output: ",a,a\n"},
	//{Input: [][]NullRawBytes{{"a", "", ""}}, Output: "a,,\n"},
	//{Input: [][]NullRawBytes{{"a", "", "a"}}, Output: "a,,a\n"},
	//{Input: [][]NullRawBytes{{"a", "a", ""}}, Output: "a,a,\n"},
	//{Input: [][]NullRawBytes{{"a", "a", "a"}}, Output: "a,a,a\n"},
	//{Input: [][]NullRawBytes{{`\.`}}, Output: "\"\\.\"\n"},
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
		out := b.String()
		if out != tt.Output {
			t.Errorf("#%d: out=%q want %q", n, out, tt.Output)
		}
	}
}

type errorWriter struct{}

func (e errorWriter) Write(b []byte) (int, error) {
	return 0, errors.New("Test")
}

//func TestError(t *testing.T) {
//	b := &bytes.Buffer{}
//	f := NewWriter(b)
//	f.Write([]NullRawBytes{{[]byte("abc"), true}})
//	f.Flush()
//	err := f.Error()
//
//	if err != nil {
//		t.Errorf("Unexpected error: %s\n", err)
//	}
//
//	f = NewWriter(errorWriter{})
//	f.Write([]NullRawBytes{{[]byte("abc"), true}})
//	f.Flush()
//	err = f.Error()
//
//	if err == nil {
//		t.Error("Error should not be nil")
//	}
//}
