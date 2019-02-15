package statsd_relay

import (
	// "log"
	"regexp"
)

type WhitelistFilter struct {
	Pattern string
	Regexp  *regexp.Regexp
}

func (f *WhitelistFilter) Init() {
	var err error
	f.Regexp, err = regexp.Compile(f.Pattern)
	ExitOnError(err)
}

func (f *WhitelistFilter) Filter(b []byte) bool {
	// log.Printf("WhitelistFilter.Filter: %s\n", string(b))

	matched := f.Regexp.Match(b)
	// log.Printf("WhitelistFilter.Filter matched: %v\n", matched)
	return matched
}
