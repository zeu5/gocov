// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bio implements common I/O abstractions used within the Go toolchain.
package bio

import (
	"bufio"
	"io"
	"log"
	"os"
)

// Reader implements a seekable buffered io.Reader.
type Reader struct {
	f *os.File
	*bufio.Reader
}

// Open returns a Reader for the file named name.
func Open(name string) (*Reader, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return NewReader(f), nil
}

// NewReader returns a Reader from an open file.
func NewReader(f *os.File) *Reader {
	return &Reader{f: f, Reader: bufio.NewReader(f)}
}

func (r *Reader) MustSeek(offset int64, whence int) int64 {
	if whence == 1 {
		offset -= int64(r.Buffered())
	}
	off, err := r.f.Seek(offset, whence)
	if err != nil {
		log.Fatalf("seeking in output: %v", err)
	}
	r.Reset(r.f)
	return off
}

func (r *Reader) Offset() int64 {
	off, err := r.f.Seek(0, 1)
	if err != nil {
		log.Fatalf("seeking in output [0, 1]: %v", err)
	}
	off -= int64(r.Buffered())
	return off
}

func (r *Reader) Close() error {
	return r.f.Close()
}

func (r *Reader) File() *os.File {
	return r.f
}

// Slice reads the next length bytes of r into a slice.
//
// This slice may be backed by mmap'ed memory. Currently, this memory
// will never be unmapped. The second result reports whether the
// backing memory is read-only.
func (r *Reader) Slice(length uint64) ([]byte, bool, error) {
	if length == 0 {
		return []byte{}, false, nil
	}

	data, ok := r.sliceOS(length)
	if ok {
		return data, true, nil
	}

	data = make([]byte, length)
	_, err := io.ReadFull(r, data)
	if err != nil {
		return nil, false, err
	}
	return data, false, nil
}

// SliceRO returns a slice containing the next length bytes of r
// backed by a read-only mmap'd data. If the mmap cannot be
// established (limit exceeded, region too small, etc) a nil slice
// will be returned. If mmap succeeds, it will never be unmapped.
func (r *Reader) SliceRO(length uint64) []byte {
	data, ok := r.sliceOS(length)
	if ok {
		return data
	}
	return nil
}
