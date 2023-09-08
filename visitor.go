package gocov

import (
	"encoding/hex"
	"fmt"
)

type pkfunc struct {
	pk, fcn uint32
}

// covDataVisitor encapsulates state and provides methods for implementing
// various dump operations. Specifically, covDataVisitor implements the
// CovDataVisitor interface, and is designed to be used in
// concert with the CovDataReader utility, which abstracts away most
// of the grubby details of reading coverage data files.
type covDataVisitor struct {
	// for batch allocation of counter arrays
	BatchCounterAlloc

	// counter merging state + methods
	cm *Merger

	// 'mm' stores values read from a counter data file; the pkfunc key
	// is a pkgid/funcid pair that uniquely identifies a function in
	// instrumented application.
	mm map[pkfunc]FuncPayload
	// pkm maps package ID to the number of functions in the package
	// with that ID. It is used to report inconsistencies in counter
	// data (for example, a counter data entry with pkgid=N funcid=10
	// where package N only has 3 functions).
	pkm map[uint32]uint32

	podHash string

	data *CoverageData
}

func (d *covDataVisitor) BeginPod(p Pod) {
	d.mm = make(map[pkfunc]FuncPayload)
}

func (d *covDataVisitor) VisitFuncCounterData(data FuncPayload) error {
	if nf, ok := d.pkm[data.PkgIdx]; !ok || data.FuncIdx > nf {
		return nil
	}
	key := pkfunc{pk: data.PkgIdx, fcn: data.FuncIdx}
	val, ok := d.mm[key]
	if !ok {
		val = FuncPayload{}
	}

	if len(val.Counters) < len(data.Counters) {
		t := val.Counters
		val.Counters = d.AllocateCounters(len(data.Counters))
		copy(val.Counters, t)
	}
	err, _ := d.cm.MergeCounters(val.Counters, data.Counters)
	if err != nil {
		return err
	}
	d.mm[key] = val
	return nil
}

func (d *covDataVisitor) VisitMetaDataFile(mfr *CoverageMetaFileReader) error {
	newgran := mfr.CounterGranularity()
	newmode := mfr.CounterMode()

	fileHash := mfr.FileHash()
	mHash := hex.EncodeToString(fileHash[:])
	podData := &PodData{
		CounterGranularity: newgran,
		CounterMode:        newmode,
		Packages:           make(map[uint32]*Package),
	}
	d.podHash = mHash
	d.data.PodData[mHash] = podData

	if err := d.cm.SetModeAndGranularity(newmode, newgran); err != nil {
		return err
	}
	// To provide an additional layer of checking when reading counter
	// data, walk the meta-data file to determine the set of legal
	// package/function combinations. This will help catch bugs in the
	// counter file reader.
	d.pkm = make(map[uint32]uint32)
	np := uint32(mfr.NumPackages())
	payload := []byte{}
	for pkIdx := uint32(0); pkIdx < np; pkIdx++ {
		var pd *CoverageMetaDataDecoder
		var err error
		pd, payload, err = mfr.GetPackageDecoder(pkIdx, payload)
		if err != nil {
			return fmt.Errorf("reading pkg %d from meta-file: %s", pkIdx, err)
		}
		d.pkm[pkIdx] = pd.NumFuncs()

		podData.Packages[pkIdx] = &Package{
			ID:       pkIdx,
			NumFuncs: pd.NumFuncs(),
			Funcs:    make(map[uint32]*Func),
		}
	}
	return nil
}

func (d *covDataVisitor) BeginPackage(pd *CoverageMetaDataDecoder, pkgIdx uint32) {
	podData := d.data.PodData[d.podHash]
	packageData, ok := podData.Packages[pkgIdx]
	if ok {
		packageData.Name = pd.PackageName()
		packageData.ImportPath = pd.PackagePath()
		packageData.ModulePath = pd.ModulePath()
	}
}

func (d *covDataVisitor) VisitFunc(pkgIdx uint32, fnIdx uint32, fd *FuncDesc) {
	var counters []uint32
	key := pkfunc{pk: pkgIdx, fcn: fnIdx}
	v, haveCounters := d.mm[key]

	if haveCounters {
		counters = v.Counters
	}

	fnData := &Func{
		Name:    fd.Funcname,
		SrcFile: fd.Srcfile,
		Units:   make([]*FuncUnit, len(fd.Units)),
	}

	podData := d.data.PodData[d.podHash]
	packageData := podData.Packages[pkgIdx]
	packageData.Funcs[fnIdx] = fnData

	for i := 0; i < len(fd.Units); i++ {
		u := fd.Units[i]
		var count uint32
		if counters != nil {
			count = counters[i]
		}

		fnData.Units[i] = &FuncUnit{
			StLine:  u.StLine,
			EnLine:  u.EnLine,
			StCol:   u.StCol,
			EnCol:   u.EnCol,
			NxStmts: u.NxStmts,
			Count:   count,
		}
	}
}
