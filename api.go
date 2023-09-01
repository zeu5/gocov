package gocov

import (
	"bytes"
	"runtime/coverage"

	"github.com/zeu5/gocov/parser"
	"golang.org/x/tools/cover"
)

type Coverage struct {
	config CoverageConfig
	data   *parser.CoverageData
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

		data, err := parser.ReadDir(c.UseDir, c.MatchPkgs)
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
		data, err := parser.ReadFromBuffer(&rawMetadata, &rawCounters, c.MatchPkgs)
		if err != nil {
			return nil, err
		}

		return &Coverage{
			config: c,
			data:   data,
		}, nil
	}
}

func (c *Coverage) GetProfiles() ([]*cover.Profile, error) {

	return []*cover.Profile{}, nil
}
