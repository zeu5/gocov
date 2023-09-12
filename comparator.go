package gocov

type funit struct {
	stline uint32
	enline uint32
}

// Return the number of new lines covered by the second argument over the first
func DiffLines(one, two *CoverageData) int {
	unitMap := make(map[funit]bool)
	for _, p := range one.PodData {
		for _, pa := range p.Packages {
			for _, f := range pa.Funcs {
				for _, u := range f.Units {
					unitMap[funit{u.StLine, u.EnLine}] = true
				}
			}
		}
	}

	new := 0
	for _, p := range two.PodData {
		for _, pa := range p.Packages {
			for _, f := range pa.Funcs {
				for _, u := range f.Units {
					unit := funit{u.StLine, u.EnLine}
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
