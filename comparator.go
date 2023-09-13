package gocov

type funit struct {
	stline uint32
	enline uint32
	stcol  uint32
	encol  uint32
	nstmts uint32
}

// Return the number of new lines covered by the second argument over the first
func DiffLines(one, two *CoverageData) int {
	unitMap := make(map[funit]bool)
	for _, p := range one.PodData {
		for _, pa := range p.Packages {
			for _, f := range pa.Funcs {
				for _, u := range f.Units {
					unitMap[funit{u.StLine, u.EnLine, u.StCol, u.EnCol, u.NxStmts}] = true
				}
			}
		}
	}

	new := 0
	for _, p := range two.PodData {
		for _, pa := range p.Packages {
			for _, f := range pa.Funcs {
				for _, u := range f.Units {
					unit := funit{u.StLine, u.EnLine, u.StCol, u.EnCol, u.NxStmts}
					if _, ok := unitMap[unit]; !ok {
						new += 1
						unitMap[unit] = true
					}
				}
			}
		}
	}
	return new
}

type mcount struct {
	cur uint32
	new uint32
	idx int
}

func (cur *CoverageData) Merge(other *CoverageData) {
	for pName, p := range other.PodData {
		if _, ok := cur.PodData[pName]; !ok {
			cur.PodData[pName] = p
			continue
		}
		for packName, pack := range p.Packages {
			if _, ok := cur.PodData[pName].Packages[packName]; !ok {
				cur.PodData[pName].Packages[packName] = pack
				continue
			}
			for fName, f := range pack.Funcs {
				if _, ok := cur.PodData[pName].Packages[packName].Funcs[fName]; !ok {
					cur.PodData[pName].Packages[packName].Funcs[fName] = f
					continue
				}
				curUnits := cur.PodData[pName].Packages[packName].Funcs[fName].Units
				unitMap := make(map[funit]*mcount)

				for _, u := range curUnits {
					uKey := funit{u.StLine, u.EnLine, u.StCol, u.EnCol, u.NxStmts}
					unitMap[uKey] = &mcount{cur: u.Count}
				}

				for _, u := range f.Units {
					uKey := funit{u.StLine, u.EnLine, u.StCol, u.EnCol, u.NxStmts}
					count, ok := unitMap[uKey]
					if !ok {
						unitMap[uKey] = &mcount{new: u.Count}
					} else {
						count.new = u.Count
					}
				}

				curCount := make([]uint32, len(unitMap))
				newCount := make([]uint32, len(unitMap))
				i := 0
				for _, c := range unitMap {
					curCount[i] = c.cur
					newCount[i] = c.new
					c.idx = i
					i += 1
				}

				m := &merger{}
				m.SetModeAndGranularity(p.CounterMode, p.CounterGranularity)
				m.MergeCounters(curCount, newCount)

				cur.PodData[pName].Packages[packName].Funcs[fName].Units = make([]*FuncUnit, len(unitMap))
				for key, count := range unitMap {
					cur.PodData[pName].Packages[packName].Funcs[fName].Units[count.idx] = &FuncUnit{
						StLine:  key.stline,
						StCol:   key.stcol,
						EnLine:  key.enline,
						EnCol:   key.encol,
						NxStmts: key.nstmts,
						Count:   curCount[count.idx],
					}
				}
			}
		}
	}
}
