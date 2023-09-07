// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocov

import (
	"regexp"
	"strings"
)

// MatchSimplePattern returns a function that can be used to check
// whether a given name matches a pattern, where pattern is a limited
// glob pattern in which '...' means 'any string', with no other
// special syntax. There is one special case for MatchPatternSimple:
// according to the rules in "go help packages": a /... at the end of
// the pattern can match an empty string, so that net/... matches both
// net and packages in its subdirectories, like net/http.
func MatchSimplePattern(pattern string, toMatch string) bool {
	// Convert pattern to regular expression.
	// The strategy for the trailing /... is to nest it in an explicit ? expression.
	// The strategy for the vendor exclusion is to change the unmatchable
	// vendor strings to a disallowed code point (vendorChar) and to use
	// "(anything but that codepoint)*" as the implementation of the ... wildcard.
	// This is a bit complicated but the obvious alternative,
	// namely a hand-written search like in most shell glob matchers,
	// is too easy to make accidentally exponential.
	// Using package regexp guarantees linear-time matching.

	const vendorChar = "\x00"

	if strings.Contains(pattern, vendorChar) {
		return false
	}

	re := regexp.QuoteMeta(pattern)
	wild := `.*`
	if strings.HasSuffix(re, `/\.\.\.`) {
		re = strings.TrimSuffix(re, `/\.\.\.`) + `(/\.\.\.)?`
	}
	re = strings.ReplaceAll(re, `\.\.\.`, wild)

	reg := regexp.MustCompile(`^` + re + `$`)

	return reg.MatchString(toMatch)
}
