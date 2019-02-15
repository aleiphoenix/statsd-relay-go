package statsd_relay

import (
	"log"
	"regexp"
)

// 占位符的文档请参考
// https://golang.org/pkg/regexp/#Regexp.ReplaceAll
type RewriteFilter struct {
	Pattern string
	Regexp  *regexp.Regexp
	Replace string // 原始的替换字符串，可能使用的是PCRE风格的占位符

	CalibredReplace []byte // 校正后的替换，使用的是golang风格的占位符
}

func (f *RewriteFilter) Init() {
	var err error
	f.Regexp, err = regexp.Compile(f.Pattern)
	ExitOnError(err)

	// 将原来支持 \1 的占位符，自动转换为 golang 风格的 $1
	holderRegexp, err := regexp.Compile("\\\\([0-9]+)")
	ExitOnError(err)

	varRegexp, err := regexp.Compile("\\\\")
	ExitOnError(err)

	//
	cnt := 0
	match := holderRegexp.FindAll([]byte(f.Replace), -1)
	if match != nil {
		cnt = len(match)
	}

	log.Printf("cnt: %d\n", cnt)
	if cnt > 0 {
		f.CalibredReplace = []byte(f.Replace)
		for i := 0; i < cnt; i++ {
			f.CalibredReplace = repalceOnce(
				varRegexp, f.CalibredReplace, []byte("$"))
		}
	} else {
		f.CalibredReplace = []byte(f.Replace)
	}

}

// 返回 bool, []byte ，表示是否匹配，与结果
func (f *RewriteFilter) Filter(b []byte) (bool, []byte) {
	if !f.Regexp.Match(b) {
		return false, b
	}
	rv := f.Regexp.ReplaceAll(b, f.CalibredReplace)
	return true, rv
}

func repalceOnce(re *regexp.Regexp, src []byte, rep []byte) []byte {
	var replaced bool = false
	rv := re.ReplaceAllFunc(src, func(a []byte) []byte {
		if replaced {
			return a
		}
		replaced = true
		rv := re.ReplaceAll(a, rep)
		return rv
	})
	return rv
}
