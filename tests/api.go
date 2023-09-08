package main

import (
	"fmt"
	"log"

	"github.com/zeu5/gocov"
)

func TestGetCoverage() error {
	c, err := gocov.GetCoverage(gocov.CoverageConfig{
		MatchPkgs: []string{},
	})
	if err != nil {
		return err
	}

	log.Default().Printf("Percent covered: %.1f%%", c.GetPercent())

	return nil
}

type testFunc func() error

func main() {
	tests := map[string]testFunc{
		"GetCoverage": TestGetCoverage,
	}

	for name, test := range tests {
		err := test()
		if err == nil {
			fmt.Printf("OK! %s\n", name)
		} else {
			fmt.Printf("FAIL! %s, %s", name, err)
		}
	}
}
