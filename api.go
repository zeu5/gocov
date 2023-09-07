package gocov

import (
	"bytes"
	"runtime/coverage"

	"golang.org/x/tools/cover"
)

type Coverage struct {
	config CoverageConfig
	data   *CoverageData
}

type CoverageConfig struct {
	UseDir    string
	MatchPkgs []string
}

func GetCoverage(c CoverageConfig) (*Coverage, error) {
	if c.UseDir != "" {
		if err := coverage.WriteMetaDir(c.UseDir); err != nil {
			return nil, err
		}
		if err := coverage.WriteCountersDir(c.UseDir); err != nil {
			return nil, err
		}

		data, err := ReadDir(c.UseDir, c.MatchPkgs)
		if err != nil {
			return nil, err
		}
		return &Coverage{
			config: c,
			data:   data,
		}, nil
	} else {
		var rawCounters bytes.Buffer
		var rawMetadata bytes.Buffer

		if err := coverage.WriteMeta(&rawMetadata); err != nil {
			return nil, err
		}

		if err := coverage.WriteCounters(&rawCounters); err != nil {
			return nil, err
		}
		data, err := ReadFromBuffer(&rawMetadata, &rawCounters, c.MatchPkgs)
		if err != nil {
			return nil, err
		}

		return &Coverage{
			config: c,
			data:   data,
		}, nil
	}
}

func (c *Coverage) GetProfiles() []cover.Profile {

	fileProfiles := make(map[string]cover.Profile)
	for _, pod := range c.data.PodData {
		for _, pack := range pod.Packages {
			for _, fn := range pack.Funcs {
				if _, ok := fileProfiles[fn.SrcFile]; !ok {
					fileProfiles[fn.SrcFile] = cover.Profile{
						FileName: fn.SrcFile,
						Mode:     pod.CounterMode.String(),
						Blocks:   make([]cover.ProfileBlock, 0),
					}
				}
				profile := fileProfiles[fn.SrcFile]

				for _, u := range fn.Units {
					profile.Blocks = append(profile.Blocks, cover.ProfileBlock{
						StartLine: int(u.StLine),
						StartCol:  int(u.StCol),
						EndLine:   int(u.EnLine),
						EndCol:    int(u.EnCol),
						NumStmt:   int(u.NxStmts),
						Count:     int(u.Count),
					})
				}
			}
		}
	}

	out := make([]cover.Profile, len(fileProfiles))
	i := 0
	for _, p := range fileProfiles {
		out[i] = p
		i++
	}

	return out
}
