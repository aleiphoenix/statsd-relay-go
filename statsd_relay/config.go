package statsd_relay

import (
	"bufio"
	yaml "gopkg.in/yaml.v2"
	"os"
)

// 定义 input 配置下的一个输入
type InputConfig struct {
	Type string `yaml:"type"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// 定义 output 配置下的一个输出
type OutputConfig struct {
	Type string `yaml:"type"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// 定义重写过滤器
type RewriteFilterConfig struct {
	Regexp  string `yaml:"regexp"`
	Replace string `yaml:"replace"`
}

// 定义过滤器配置
type FilterConfig struct {
	Rewrite   []RewriteFilterConfig `yaml:"rewrite"`
	Whitelist []string              `yaml:"whitelist"`
	Blacklist []string              `yaml:"blacklist"`
}

type Config struct {
	BlacklistReport bool           `yaml:"blacklist_report"`
	Input           []InputConfig  `yaml:"input"`
	Output          []OutputConfig `yaml:"output"`
	Filters         FilterConfig   `yaml:"filters"`
}

func LoadConfig(fn string) Config {
	rv := Config{}
	var err error
	f, err := os.Open(fn)
	ExitOnError(err)
	defer f.Close()
	in := bufio.NewReader(f)

	decoder := yaml.NewDecoder(in)
	decoder.Decode(&rv)
	return rv
}
