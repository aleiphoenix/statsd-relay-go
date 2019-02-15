package statsd_relay

import (
	// "log"
	"regexp"
)

type BlacklistFilter struct {
	Pattern string
	Regexp  *regexp.Regexp
}

func (f *BlacklistFilter) Init() {
	var err error
	f.Regexp, err = regexp.Compile(f.Pattern)
	ExitOnError(err)
}

func (f *BlacklistFilter) Filter(b []byte) bool {
	// log.Printf("BlacklistFilter.Filter: %s\n", string(b))
	matched := f.Regexp.Match(b)
	// log.Printf("BlacklistFilter.Filter matched: %v\n", matched)
	return matched
}
