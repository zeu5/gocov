// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocov

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/zeu5/gocov/bio"
)

// covDataReader is a general-purpose helper/visitor object for
// reading coverage data files in a structured way. Clients create a
// covDataReader to process a given collection of coverage data file
// directories, then pass in a visitor object with methods that get
// invoked at various important points. covDataReader is intended
// to facilitate common coverage data file operations such as
// merging or intersecting data files, analyzing data files, or
// dumping data files.
type covDataReader struct {
	vis            *covDataVisitor
	dir            string
	counterBuffer  *bytes.Buffer
	metadataBuffer *bytes.Buffer
	pkgs           []string
}

// MakeCovDataReader creates a CovDataReader object to process the
// given set of input directories. Here 'vis' is a visitor object
// providing methods to be invoked as we walk through the data,
// 'indirs' is the set of coverage data directories to examine,
// 'verbosityLevel' controls the level of debugging trace messages
// (zero for off, higher for more output), 'flags' stores flags that
// indicate what to do if errors are detected, and 'matchpkg' is a
// caller-provided function that can be used to select specific
// packages by name (if nil, then all packages are included).
func makeCovDataDirReader(vis *covDataVisitor, dir string, pkgs ...string) *covDataReader {
	return &covDataReader{
		vis:  vis,
		dir:  dir,
		pkgs: pkgs,
	}
}

func makeCovDataBufferReader(vis *covDataVisitor, counter, metadata *bytes.Buffer, pkgs ...string) *covDataReader {
	return &covDataReader{
		vis:            vis,
		counterBuffer:  counter,
		metadataBuffer: metadata,
		pkgs:           pkgs,
	}
}

// CovDataVisitor defines hooks for clients of CovDataReader. When the
// coverage data reader makes its way through a coverage meta-data
// file and counter data files, it will invoke the methods below to
// hand off info to the client. The normal sequence of expected
// visitor method invocations is:
//
//	for each pod P {
//		BeginPod(p)
//		let MF be the meta-data file for P
//		VisitMetaDataFile(MF)
//		for each counter data file D in P {
//			BeginCounterDataFile(D)
//			for each live function F in D {
//				VisitFuncCounterData(F)
//			}
//			EndCounterDataFile(D)
//		}
//		EndCounters(MF)
//		for each package PK in MF {
//			BeginPackage(PK)
//			if <PK matched according to package pattern and/or modpath> {
//				for each function PF in PK {
//					VisitFunc(PF)
//				}
//			}
//			EndPackage(PK)
//		}
//		EndPod(p)
//	}
//	Finish()

func (r *covDataReader) Visit() error {
	if r.dir != "" {
		podlist, err := collectPods(r.dir)
		if err != nil {
			return fmt.Errorf("reading inputs: %v", err)
		}
		for _, p := range podlist {
			if err := r.visitPod(p); err != nil {
				return err
			}
		}
	} else {
		return r.visitSinglePod()
	}
	return nil
}

func (r *covDataReader) visitSinglePod() error {
	r.vis.BeginPod(pod{})

	f := bytes.NewReader(r.metadataBuffer.Bytes())
	fileView := r.metadataBuffer.Bytes()
	var mfr *coverageMetaFileReader
	mfr, err := newCoverageMetaFileReader(f, fileView)
	if err != nil {
		return fmt.Errorf("decoding meta-file: %s", err)
	}
	err = r.vis.VisitMetaDataFile(mfr)
	if err != nil {
		return err
	}

	mr := bytes.NewReader(r.counterBuffer.Bytes())
	var cdr *counterDataReader
	cdr, err = newCounterDataReader(mr)
	if err != nil {
		return fmt.Errorf("reading counter data file: %s", err)
	}
	var data funcPayload
	for {
		ok, err := cdr.NextFunc(&data)
		if err != nil {
			return fmt.Errorf("reading counter data file: %v", err)
		}
		if !ok {
			break
		}
		err = r.vis.VisitFuncCounterData(data)
		if err != nil {
			return err
		}
	}

	np := uint32(mfr.NumPackages())
	payload := []byte{}
	for pkIdx := uint32(0); pkIdx < np; pkIdx++ {
		var pd *coverageMetaDataDecoder
		pd, payload, err = mfr.GetPackageDecoder(pkIdx, payload)
		if err != nil {
			return fmt.Errorf("reading pkg %d from meta-file: %s", pkIdx, err)
		}
		r.processPackage(pd, pkIdx)
	}

	return nil
}

// visitPod examines a coverage data 'pod', that is, a meta-data file and
// zero or more counter data files that refer to that meta-data file.
func (r *covDataReader) visitPod(p pod) error {
	r.vis.BeginPod(p)

	// Open meta-file
	f, err := os.Open(p.MetaFile)
	if err != nil {
		return fmt.Errorf("unable to open meta-file %s", p.MetaFile)
	}
	defer f.Close()
	br := bio.NewReader(f)
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("unable to stat metafile %s: %v", p.MetaFile, err)
	}
	fileView := br.SliceRO(uint64(fi.Size()))
	br.MustSeek(0, io.SeekStart)

	var mfr *coverageMetaFileReader
	mfr, err = newCoverageMetaFileReader(f, fileView)
	if err != nil {
		return fmt.Errorf("decoding meta-file %s: %s", p.MetaFile, err)
	}
	err = r.vis.VisitMetaDataFile(mfr)
	if err != nil {
		return err
	}

	// Read counter data files.
	for _, cdf := range p.CounterDataFiles {
		cf, err := os.Open(cdf)
		if err != nil {
			return fmt.Errorf("opening counter data file %s: %s", cdf, err)
		}
		defer func(f *os.File) {
			f.Close()
		}(cf)
		var mr *mReader
		mr, err = newMreader(cf)
		if err != nil {
			return fmt.Errorf("creating reader for counter data file %s: %s", cdf, err)
		}
		var cdr *counterDataReader
		cdr, err = newCounterDataReader(mr)
		if err != nil {
			return fmt.Errorf("reading counter data file %s: %s", cdf, err)
		}
		var data funcPayload
		for {
			ok, err := cdr.NextFunc(&data)
			if err != nil {
				return fmt.Errorf("reading counter data file %s: %v", cdf, err)
			}
			if !ok {
				break
			}
			err = r.vis.VisitFuncCounterData(data)
			if err != nil {
				return err
			}

		}
	}

	// NB: packages in the meta-file will be in dependency order (basically
	// the order in which init files execute). Do we want an additional sort
	// pass here, say by packagepath?
	np := uint32(mfr.NumPackages())
	payload := []byte{}
	for pkIdx := uint32(0); pkIdx < np; pkIdx++ {
		var pd *coverageMetaDataDecoder
		pd, payload, err = mfr.GetPackageDecoder(pkIdx, payload)
		if err != nil {
			return fmt.Errorf("reading pkg %d from meta-file %s: %s", pkIdx, p.MetaFile, err)
		}
		r.processPackage(pd, pkIdx)
	}

	return nil
}

func (r *covDataReader) processPackage(pd *coverageMetaDataDecoder, pkgIdx uint32) error {
	if !r.matchpkg(pd.PackagePath()) {
		return nil
	}
	r.vis.BeginPackage(pd, pkgIdx)
	nf := pd.NumFuncs()
	var fd funcDesc
	for fidx := uint32(0); fidx < nf; fidx++ {
		if err := pd.ReadFunc(fidx, &fd); err != nil {
			return fmt.Errorf("reading meta-data file: %v", err)
		}
		r.vis.VisitFunc(pkgIdx, fidx, &fd)
	}
	return nil
}

func (r *covDataReader) matchpkg(path string) bool {
	if len(r.pkgs) == 0 {
		return true
	}
	for _, p := range r.pkgs {
		if matchSimplePattern(p, path) {
			return true
		}
	}
	return false
}
