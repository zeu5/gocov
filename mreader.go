// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocov

import (
	"io"
	"os"

	"github.com/zeu5/gocov/bio"
)

// This file contains the helper "MReader", a wrapper around bio plus
// an "mmap'd read-only" view of the file obtained from bio.SliceRO().
// MReader is designed to implement the io.ReaderSeeker interface.
// Since bio.SliceOS() is not guaranteed to succeed, MReader falls back
// on explicit reads + seeks provided by bio.Reader if needed.

type mReader struct {
	rdr      *bio.Reader
	fileView []byte
	off      int64
}

func newMreader(f *os.File) (*mReader, error) {
	rdr := bio.NewReader(f)
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	r := mReader{
		rdr:      rdr,
		fileView: rdr.SliceRO(uint64(fi.Size())),
	}
	return &r, nil
}

func (r *mReader) Read(p []byte) (int, error) {
	if r.fileView != nil {
		amt := len(p)
		toread := r.fileView[r.off:]
		if len(toread) < 1 {
			return 0, io.EOF
		}
		if len(toread) < amt {
			amt = len(toread)
		}
		copy(p, toread)
		r.off += int64(amt)
		return amt, nil
	}
	return io.ReadFull(r.rdr, p)
}

func (r *mReader) ReadByte() (byte, error) {
	if r.fileView != nil {
		toread := r.fileView[r.off:]
		if len(toread) < 1 {
			return 0, io.EOF
		}
		rv := toread[0]
		r.off++
		return rv, nil
	}
	return r.rdr.ReadByte()
}

func (r *mReader) Seek(offset int64, whence int) (int64, error) {
	if r.fileView == nil {
		return r.rdr.MustSeek(offset, whence), nil
	}
	switch whence {
	case io.SeekStart:
		r.off = offset
		return offset, nil
	case io.SeekCurrent:
		return r.off, nil
	case io.SeekEnd:
		r.off = int64(len(r.fileView)) + offset
		return r.off, nil
	}
	panic("other modes not implemented")
}
