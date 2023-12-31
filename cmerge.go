// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocov

import (
	"fmt"
	"math"
)

// merger provides state and methods to help manage the process of
// merging together coverage counter data for a given function, for
// tools that need to implicitly merge counter as they read multiple
// coverage counter data files.
type merger struct {
	cmode    counterMode
	cgran    CounterGranularity
	overflow bool
}

// MergeCounters takes the counter values in 'src' and merges them
// into 'dst' according to the correct counter mode.
func (m *merger) MergeCounters(dst, src []uint32) (error, bool) {
	if len(src) != len(dst) {
		return fmt.Errorf("merging counters: len(dst)=%d len(src)=%d", len(dst), len(src)), false
	}
	if m.cmode == CtrModeSet {
		for i := 0; i < len(src); i++ {
			if src[i] != 0 {
				dst[i] = 1
			}
		}
	} else {
		for i := 0; i < len(src); i++ {
			dst[i] = m.SaturatingAdd(dst[i], src[i])
		}
	}
	ovf := m.overflow
	m.overflow = false
	return nil, ovf
}

// Saturating add does a saturating addition of 'dst' and 'src',
// returning added value or math.MaxUint32 if there is an overflow.
// Overflows are recorded in case the client needs to track them.
func (m *merger) SaturatingAdd(dst, src uint32) uint32 {
	result, overflow := saturatingAdd(dst, src)
	if overflow {
		m.overflow = true
	}
	return result
}

// Saturating add does a saturing addition of 'dst' and 'src',
// returning added value or math.MaxUint32 plus an overflow flag.
func saturatingAdd(dst, src uint32) (uint32, bool) {
	d, s := uint64(dst), uint64(src)
	sum := d + s
	overflow := false
	if uint64(uint32(sum)) != sum {
		overflow = true
		sum = math.MaxUint32
	}
	return uint32(sum), overflow
}

// SetModeAndGranularity records the counter mode and granularity for
// the current merge. In the specific case of merging across coverage
// data files from different binaries, where we're combining data from
// more than one meta-data file, we need to check for mode/granularity
// clashes.
func (cm *merger) SetModeAndGranularity(cmode counterMode, cgran CounterGranularity) error {
	// Collect counter mode and granularity so as to detect clashes.
	if cm.cmode != CtrModeInvalid {
		if cm.cmode != cmode {
			return fmt.Errorf("counter mode clash while reading meta-data file, previous file had %s, new file has %s", cm.cmode.String(), cmode.String())
		}
		if cm.cgran != cgran {
			return fmt.Errorf("counter granularity clash while reading meta-data file, previous file had %s, new file has %s", cm.cgran.String(), cgran.String())
		}
	}
	cm.cmode = cmode
	cm.cgran = cgran
	return nil
}

func (cm *merger) ResetModeAndGranularity() {
	cm.cmode = CtrModeInvalid
	cm.cgran = CtrGranularityInvalid
	cm.overflow = false
}

func (cm *merger) Mode() counterMode {
	return cm.cmode
}

func (cm *merger) Granularity() CounterGranularity {
	return cm.cgran
}
