// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocov

// Types and constants related to the output files files written
// by code coverage tooling. When a coverage-instrumented binary
// is run, it emits two output files: a meta-data output file, and
// a counter data output file.

//.....................................................................
//
// Meta-data definitions:
//
// The meta-data file is composed of a file header, a series of
// meta-data blobs/sections (one per instrumented package), and an offsets
// area storing the offsets of each section. Format of the meta-data
// file looks like:
//
// --header----------
//  | magic: [4]byte magic string
//  | version
//  | total length of meta-data file in bytes
//  | numPkgs: number of package entries in file
//  | hash: [16]byte hash of entire meta-data payload
//  | offset to string table section
//  | length of string table
//  | number of entries in string table
//  | counter mode
//  | counter granularity
//  --package offsets table------
//  <offset to pkg 0>
//  <offset to pkg 1>
//  ...
//  --package lengths table------
//  <length of pkg 0>
//  <length of pkg 1>
//  ...
//  --string table------
//  <uleb128 len> 8
//  <data> "somestring"
//  ...
//  --package payloads------
//  <meta-symbol for pkg 0>
//  <meta-symbol for pkg 1>
//  ...
//
// Each package payload is a stand-alone blob emitted by the compiler,
// and does not depend on anything else in the meta-data file. In
// particular, each blob has it's own string table. Note that the
// file-level string table is expected to be very short (most strings
// will be in the meta-data blobs themselves).

// covMetaMagic holds the magic string for a meta-data file.
var covMetaMagic = [4]byte{'\x00', '\x63', '\x76', '\x6d'}

// metaFilePref is a prefix used when emitting meta-data files; these
// files are of the form "covmeta.<hash>", where hash is a hash
// computed from the hashes of all the package meta-data symbols in
// the program.
const metaFilePref = "covmeta"

// metaFileVersion contains the current (most recent) meta-data file version.
const metaFileVersion = 1

// metaFileHeader stores file header information for a meta-data file.
type metaFileHeader struct {
	Magic        [4]byte
	Version      uint32
	TotalLength  uint64
	Entries      uint64
	MetaFileHash [16]byte
	StrTabOffset uint32
	StrTabLength uint32
	CMode        counterMode
	CGranularity CounterGranularity
	_            [6]byte // padding
}

// metaSymbolHeader stores header information for a single
// meta-data blob, e.g. the coverage meta-data payload
// computed for a given Go package.
type metaSymbolHeader struct {
	Length     uint32 // size of meta-symbol payload in bytes
	PkgName    uint32 // string table index
	PkgPath    uint32 // string table index
	ModulePath uint32 // string table index
	MetaHash   [16]byte
	_          byte    // currently unused
	_          [3]byte // padding
	NumFiles   uint32
	NumFuncs   uint32
}

const covMetaHeaderSize = 16 + 4 + 4 + 4 + 4 + 4 + 4 + 4 // keep in sync with above

// As an example, consider the following Go package:
//
// 01: package p
// 02:
// 03: var v, w, z int
// 04:
// 05: func small(x, y int) int {
// 06:   v++
// 07:   // comment
// 08:   if y == 0 {
// 09:     return x
// 10:   }
// 11:   return (x << 1) ^ (9 / y)
// 12: }
// 13:
// 14: func Medium(q, r int) int {
// 15:   s1 := small(q, r)
// 16:   z += s1
// 17:   s2 := small(r, q)
// 18:   w -= s2
// 19:   return w + z
// 20: }
//
// The meta-data blob for the single package above might look like the
// following:
//
// -- MetaSymbolHeader header----------
//  | size: size of this blob in bytes
//  | packagepath: <path to p>
//  | modulepath: <modpath for p>
//  | nfiles: 1
//  | nfunctions: 2
//  --func offsets table------
//  <offset to func 0>
//  <offset to func 1>
//  --string table (contains all files and functions)------
//  | <uleb128 len> 4
//  | <data> "p.go"
//  | <uleb128 len> 5
//  | <data> "small"
//  | <uleb128 len> 6
//  | <data> "Medium"
//  --func 0------
//  | <uleb128> num units: 3
//  | <uleb128> func name: S1 (index into string table)
//  | <uleb128> file: S0 (index into string table)
//  | <unit 0>:  S0   L6     L8    2
//  | <unit 1>:  S0   L9     L9    1
//  | <unit 2>:  S0   L11    L11   1
//  --func 1------
//  | <uleb128> num units: 1
//  | <uleb128> func name: S2 (index into string table)
//  | <uleb128> file: S0 (index into string table)
//  | <unit 0>:  S0   L15    L19   5
//  ---end-----------

// The following types and constants used by the meta-data encoder/decoder.

// funcDesc encapsulates the meta-data definitions for a single Go function.
// This version assumes that we're looking at a function before inlining;
// if we want to capture a post-inlining view of the world, the
// representations of source positions would need to be a good deal more
// complicated.
type funcDesc struct {
	Funcname string
	Srcfile  string
	Units    []coverableUnit
	Lit      bool // true if this is a function literal
}

// coverableUnit describes the source characteristics of a single
// program unit for which we want to gather coverage info. Coverable
// units are either "simple" or "intraline"; a "simple" coverable unit
// corresponds to a basic block (region of straight-line code with no
// jumps or control transfers). An "intraline" unit corresponds to a
// logical clause nested within some other simple unit. A simple unit
// will have a zero Parent value; for an intraline unit NxStmts will
// be zero and and Parent will be set to 1 plus the index of the
// containing simple statement. Example:
//
//	L7:   q := 1
//	L8:   x := (y == 101 || launch() == false)
//	L9:   r := x * 2
//
// For the code above we would have three simple units (one for each
// line), then an intraline unit describing the "launch() == false"
// clause in line 8, with Parent pointing to the index of the line 8
// unit in the units array.
//
// Note: in the initial version of the coverage revamp, only simple
// units will be in use.
type coverableUnit struct {
	StLine, StCol uint32
	EnLine, EnCol uint32
	NxStmts       uint32
	Parent        uint32
}

// counterMode tracks the "flavor" of the coverage counters being
// used in a given coverage-instrumented program.
type counterMode uint8

const (
	CtrModeInvalid  counterMode = iota
	CtrModeSet                  // "set" mode
	CtrModeCount                // "count" mode
	CtrModeAtomic               // "atomic" mode
	CtrModeRegOnly              // registration-only pseudo-mode
	CtrModeTestMain             // testmain pseudo-mode
)

func (cm counterMode) String() string {
	switch cm {
	case CtrModeSet:
		return "set"
	case CtrModeCount:
		return "count"
	case CtrModeAtomic:
		return "atomic"
	case CtrModeRegOnly:
		return "regonly"
	case CtrModeTestMain:
		return "testmain"
	}
	return "<invalid>"
}

func ParseCounterMode(mode string) counterMode {
	var cm counterMode
	switch mode {
	case "set":
		cm = CtrModeSet
	case "count":
		cm = CtrModeCount
	case "atomic":
		cm = CtrModeAtomic
	case "regonly":
		cm = CtrModeRegOnly
	case "testmain":
		cm = CtrModeTestMain
	default:
		cm = CtrModeInvalid
	}
	return cm
}

// CounterGranularity tracks the granularity of the coverage counters being
// used in a given coverage-instrumented program.
type CounterGranularity uint8

const (
	CtrGranularityInvalid CounterGranularity = iota
	CtrGranularityPerBlock
	CtrGranularityPerFunc
)

func (cm CounterGranularity) String() string {
	switch cm {
	case CtrGranularityPerBlock:
		return "perblock"
	case CtrGranularityPerFunc:
		return "perfunc"
	}
	return "<invalid>"
}

//.....................................................................
//
// Counter data definitions:
//

// A counter data file is composed of a file header followed by one or
// more "segments" (each segment representing a given run or partial
// run of a give binary) followed by a footer.

// covCounterMagic holds the magic string for a coverage counter-data file.
var covCounterMagic = [4]byte{'\x00', '\x63', '\x77', '\x6d'}

// counterFileVersion stores the most recent counter data file version.
const counterFileVersion = 1

// counterFileHeader stores files header information for a counter-data file.
type counterFileHeader struct {
	Magic     [4]byte
	Version   uint32
	MetaHash  [16]byte
	CFlavor   counterFlavor
	BigEndian bool
	_         [6]byte // padding
}

// counterSegmentHeader encapsulates information about a specific
// segment in a counter data file, which at the moment contains
// counters data from a single execution of a coverage-instrumented
// program. Following the segment header will be the string table and
// args table, and then (possibly) padding bytes to bring the byte
// size of the preamble up to a multiple of 4. Immediately following
// that will be the counter payloads.
//
// The "args" section of a segment is used to store annotations
// describing where the counter data came from; this section is
// basically a series of key-value pairs (can be thought of as an
// encoded 'map[string]string'). At the moment we only write os.Args()
// data to this section, using pairs of the form "argc=<integer>",
// "argv0=<os.Args[0]>", "argv1=<os.Args[1]>", and so on. In the
// future the args table may also include things like GOOS/GOARCH
// values, and/or tags indicating which tests were run to generate the
// counter data.
type counterSegmentHeader struct {
	FcnEntries uint64
	StrTabLen  uint32
	ArgsLen    uint32
}

// counterFileFooter appears at the tail end of a counter data file,
// and stores the number of segments it contains.
type counterFileFooter struct {
	Magic       [4]byte
	_           [4]byte // padding
	NumSegments uint32
	_           [4]byte // padding
}

// counterFilePref is the file prefix used when emitting coverage data
// output files. CounterFileTemplate describes the format of the file
// name: prefix followed by meta-file hash followed by process ID
// followed by emit UnixNanoTime.
const counterFilePref = "covcounters"
const counterFileRegexp = `^%s\.(\S+)\.(\d+)\.(\d+)+$`

// counterFlavor describes how function and counters are
// stored/represented in the counter section of the file.
type counterFlavor uint8

const (
	// "Raw" representation: all values (pkg ID, func ID, num counters,
	// and counters themselves) are stored as uint32's.
	ctrRaw counterFlavor = iota + 1

	// "ULeb" representation: all values (pkg ID, func ID, num counters,
	// and counters themselves) are stored with ULEB128 encoding.
	ctrULeb128
)
