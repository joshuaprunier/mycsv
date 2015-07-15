// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"io"
)

// A Writer writes records to a MySQL compatible CSV encoded file.
// It is heavily influenced by the std lib encoding/CSV package.
//
// As returned by NewWriter, a Writer writes fields delimited by a comma, escapes special
// characters with a back slash and lines are terminated with a newline. The exported fields
// can be changed to customize the details before the first call to Write or WriteAll.
type Writer struct {
	Delimiter  rune   // Field delimiter (set to ',' by NewWriter)
	Quote      rune   // Quote character
	Escape     rune   // Escape character
	Terminator string // Character to end each line
	w          *bufio.Writer
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Delimiter:  ',',
		Quote:      '"',
		Escape:     '\\',
		Terminator: "\n",
		w:          bufio.NewWriter(w),
	}
}

// Writer writes a single CSV record to w along with any necessary quoting.
// A record is a slice of NullRawBytes so NULL's can be detected and escaped.
func (w *Writer) Write(record []NullRawBytes) (buf int, err error) {
	for n, field := range record {
		// Shortcut exit for empty strings
		if n > 0 {
			if _, err = w.w.WriteRune(w.Delimiter); err != nil {
				return
			}
		}

		// Check if and escape/translate if field is NULL
		if !field.Valid {
			_, err = w.w.WriteString("\\N")
			continue
		}

		// Write quote character if set
		if w.Quote > 0 {
			if _, err = w.w.WriteRune(w.Quote); err != nil {
				return
			}
		}

		// We need to examine each byte to determine if special characters need to be escaped
		bytes := field.RawBytes
		for _, byte := range bytes {
			switch rune(byte) {
			case w.Delimiter:
				if w.Quote < 0 {
					_, err = w.w.WriteRune(w.Escape)
					_, err = w.w.WriteRune(w.Delimiter)
				} else {
					_, err = w.w.WriteRune(w.Delimiter)
				}
			case w.Quote:
				_, err = w.w.WriteRune(w.Escape)
				_, err = w.w.WriteRune(w.Quote)
			case w.Escape:
				_, err = w.w.WriteRune(w.Escape)
				_, err = w.w.WriteRune(w.Escape)
			case 0x00:
				_, err = w.w.WriteRune(w.Escape)
				_, err = w.w.WriteRune('0')
			default:
				err = w.w.WriteByte(byte)

			}
			if err != nil {
				return
			}
		}

		// Write quote character if set
		if w.Quote > 0 {
			if _, err = w.w.WriteRune(w.Quote); err != nil {
				return
			}
		}
	}

	// Write line terminator
	_, err = w.w.WriteString(w.Terminator)

	// Return the number of bytes written to the current buffer
	buf = w.w.Buffered()

	return buf, err
}

// Flush writes any buffered data to the underlying io.Writer.
// To check if an error occurred during the Flush, call Error.
func (w *Writer) Flush() {
	w.w.Flush()
}

// Error reports any error that has occurred during a previous Write or Flush.
func (w *Writer) Error() error {
	_, err := w.w.Write(nil)
	return err
}

// WriteAll writes multiple CSV records to w using Write and then calls Flush.
func (w *Writer) WriteAll(records [][]NullRawBytes) (err error) {
	for _, record := range records {
		_, err = w.Write(record)
		if err != nil {
			return err
		}
	}
	return w.w.Flush()
}
