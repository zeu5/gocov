// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocov

import "bytes"

type PodData struct {
	CounterGranularity CounterGranularity
	CounterMode        CounterMode
	// Number of functions in each package
	Packages map[uint32]*Package
}

type Package struct {
	ID         uint32
	Name       string
	ImportPath string
	ModulePath string
	NumFuncs   uint32
	Funcs      map[uint32]*Func
}

type Func struct {
	Name    string
	SrcFile string
	Units   []*FuncUnit
}

type FuncUnit struct {
	StLine, StCol uint32
	EnLine, EnCol uint32
	NxStmts       uint32
	Parent        uint32
	Count         uint32
}

type CoverageData struct {
	PodData map[string]*PodData
}

func ReadDir(dir string, matchPkgs []string) (*CoverageData, error) {
	data := &CoverageData{
		PodData: make(map[string]*PodData),
	}

	vis := &covDataVisitor{
		cm:   &Merger{},
		data: data,
	}
	reader := MakeCovDataDirReader(vis, dir, matchPkgs...)
	err := reader.Visit()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func ReadFromBuffer(meta, counters *bytes.Buffer, matchPkgs []string) (*CoverageData, error) {
	data := &CoverageData{
		PodData: make(map[string]*PodData),
	}

	vis := &covDataVisitor{
		cm:   &Merger{},
		data: data,
	}
	reader := MakeCovDataBufferReader(vis, counters, meta, matchPkgs...)
	err := reader.Visit()
	if err != nil {
		return nil, err
	}
	return data, nil
}
